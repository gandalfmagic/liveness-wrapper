package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
)

type Server interface {
	Start() (chan<- bool, chan<- bool, <-chan struct{})
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

var httpServerShutdown = func(ctx context.Context, server *http.Server) {

	logger.Info("shutting down the http server...")

	ctxShutdown, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Fatal("could not shut down the http server: %s", err)
	}

	logger.Info("http server shutdown complete...")
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
	mux.Handle("/ready", LoggingMiddleware()(MethodsMiddleware([]string{"GET"})(http.HandlerFunc(s.ReadyHandler))))
	mux.Handle("/alive", LoggingMiddleware()(MethodsMiddleware([]string{"GET"})(http.HandlerFunc(s.AliveHandler))))
	mux.Handle("/ping", LoggingMiddleware()(MethodsMiddleware([]string{"GET"})(http.HandlerFunc(s.PingHandler))))
	mux.Handle("/", LoggingMiddleware()(MethodsMiddleware([]string{"GET"})(http.HandlerFunc(RootHandler))))

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
	//defer close(s.updateReady)

	timer := time.NewTimer(s.pingInterval)

	isPingAlive := true
	isExternalAlive := false

	for {
		select {
		case <-s.ctx.Done():
			logger.Debug("http server context is closing")
			s.isReady = false
			timer.Stop()
			httpServerShutdown(s.ctx, s.server)
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

func (s *server) Start() (chan<- bool, chan<- bool, <-chan struct{}) {
	isReady := make(chan struct{})
	go s.do()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		logger.Info("starting http server on %s...", s.server.Addr)
		s.updateReady <- true
		close(isReady)

		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("cannot bind http server on %s: %s", s.server.Addr, err)
		}
	}()

	// Make sure the main process is ready before returning
	<-isReady

	return s.updateReady, s.externalAlive, serverDone
}
