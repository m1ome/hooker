package main

import (
	"net/http"
	"os"
	"sync"
	"time"

	"encoding/json"

	"github.com/cryptopay-dev/go-metrics"
)

type controller struct {
	mu      sync.Mutex
	files   map[string]chan struct{}
	dirlist []os.FileInfo
	options options
}

func newController(opts options) *controller {
	return &controller{
		files:   make(map[string]chan struct{}),
		options: opts,
	}
}

func (c *controller) watch() {
	for {
		c.mu.Lock()
		metrics.Send(metrics.M{
			"files_in_work": len(c.files),
		})
		c.mu.Unlock()

		time.Sleep(time.Second * 10)
	}
}

func (c *controller) filesInWork() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	files := []string{}
	for file := range c.files {
		files = append(files, file)
	}

	return files
}

func (c *controller) filesInDir() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	files := []string{}
	for _, file := range c.dirlist {
		files = append(files, file.Name())
	}

	return files
}

func (c *controller) setDirectoryListing(list []os.FileInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dirlist = list
}

func (c *controller) serve() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := json.Marshal(map[string]interface{}{
			"dir_files":     c.filesInDir(),
			"working_files": c.filesInWork(),
		})

		if err != nil {
			http.Error(w, "JSON marshalling error", http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	http.ListenAndServe(":9090", nil)
}

func (c *controller) spawn(file os.FileInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.files[file.Name()]; ok {
		return
	}

	ch := make(chan struct{})
	c.files[file.Name()] = ch
	parser := newParser(file, ch, c.options)
	go parser.parse()

	go func(ch chan struct{}, name string, cc *controller) {
		<-ch
		c.mu.Lock()
		delete(cc.files, name)
		c.mu.Unlock()
	}(ch, file.Name(), c)
}
