package http

import (
	"context"
	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
	"net/http"
	"testing"
	"time"
)

func Test_server_do(t *testing.T) {
	t.Run("With_Timeout", func(t *testing.T) {
		// mock the server shutdown function
		// the mocked version closes a channel when it's called
		oldHttpServerShutdown := httpServerShutdown
		serverDone := make(chan struct{})
		httpServerShutdown = func(ctx context.Context, server *http.Server) {
			close(serverDone)
		}

		// create the context
		ctx, cancel := context.WithCancel(context.Background())

		s := &server{
			ctx:           ctx,
			externalAlive: make(chan bool),
			pingChannel:   make(chan bool),
			pingInterval:  100 * time.Millisecond,
			updateReady:   make(chan bool),
			server:        nil,
		}
		go s.do()

		if s.isReady != false {
			t.Errorf("isReady must be false: got %v", s.isReady)
		}

		if s.isAlive != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		t.Log("changing ready state to true")
		s.updateReady <- true
		t.Log("waiting for the status of isReady to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isReady != true {
			t.Errorf("isReady must be true: got %v", s.isReady)
		}

		t.Log("signaling that the external process is alive")
		s.externalAlive <- true
		t.Log("waiting for the status of isAlive to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isAlive != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		t.Log("sending a true value to the ping channel")
		s.pingChannel <- true
		t.Log("waiting for the status of isAlive to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isAlive != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		t.Log("let the timer expire")
		time.Sleep(110 * time.Millisecond)

		if s.isAlive != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		t.Log("sending a true value to the ping channel")
		s.pingChannel <- true
		t.Log("waiting for the status of isAlive to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isAlive != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		t.Log("signaling that the external process is down")
		s.externalAlive <- false
		t.Log("waiting for the status of isAlive to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isAlive != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		t.Log("let the timer expire again")
		time.Sleep(110 * time.Millisecond)

		if s.isAlive != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		t.Log("changing ready state to false")
		s.updateReady <- false
		t.Log("waiting for the status of isReady to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isReady != false {
			t.Errorf("isReady must be false: got %v", s.isReady)
		}

		t.Log("cancel the context, ending tests")
		cancel()
		<-serverDone

		// Restore the default shutdown function
		httpServerShutdown = oldHttpServerShutdown
	})

	t.Run("Without_Timeout", func(t *testing.T) {
		// mock the server shutdown function
		// the mocked version closes a channel when it's called
		oldHttpServerShutdown := httpServerShutdown
		serverDone := make(chan struct{})
		httpServerShutdown = func(ctx context.Context, server *http.Server) {
			close(serverDone)
		}

		// create the context
		ctx, cancel := context.WithCancel(context.Background())

		s := &server{
			ctx:           ctx,
			externalAlive: make(chan bool),
			pingChannel:   make(chan bool),
			pingInterval:  0,
			updateReady:   make(chan bool),
			server:        nil,
		}
		go s.do()

		if s.isReady != false {
			t.Errorf("isReady must be false: got %v", s.isReady)
		}

		if s.isAlive != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		t.Log("changing ready state to true")
		s.updateReady <- true
		t.Log("waiting for the status of isReady to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isReady != true {
			t.Errorf("isReady must be true: got %v", s.isReady)
		}

		t.Log("signaling that the external process is alive")
		s.externalAlive <- true
		t.Log("waiting for the status of isAlive to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isAlive != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		t.Log("sending a true value to the ping channel")
		s.pingChannel <- true
		t.Log("waiting for the status of isAlive to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isAlive != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		t.Log("let the timer expire")
		time.Sleep(110 * time.Millisecond)

		if s.isAlive != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		t.Log("sending a true value to the ping channel")
		s.pingChannel <- true
		t.Log("waiting for the status of isAlive to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isAlive != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		t.Log("let the timer expire again")
		time.Sleep(110 * time.Millisecond)

		if s.isAlive != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		t.Log("signaling that the external process is down")
		s.externalAlive <- false
		t.Log("waiting for the status of isAlive to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isAlive != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		t.Log("changing ready state to false")
		s.updateReady <- false
		t.Log("waiting for the status of isReady to be updated")
		time.Sleep(1 * time.Millisecond)

		if s.isReady != false {
			t.Errorf("isReady must be false: got %v", s.isReady)
		}

		t.Log("cancel the context, ending tests")
		cancel()
		<-serverDone

		// Restore the default shutdown function
		httpServerShutdown = oldHttpServerShutdown
	})
}

func TestServer(t *testing.T) {
	t.Run("Graceful_shutdown", func(t *testing.T) {
		logger.Configure("test", "ERROR")
		ctx, cancel := context.WithCancel(context.Background())
		server := NewServer(ctx, "127.0.0.1:6060", 100*time.Millisecond)
		_, serverDone := server.Start()
		cancel()
		<-serverDone
	})
	t.Run("Port_conflict", func(t *testing.T) {
		logger.Configure("test", "ERROR")
		ctx, cancel := context.WithCancel(context.Background())
		server := NewServer(ctx, "127.0.0.1:6060", 100*time.Millisecond)
		_, serverDone := server.Start()

		server2 := NewServer(ctx, "127.0.0.1:6060", 100*time.Millisecond)
		_, server2Done := server2.Start()
		<-server2Done

		cancel()
		<-serverDone
	})
}