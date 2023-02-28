package http

import (
	"io"
	"net/http"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
)

func writeToResponse(handler string, status int, w http.ResponseWriter, zLogger *logger.Logger) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)

	if _, err := io.WriteString(w, http.StatusText(status)); err != nil {
		zLogger.Errorf("cannot write response from %s handler: %s", handler, err)
		return
	}
}

func (s *server) ReadyHandler(w http.ResponseWriter, _ *http.Request) {
	status := http.StatusOK

	isReady := s.IsReady()
	if !isReady {
		status = http.StatusServiceUnavailable
	}

	writeToResponse("/ready", status, w, s.logger)
}

func (s *server) AliveHandler(w http.ResponseWriter, _ *http.Request) {
	status := http.StatusOK

	isAlive := s.IsAlive()
	if !isAlive {
		status = http.StatusServiceUnavailable
	}

	writeToResponse("/alive", status, w, s.logger)
}

func (s *server) PingHandler(w http.ResponseWriter, _ *http.Request) {
	s.pingChannel <- true

	writeToResponse("/ping", http.StatusOK, w, s.logger)
}

func (s *server) RootHandler(w http.ResponseWriter, _ *http.Request) {
	writeToResponse("/*", http.StatusNotFound, w, s.logger)
}
