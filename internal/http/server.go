package http

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
)

const (
	readTimeout  = 5 * time.Second
	writeTimeout = 10 * time.Second
	idleTimeout  = 15 * time.Second
)

type Server interface {
	Start(ctx context.Context) (chan<- bool, chan<- bool, <-chan struct{})
}

type server struct {
	externalAlive   chan bool
	isAlive         bool
	isReady         bool
	pingChannel     chan bool
	pingInterval    time.Duration
	server          *http.Server
	shutdownTimeout time.Duration
	updateReady     chan bool
	mux             sync.Mutex
}

var httpServerShutdown = func(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) {

	logger.Infof("shutting down the http server...")

	ctxShutdown, shutdownCancel := context.WithTimeout(ctx, shutdownTimeout)
	defer shutdownCancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctxShutdown); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatalf("could not shut down the http server: %s", err)
	}

	logger.Infof("http server shutdown complete...")
}

func NewServer(addr string, shutdownTimeout, pingInterval time.Duration) Server {
	s := &server{
		externalAlive:   make(chan bool),
		pingChannel:     make(chan bool),
		pingInterval:    pingInterval,
		shutdownTimeout: shutdownTimeout,
		updateReady:     make(chan bool),
	}

	mux := http.NewServeMux()
	mux.Handle("/ready", LoggingMiddleware()(MethodsMiddleware([]string{"GET"})(http.HandlerFunc(s.ReadyHandler))))
	mux.Handle("/alive", LoggingMiddleware()(MethodsMiddleware([]string{"GET"})(http.HandlerFunc(s.AliveHandler))))
	mux.Handle("/ping", LoggingMiddleware()(MethodsMiddleware([]string{"GET"})(http.HandlerFunc(s.PingHandler))))
	mux.Handle("/", LoggingMiddleware()(MethodsMiddleware([]string{"GET"})(http.HandlerFunc(RootHandler))))

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	return s
}

func (s *server) do(ctx context.Context, serverError chan error, serverDone chan struct{}) {
	defer close(serverDone)
	defer close(s.pingChannel)
	defer close(serverError)

	timer := time.NewTimer(s.pingInterval)

	isPingAlive := true
	isExternalAlive := false

	for {
		select {
		case <-serverError:
			s.setReady(false)

			_ = timer.Stop()

			return

		case <-ctx.Done():
			logger.Debugf("http server context is closing")
			s.setReady(false)

			_ = timer.Stop()

			httpServerShutdown(ctx, s.server, s.shutdownTimeout)

			return

		case isExternalAlive = <-s.externalAlive:
			s.setAlive(isExternalAlive && isPingAlive)
			logger.Debugf("alive status changed to %t", isExternalAlive && isPingAlive)

		case isPingAlive = <-s.pingChannel:
			if s.pingInterval == 0 {
				logger.Debugf("timeout is %s, ignoring ping endpoint", s.pingInterval)

				isPingAlive = true

				s.setAlive(isExternalAlive && isPingAlive)

				continue
			}

			s.setAlive(isExternalAlive && isPingAlive)
			logger.Debugf("alive status changed to %t", isExternalAlive && isPingAlive)

			if !timer.Stop() {
				<-timer.C
			}

			timer.Reset(s.pingInterval)
			logger.Debugf("timer restarted")

		case isReady := <-s.updateReady:
			s.setReady(isReady)
			logger.Debugf("ready status changed to %t", isReady)

		case <-timer.C:
			if s.pingInterval == 0 {
				logger.Debugf("timeout is %s, the timeout is ignored", s.pingInterval)
				continue
			}

			isPingAlive = false

			s.setAlive(isExternalAlive && isPingAlive)
			timer.Reset(s.pingInterval)
			logger.Debugf("timer is expired, restarted with interval %s", s.pingInterval)
		}
	}
}

func (s *server) Start(ctx context.Context) (chan<- bool, chan<- bool, <-chan struct{}) {
	isReady := make(chan struct{})
	serverDone := make(chan struct{})
	serverError := make(chan error)

	go s.do(ctx, serverError, serverDone)

	go func() {
		s.mux.Lock()
		addr := s.server.Addr
		s.mux.Unlock()

		logger.Infof("starting http server on %s...", addr)
		s.updateReady <- true

		close(isReady)

		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("cannot bind http server on %s: %s", addr, err)
			serverError <- err
		}
	}()

	// Make sure the main process is ready before returning
	<-isReady

	return s.updateReady, s.externalAlive, serverDone
}

func (s *server) setAlive(isAlive bool) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.isAlive = isAlive
}

func (s *server) IsAlive() bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	return s.isAlive
}

func (s *server) setReady(isReady bool) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.isReady = isReady
}

func (s *server) IsReady() bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	return s.isReady
}
