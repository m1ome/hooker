package main

import (
	"os"
	"sync"
)

type controller struct {
	mu      sync.Mutex
	files   map[string]chan struct{}
	options options
}

func newController(opts options) *controller {
	return &controller{
		files:   make(map[string]chan struct{}),
		options: opts,
	}
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
