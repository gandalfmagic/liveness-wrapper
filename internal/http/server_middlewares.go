package http

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					logger.ErrorWithStack("internal server error:", debug.Stack(), err)
				}
			}()

			start := time.Now()
			wrapped := wrapResponseWriter(w)
			next.ServeHTTP(wrapped, r)
			logger.HttpDebugWithDuration(r, wrapped.status, time.Since(start))
		}

		return http.HandlerFunc(fn)
	}
}

func inStringSlice(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func MethodsMiddleware(methods []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if inStringSlice(methods, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			writeToResponse("methods-middleware", http.StatusMethodNotAllowed, w)
		}

		return http.HandlerFunc(fn)
	}
}
