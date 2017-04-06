package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/xml"
)

var verbose *bool
var checkInterval *int

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Getwd() error: %s\n", err)
	}

	interval := flag.Int("interval", 60, "Time in seconds to sleep between checks")
	dir := flag.String("dir", cwd, "Directory we should look for a new files")
	pattern := flag.String("pattern", ".xml", "Pattern we look files in directory")
	verbose = flag.Bool("v", false, "Verbose output")
	checkInterval = flag.Int("check", 180, "Interval in seconds of file check")
	url := flag.String("url", "http://localhost:3000/", "URL of reports API")
	token := flag.String("token", "", "Auth token for API")
	clear := flag.Bool("clear", true, "Clear file after send")
	flag.Parse()

	// Printing header
	for _, s := range strings.Split(art, "\n") {
		fmt.Println(s)
	}

	fmt.Println("====================================================================")
	fmt.Println("Configuration:")
	fmt.Printf("  Interval:\t%d seconds\n", *interval)
	fmt.Printf("  Size Check:\t%d minutes\n", *checkInterval)
	fmt.Printf("  Directory:\t%s\n", *dir)
	fmt.Printf("  Pattern:\t%s\n", *pattern)
	fmt.Printf("  URL:\t\t%s, Token:%s\n", *url, *token)
	fmt.Printf("  Clear:\t%t\n", *clear)
	fmt.Printf("  Verbose:\t%t\n", *verbose)
	fmt.Println("====================================================================")

	for {
		if *verbose {
			log.Println("Scanning directory for a new files")
		}

		files, err := ioutil.ReadDir(*dir)
		if err != nil {
			log.Fatalf("Directory traverse error: %s\n", err)
		}

		if len(files) > 0 {
			for _, file := range files {
				filePath := path.Join(*dir, file.Name())

				// Skip if this is directory
				if file.IsDir() {
					if *verbose {
						log.Printf("Path %s is directory skipping\n", filePath)
					}

					continue
				}

				// Skip if file has wrong suffix
				if !strings.HasSuffix(filePath, *pattern) {
					if *verbose {
						log.Printf("File %s is not accepted by system\n", filePath)
					}

					continue
				}

				log.Printf("Found new file %s\n", filePath)

				// Checking that file have good size
				err = uploaded(filePath)
				if err != nil {
					log.Fatalf("File size checking error: %s\n", err)
				}

				// Sending stuff and deleting file
				buf, err := ioutil.ReadFile(filePath)
				if err != nil {
					log.Fatalf("Reading file error: %s\n", err)
				}

				err = send(*url, *token, buf, file.Name())
				if err != nil {
					log.Fatalf("[Error] sending to API: %s\n", err)
				}

				log.Println("Successfully send data to API")

				// Deleting file
				err = os.Remove(filePath)
				if err != nil {
					log.Fatalf("[Error] deleting file: %s\n", err)
				}
			}
		}

		if *verbose {
			log.Printf("Sleeping for a %d sec\n", *interval)
		}

		time.Sleep(time.Second * time.Duration(*interval))
	}
}

func uploaded(filePath string) error {
	var size int64

	for {
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}

		info, err := file.Stat()
		if err != nil {
			return err
		}

		tmp := info.Size()
		if tmp == size {
			return nil
		}

		if *verbose {
			log.Printf("File %s size is %d bytes", filePath, tmp)
			log.Printf("Next file size check in %d minutes\n", *checkInterval)
		}

		size = tmp
		time.Sleep(time.Second * time.Duration(*checkInterval))
	}
}

func send(url, token string, info []byte, filename string) error {
	backoff := 0

	for {
		log.Printf("Sending data to API %d try\n", backoff+1)

		err := post(url, token, info, filename)
		if err == nil {
			return nil
		}

		backoff++
		mul := math.Pow(2, float64(backoff)) // 2 4 16 32 64
		log.Printf("[Error] sending to API: %s", err)
		if backoff > 5 {
			break
		}

		log.Printf("Backoff for %d mins\n", int64(mul))
		time.Sleep(time.Minute * time.Duration(mul))
	}

	return errors.New("Unable to send data to API")
}

func post(url, token string, data []byte, filename string) error {
	var header []byte
	var t string
	if len(data) < 200 {
		header = data[:]
	} else {
		header = data[0:200]
	}

	if bytes.Contains(header, []byte(`<Transactions>`)) {
		t = "transactions"
	} else if bytes.Contains(header, []byte(`<SCHEME ID="GPS">`)) {
		t = "cards"
	} else {
		return errors.New("Unknown type on request")
	}

	// Minification
	m := minify.New()
	m.AddFunc("xml", xml.Minify)

	minified, err := m.Bytes("xml", data)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Name = filename
	gz.Comment = t
	n, err := gz.Write(minified)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("Written 0 bytes")
	}
	gz.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if req != nil {
		defer req.Body.Close()
	}

	if err != nil {
		return err
	}

	req.Header.Set("X-Access-Token", token)
	req.Header.Set("Content-Encoding", "gzip")

	client := &http.Client{}
	response, err := client.Do(req)
	if response != nil {
		defer response.Body.Close()
	}

	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Http status: %d", response.StatusCode)
	}

	return nil
}
