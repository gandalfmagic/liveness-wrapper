package testconsole

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type TestConsole struct {
	lineIterator chan string
	mux          sync.RWMutex
	enabled      bool
}

var ErrTimeout = fmt.Errorf("timeout waiting for the output line")

func (c *TestConsole) Write(p []byte) (int, error) {
	c.mux.RLock()
	enabled := c.enabled
	c.mux.RUnlock()

	if enabled {
		c.lineIterator <- string(p)
	}

	fmt.Print(string(p))

	return len(p), nil
}

func (c *TestConsole) Close() error {
	return nil
}

func (c *TestConsole) WaitForText(line string, timeout time.Duration) <-chan error {
	c.mux.Lock()
	c.enabled = true
	c.mux.Unlock()

	ch := make(chan error)
	go c.run(line, ch, timeout)

	return ch
}

func (c *TestConsole) run(expectedLine string, ch chan<- error, timeout time.Duration) {
	timer := time.NewTimer(timeout)

	defer func() {
		timer.Stop()
		close(ch)
	}()

	if expectedLine == "" {
		ch <- nil

		return
	}

	for {
		select {
		case line := <-c.lineIterator:
			if strings.Contains(line, expectedLine) {
				ch <- nil

				c.mux.Lock()
				c.enabled = false
				c.mux.Unlock()

				return
			}
		case <-timer.C:
			ch <- ErrTimeout

			return
		}
	}
}

func NewTestConsole() *TestConsole {
	return &TestConsole{
		lineIterator: make(chan string),
	}
}
