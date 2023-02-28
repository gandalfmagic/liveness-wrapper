package http

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
)

func Test_server_do(t *testing.T) {
	zLogger, _ := logger.NewLogger(os.Stdout, "test", "info")

	t.Run("With_Timeout", func(t *testing.T) {
		// mock the server shutdown function
		// the mocked version closes a channel when it's called
		oldHttpServerShutdown := httpServerShutdown
		httpServerShutdown = func(context.Context, *http.Server, time.Duration, *logger.Logger) {}

		// create the context
		ctx, cancel := context.WithCancel(context.Background())

		s := &server{
			externalAlive: make(chan bool),
			pingChannel:   make(chan bool),
			pingInterval:  100 * time.Millisecond,
			updateReady:   make(chan bool),
			server:        nil,
			logger:        zLogger,
		}
		serverError := make(chan error)
		serverDone := make(chan struct{})
		go s.do(ctx, serverError, serverDone)

		if s.IsReady() != false {
			t.Errorf("isReady must be false: got %v", s.isReady)
		}

		if s.IsAlive() != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		// changing ready state to true
		s.updateReady <- true
		// waiting for the status of isReady to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsReady() != true {
			t.Errorf("isReady must be true: got %v", s.isReady)
		}

		// signaling that the external process is alive
		s.externalAlive <- true
		// waiting for the status of isAlive to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsAlive() != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		// sending a true value to the ping channel
		s.pingChannel <- true
		// waiting for the status of isAlive to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsAlive() != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		// let the timer expire
		time.Sleep(110 * time.Millisecond)

		if s.IsAlive() != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		// sending a true value to the ping channel
		s.pingChannel <- true
		// waiting for the status of isAlive to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsAlive() != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		// signaling that the external process is down
		s.externalAlive <- false
		// waiting for the status of isAlive to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsAlive() != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		// let the timer expire again
		time.Sleep(110 * time.Millisecond)

		if s.IsAlive() != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		// changing ready state to false
		s.updateReady <- false
		// waiting for the status of isReady to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsReady() != false {
			t.Errorf("isReady must be false: got %v", s.isReady)
		}

		// cancel the context, ending tests
		cancel()
		<-serverDone

		// Restore the default shutdown function
		httpServerShutdown = oldHttpServerShutdown
	})

	t.Run("Without_Timeout", func(t *testing.T) {
		// mock the server shutdown function
		// the mocked version closes a channel when it's called
		oldHttpServerShutdown := httpServerShutdown
		httpServerShutdown = func(context.Context, *http.Server, time.Duration, *logger.Logger) {}

		// create the context
		ctx, cancel := context.WithCancel(context.Background())

		s := &server{
			externalAlive: make(chan bool),
			pingChannel:   make(chan bool),
			pingInterval:  0,
			updateReady:   make(chan bool),
			server:        nil,
			logger:        zLogger,
		}
		serverError := make(chan error)
		serverDone := make(chan struct{})
		go s.do(ctx, serverError, serverDone)

		if s.IsReady() != false {
			t.Errorf("isReady must be false: got %v", s.isReady)
		}

		if s.IsAlive() != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		// changing ready state to true
		s.updateReady <- true
		// waiting for the status of isReady to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsReady() != true {
			t.Errorf("isReady must be true: got %v", s.isReady)
		}

		// signaling that the external process is alive
		s.externalAlive <- true
		// waiting for the status of isAlive to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsAlive() != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		// sending a true value to the ping channel
		s.pingChannel <- true
		// waiting for the status of isAlive to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsAlive() != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		// let the timer expire
		time.Sleep(110 * time.Millisecond)

		if s.IsAlive() != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		// sending a true value to the ping channel
		s.pingChannel <- true
		// waiting for the status of isAlive to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsAlive() != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		// let the timer expire again
		time.Sleep(110 * time.Millisecond)

		if s.IsAlive() != true {
			t.Errorf("isAlive must be true: got %v", s.isAlive)
		}

		// signaling that the external process is down
		s.externalAlive <- false
		// waiting for the status of isAlive to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsAlive() != false {
			t.Errorf("isAlive must be false: got %v", s.isAlive)
		}

		// changing ready state to false
		s.updateReady <- false
		// waiting for the status of isReady to be updated
		time.Sleep(1 * time.Millisecond)

		if s.IsReady() != false {
			t.Errorf("isReady must be false: got %v", s.isReady)
		}

		// cancel the context, ending tests
		cancel()
		<-serverDone

		// Restore the default shutdown function
		httpServerShutdown = oldHttpServerShutdown
	})
}

func TestServer(t *testing.T) {
	t.Run("Graceful_shutdown", func(t *testing.T) {
		zLogger, _ := logger.NewLogger(os.Stdout, "test", "error")
		ctx, cancel := context.WithCancel(context.Background())
		server := NewServer("127.0.0.1:6060", 15*time.Second, 100*time.Millisecond, zLogger)
		_, _, serverDone := server.Start(ctx)
		cancel()
		<-serverDone
	})
	t.Run("Port_conflict", func(t *testing.T) {
		zLogger, _ := logger.NewLogger(os.Stdout, "test", "error")
		ctx, cancel := context.WithCancel(context.Background())
		server := NewServer("127.0.0.1:6060", 15*time.Second, 100*time.Millisecond, zLogger)
		_, _, serverDone := server.Start(ctx)

		server2 := NewServer("127.0.0.1:6060", 15*time.Second, 200*time.Millisecond, zLogger)
		_, _, server2Done := server2.Start(ctx)
		<-server2Done

		cancel()
		<-serverDone
	})
}
