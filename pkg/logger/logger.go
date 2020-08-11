package logger

import (
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jcelliott/lumber"
)

const DefaultLogLevel = "INFO"

var mux sync.Mutex
var defaultLogger *lumber.ConsoleLogger

func CheckFatal(message string, err error) {
	mux.Lock()
	defer mux.Unlock()

	if err != nil {
		Fatalf(message+": ", err)
	}
}

func Fatalf(format string, v ...interface{}) {
	mux.Lock()
	getLogger().Fatal(format, v...)
	mux.Unlock()
	os.Exit(1)
}

func Errorf(format string, v ...interface{}) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Error(format, v...)
}

func Warnf(format string, v ...interface{}) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Warn(format, v...)
}

func Infof(format string, v ...interface{}) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Info(format, v...)
}

func Debugf(format string, v ...interface{}) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Debug(format, v...)
}

func HTTPError(r *http.Request, status int) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Error("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HTTPWarn(r *http.Request, status int) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Warn("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HTTPInfo(r *http.Request, status int) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Info("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HTTPDebug(r *http.Request, status int) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Debug("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HTTPDebugWithDuration(r *http.Request, status int, duration time.Duration) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Debug("%s %s \"%s\" %d \"%s\" \"%s\" %s", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent(), duration)
}

func Configure(out io.WriteCloser, prefix, level string) {
	mux.Lock()
	defer mux.Unlock()

	// convert the log level
	logLvl := lumber.LvlInt(level)

	if defaultLogger == nil {
		defaultLogger = lumber.NewBasicLogger(out, logLvl)
	} else {
		defaultLogger.Level(logLvl)
	}

	if prefix != "" {
		defaultLogger.Prefix("[" + prefix + "]")
	} else {
		defaultLogger.Prefix("")
	}

	defaultLogger.TimeFormat("2006-01-02T15:04:05-0700")
}

func New(out io.WriteCloser, prefix, level string) {
	mux.Lock()
	defer mux.Unlock()

	// convert the log level
	logLvl := lumber.LvlInt(level)

	defaultLogger = lumber.NewBasicLogger(out, logLvl)

	if prefix != "" {
		defaultLogger.Prefix("[" + prefix + "]")
	} else {
		defaultLogger.Prefix("")
	}

	defaultLogger.TimeFormat("2006-01-02T15:04:05-0700")
}

func getLogger() *lumber.ConsoleLogger {
	if defaultLogger == nil {
		defaultLogger = lumber.NewBasicLogger(os.Stdout, lumber.INFO)
		defaultLogger.TimeFormat("2006-01-02T15:04:05-0700")

		return defaultLogger
	}

	return defaultLogger
}

type logInfoWriter struct {
	prefix string
}

func NewLogInfoWriter(prefix string) io.Writer {
	lw := &logInfoWriter{prefix: prefix}
	return lw
}

func (lw logInfoWriter) Write(p []byte) (n int, err error) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Info(lw.prefix + ": " + string(p))

	return len(p), nil
}

type logErrorWriter struct {
	prefix string
}

func NewLogErrorWriter(prefix string) io.Writer {
	lw := &logErrorWriter{prefix: prefix}
	return lw
}

func (lw logErrorWriter) Write(p []byte) (n int, err error) {
	mux.Lock()
	defer mux.Unlock()
	getLogger().Error(lw.prefix + ": " + string(p))

	return len(p), nil
}

type WriterFunc func([]byte) (int, error)

func (w WriterFunc) Write(p []byte) (n int, err error) {
	return w(p)
}

func SignalOnWrite(signal chan<- int, wrapped io.Writer) io.Writer {
	return WriterFunc(func(p []byte) (n int, err error) {
		n, err = wrapped.Write(p)
		signal <- n
		return
	})
}
