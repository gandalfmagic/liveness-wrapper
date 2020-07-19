package http

import (
	"io"
	"net/http"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
)

func writeToResponse(handler string, status int, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)

	if _, err := io.WriteString(w, http.StatusText(status)); err != nil {
		logger.Errorf("cannot write response from %s handler: %s", handler, err)
		return
	}
}

func (s *server) ReadyHandler(w http.ResponseWriter, _ *http.Request) {
	status := http.StatusOK

	isReady := s.IsReady()
	if !isReady {
		status = http.StatusServiceUnavailable
	}

	writeToResponse("/ready", status, w)
}

func (s *server) AliveHandler(w http.ResponseWriter, _ *http.Request) {
	status := http.StatusOK

	isAlive := s.IsAlive()
	if !isAlive {
		status = http.StatusServiceUnavailable
	}

	writeToResponse("/alive", status, w)
}

func (s *server) PingHandler(w http.ResponseWriter, _ *http.Request) {
	s.pingChannel <- true

	writeToResponse("/ping", http.StatusOK, w)
}

func RootHandler(w http.ResponseWriter, _ *http.Request) {
	writeToResponse("/*", http.StatusNotFound, w)
}
