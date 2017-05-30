package metrics

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/go-nats"
)

type conn struct {
	mu          sync.RWMutex
	nats        *nats.Conn
	enabled     bool
	queue       string
	url         string
	hostname    string
	application string
}

// M metrics storage
// Example:
// m := metrics.M{
// 	"metric": 1000,
//	"gauge": 1,
// }
type M map[string]interface{}

// T tags storage
// Example:
// m := metrics.T{
//	"tag": "some_default_tag"
// }
type T map[string]string

// DefaultConn shared default metric
// connection
var DefaultConn *conn

// DefaultQueue is queue where we puts event into NATS
const DefaultQueue = "telegraf"

// Setup rewrites default metrics configuration
//
// Params:
// - url (in e.g. "nats://localhost:4222")
// - options nats.Option array
//
// Example:
// import (
//     "log"
//
//     "github.com/cryptopay.dev/go-metrics"
// )
//
// func main() {
//     err := metrics.Setup("nats://localhost:4222")
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     for i:=0; i<10; i++ {
//         err = metrics.SendAndWait(metrics.M{
//             "counter": i,
//         })
//
//         if err != nil {
//             log.Fatal(err)
//         }
//     }
// }
func Setup(url string, application, hostname string, options ...nats.Option) error {
	metrics, err := New(url, application, hostname, options...)
	if err != nil {
		return err
	}

	DefaultConn = metrics
	return nil
}

// New creates new metrics connection
//
// Params:
// - url (in e.g. "nats://localhost:4222")
// - options nats.Option array
//
// Example:
// import (
//     "log"
//
//     "github.com/cryptopay.dev/go-metrics"
// )
//
// func main() {
//     m, err := metrics.New("nats://localhost:4222")
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     for i:=0; i<10; i++ {
//         err = m.SendAndWait(metrics.M{
//             "counter": i,
//         })
//
//         if err != nil {
//             log.Fatal(err)
//         }
//     }
// }
func New(url string, application, hostname string, options ...nats.Option) (*conn, error) {
	if url == "" {
		return &conn{
			enabled: false,
		}, nil
	}

	// Getting current environment
	if application == "" {
		return nil, errors.New("Application name not set")
	}

	if hostname == "" {
		return nil, errors.New("Hostname not set")
	}

	nc, err := nats.Connect(url, options...)
	if err != nil {
		return nil, err
	}

	conn := &conn{
		nats:        nc,
		hostname:    hostname,
		enabled:     true,
		queue:       DefaultQueue,
		application: application,
	}

	return conn, nil
}

// Send metrics to NATS queue
//
// Example:
// m.Send(metrics.M{
// 		"counter": i,
// })
func Send(name string, metrics M, tags T) (err chan error) {
	if DefaultConn == nil {
		return nil
	}

	return DefaultConn.Send(name, metrics, tags)
}

// SendAndWait metrics to NATS queue waiting for response
//
// Example:
// err = m.SendAndWait(metrics.M{
// 		"counter": i,
// })
func SendAndWait(name string, metrics M, tags T) error {
	if DefaultConn == nil {
		return nil
	}

	return DefaultConn.SendAndWait(name, metrics, tags)
}

// Send metrics to NATS queue
//
// Example:
// m.Send(metrics.M{
// 		"counter": i,
// })
func (m *conn) Send(name string, metrics M, tags T) chan error {
	ch := make(chan error, 1)

	go func() {
		ch <- m.SendAndWait(name, metrics, tags)
	}()

	return ch
}

// SendAndWait metrics to NATS queue waiting for response
//
// Example:
// err = m.SendAndWait(metrics.M{
// 		"counter": i,
// })
func (m *conn) SendAndWait(name string, metrics M, tags T) error {
	m.mu.RLock()
	if !m.enabled {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	if len(metrics) == 0 {
		return nil
	}

	if tags == nil {
		tags = make(T)
	}

	m.mu.RLock()
	tags["hostname"] = m.hostname
	m.mu.RUnlock()

	metricName := []string{m.application, name}
	buf := format(strings.Join(metricName, ":"), metrics, tags)

	m.mu.RLock()
	queue := m.queue
	m.mu.RUnlock()

	return m.nats.Publish(queue, buf)
}

// Disable disables watcher and disconnects
func (m *conn) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = false
	m.nats.Close()
}

// Disable disables watcher and disconnects
func Disable() {
	if DefaultConn == nil {
		return
	}

	DefaultConn.mu.Lock()
	defer DefaultConn.mu.Unlock()

	DefaultConn.enabled = false
	DefaultConn.nats.Close()
}

// Watch watches memory, goroutine counter
func (m *conn) Watch(interval time.Duration) error {
	var mem runtime.MemStats

	for {
		m.mu.RLock()
		enabled := m.enabled
		m.mu.RUnlock()

		if !enabled {
			break
		}

		// Getting memory stats
		runtime.ReadMemStats(&mem)
		metric := M{
			"alloc":         mem.Alloc,
			"alloc_objects": mem.HeapObjects,
			"gorotines":     runtime.NumGoroutine(),
			"gc":            mem.LastGC,
			"next_gc":       mem.NextGC,
			"pause_ns":      mem.PauseNs[(mem.NumGC+255)%256],
		}
		err := m.SendAndWait("gostats", metric, nil)
		if err != nil {
			return err
		}

		time.Sleep(interval)
	}

	return nil
}

// Watch watches memory, goroutine counter
func Watch(interval time.Duration) error {
	if DefaultConn == nil {
		return nil
	}

	return DefaultConn.Watch(interval)
}

func format(name string, metrics M, tags T) []byte {
	buf := bytes.NewBufferString(name)

	if len(tags) > 0 {
		var tagKeys []string
		for k := range tags {
			tagKeys = append(tagKeys, k)
		}
		sort.Strings(tagKeys)

		for _, k := range tagKeys {
			buf.WriteRune(',')
			buf.WriteString(k)
			buf.WriteRune('=')
			buf.WriteString(tags[k])
		}
	}

	buf.WriteRune(' ')
	count := 0

	var metricKeys []string
	for k := range metrics {
		metricKeys = append(metricKeys, k)
	}
	sort.Strings(metricKeys)

	for _, k := range metricKeys {
		if count > 0 {
			buf.WriteRune(',')
		}
		buf.WriteString(k)
		buf.WriteRune('=')

		v := metrics[k]
		switch v.(type) {
		case string:
			buf.WriteRune('"')
			buf.WriteString(v.(string))
			buf.WriteRune('"')
		default:
			buf.WriteString(fmt.Sprintf("%v", v))
		}
		count++
	}

	return buf.Bytes()
}
