package http

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
)

type Server interface {
	Start() (chan<- bool, <-chan struct{})
}

type server struct {
	ctx           context.Context
	externalAlive chan bool
	isAlive       bool
	isReady       bool
	pingChannel   chan bool
	pingInterval  time.Duration
	updateReady   chan bool
	server        *http.Server
}

func NewServer(ctx context.Context, addr string, pingInterval time.Duration) Server {

	s := &server{
		ctx:           ctx,
		externalAlive: make(chan bool),
		pingChannel:   make(chan bool),
		pingInterval:  pingInterval,
		updateReady:   make(chan bool),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/ready", func(wr http.ResponseWriter, r *http.Request) {

		wr.Header().Set("Content-Type", "text/plain")

		if s.isReady {
			wr.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(wr, "Ready")
			logger.HttpDebug(r, http.StatusOK)
		} else {
			wr.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(wr, "Not ready")
			logger.HttpDebug(r, http.StatusServiceUnavailable)
		}
	})

	mux.HandleFunc("/alive", func(wr http.ResponseWriter, r *http.Request) {

		wr.Header().Set("Content-Type", "text/plain")

		if s.isAlive {
			wr.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(wr, "Service available")
			logger.HttpDebug(r, http.StatusOK)
		} else {
			wr.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(wr, "Service down")
			logger.HttpDebug(r, http.StatusServiceUnavailable)
		}
	})

	mux.HandleFunc("/ping", func(wr http.ResponseWriter, r *http.Request) {

		s.pingChannel <- true

		wr.Header().Set("Content-Type", "text/plain")
		wr.WriteHeader(http.StatusOK)

		_, _ = io.WriteString(wr, "Pong")

		logger.HttpDebug(r, http.StatusOK)
	})

	mux.HandleFunc("/", func(wr http.ResponseWriter, r *http.Request) {

		wr.Header().Set("Content-Type", "text/plain")
		wr.WriteHeader(http.StatusNotFound)

		_, _ = io.WriteString(wr, "Not Found")

		logger.HttpError(r, http.StatusNotFound)
	})

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return s
}

func (s *server) do() {
	defer close(s.pingChannel)
	defer close(s.updateReady)

	timer := time.NewTimer(s.pingInterval)

	isPingAlive := true
	isExternalAlive := true

	for {
		select {
		case <-s.ctx.Done():
			logger.Debug("http server context is closing")
			s.isReady = false
			timer.Stop()
			s.shutdown()
			return

		case isExternalAlive = <-s.externalAlive:
			s.isAlive = isExternalAlive && isPingAlive
			logger.Debug("alive status changed to %t", s.isAlive)

		case isPingAlive = <-s.pingChannel:
			if s.pingInterval == 0 {
				logger.Debug("timeout is %s, ignoring ping endpoint", s.pingInterval)
				isPingAlive = true
				s.isAlive = isExternalAlive && isPingAlive
				continue
			}

			s.isAlive = isExternalAlive && isPingAlive
			logger.Debug("alive status changed to %t", s.isAlive)

			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(s.pingInterval)
			logger.Debug("timer restarted")

		case s.isReady = <-s.updateReady:
			logger.Debug("ready status changed to %t", s.isReady)

		case <-timer.C:
			if s.pingInterval == 0 {
				logger.Debug("timeout is %s, the timeout is ignored", s.pingInterval)
				continue
			}

			isPingAlive = false
			s.isAlive = isExternalAlive && isPingAlive
			timer.Reset(s.pingInterval)
			logger.Debug("timer is expired, restarted with interval %s", s.pingInterval)
		}
	}
}

func (s *server) shutdown() {

	logger.Info("shutting down the http server...")

	ctxShutdown, shutdownCancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer shutdownCancel()

	s.server.SetKeepAlivesEnabled(false)
	if err := s.server.Shutdown(ctxShutdown); err != nil {
		logger.Fatal("could not shut down the http server: %s", err)
	}

	logger.Info("http server shutdown complete...")
}

func (s *server) Start() (chan<- bool, <-chan struct{}) {

	serverDone := make(chan struct{})

	go func() {

		defer close(serverDone)
		s.updateReady <- true
		logger.Info("starting http server on %s...", s.server.Addr)

		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("cannot bind http server on %s: %s", s.server.Addr, err)
		}
	}()

	go s.do()

	return s.externalAlive, serverDone
}
