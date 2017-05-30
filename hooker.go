package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cryptopay-dev/go-metrics"
	"github.com/getsentry/raven-go"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Getwd() error: %s\n", err)
	}

	interval := flag.Int("interval", 60, "Time in seconds to sleep between checks")
	dir := flag.String("dir", cwd, "Directory we should look for a new files")
	out := flag.String("out", cwd, "Directory we should place zip files into")
	separator := flag.String("sep", ",", "Pattern separator")
	patterns := flag.String("patterns", ".xml, .xlsx", fmt.Sprintf("Patterns we look files in directory (seperated by: %s)", *separator))
	timeout := flag.Int("timeout", 180, "Timeout waiting request from API")
	verbose := flag.Bool("v", false, "Verbose output")
	checkInterval := flag.Int("check", 180, "Interval in seconds of file check")
	url := flag.String("url", "http://localhost:3000/", "URL of reports API")
	token := flag.String("token", "", "Auth token for API")
	zipFile := flag.Bool("zip", true, "Zip file")
	clear := flag.Bool("clear", true, "Clear file after send")
	listen := flag.String("listen", ":8080", "Server listen address")

	flag.Parse()

	// Printing header
	fmt.Println(art)

	// Setting options
	opts := options{
		interval:      *interval,
		dir:           *dir,
		out:           *out,
		patterns:      *patterns,
		timeout:       *timeout,
		verbose:       *verbose,
		checkInterval: *checkInterval,
		url:           *url,
		token:         *token,
		zip:           *zipFile,
		clear:         *clear,
		separator:     *separator,
		listen:        *listen,
	}

	sentry := os.Getenv("SENTRY_DSN")
	if sentry != "" {
		raven.SetDSN(sentry)
		raven.SetTagsContext(map[string]string{
			"dir":     opts.dir,
			"pattern": opts.patterns,
			"url":     opts.url,
		})
	} else {
		fmt.Println("** WARNING: You currently have disabled Sentry **")
	}

	if opts.token == "" {
		fmt.Println("** WARNING: You providen empty token! **")
	}

	// Enable metrics
	if err := metrics.Setup(os.Getenv("METRICS_URL"), os.Getenv("METRICS_APPLICATION"), os.Getenv("METRICS_HOSTNAME")); err == nil {
		go metrics.Watch(time.Second * 10)
	} else {
		log.Fatalf("Metrcis setup error: %s\n", err.Error())
	}

	fmt.Println("====================================================================")
	fmt.Println("Configuration:")
	fmt.Printf("  Interval:\t%d seconds\n", opts.interval)
	fmt.Printf("  Timeout:\t%d seconds\n", opts.timeout)
	fmt.Printf("  XML Check:\t%d seconds\n", opts.checkInterval)
	fmt.Printf("  Directory:\t%s\n", opts.dir)
	fmt.Printf("  Zip dir:\t%s\n", opts.out)
	fmt.Printf("  Patterns:\t%s (separator: %s)\n", opts.patterns, opts.separator)
	fmt.Printf("  URL:\t\t%s, Token:%s\n", opts.url, opts.token)
	fmt.Printf("  Clear:\t%t\n", opts.clear)
	fmt.Printf("  Zip:\t\t%t\n", opts.zip)
	fmt.Printf("  Verbose:\t%t\n", opts.verbose)
	fmt.Printf("  Listen:\t%s\n", opts.listen)
	fmt.Println("====================================================================")

	c := newController(opts)
	go c.watch()
	go c.serve()

	for {
		if opts.verbose {
			log.Println("Scanning directory for a new files")
		}

		files, err := ioutil.ReadDir(opts.dir)
		if err != nil {
			raven.CaptureErrorAndWait(err, map[string]string{
				"directory": opts.dir,
			})

			log.Fatalf("Directory traverse error: %s\n", err)
		}
		c.setDirectoryListing(files)

		if len(files) > 0 {
			for _, file := range files {
				// Skip if this is directory
				if file.IsDir() {
					if opts.verbose {
						log.Printf("Path %s is directory skipping\n", file.Name())
					}

					continue
				}

				// Skip if file has wrong suffix
				goodFile := false
				for _, suffix := range strings.Split(opts.patterns, opts.separator) {
					if strings.HasSuffix(file.Name(), strings.TrimSpace(suffix)) {
						goodFile = true
						break
					}
				}

				if !goodFile {
					if opts.verbose {
						metrics.SendAndWait("files", metrics.M{
							"skipped": true,
						}, nil)
						log.Printf("File %s is not accepted by system\n", file.Name())
					}
					continue
				}

				c.spawn(file)
			}
		}

		if opts.verbose {
			log.Printf("Sleeping for a %d sec\n", opts.interval)
		}

		time.Sleep(time.Second * time.Duration(opts.interval))
	}

}
