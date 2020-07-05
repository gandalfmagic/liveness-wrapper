package logger

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jcelliott/lumber"
)

func CheckFatal(message string, err error) {
	if err != nil {
		Fatal(message+": ", err)
	}
}

func Fatal(format string, v ...interface{}) {
	lumber.Fatal(format, v...)
	os.Exit(1)
}

func ErrorWithStack(format string, stack []byte, v ...interface{}) {
	format = fmt.Sprintf("%s\n%s", format, stack)
	lumber.Error(format, v...)
}

func Error(format string, v ...interface{}) {
	lumber.Error(format, v...)
}

func Warn(format string, v ...interface{}) {
	lumber.Warn(format, v...)
}

func Info(format string, v ...interface{}) {
	lumber.Info(format, v...)
}

func Debug(format string, v ...interface{}) {
	lumber.Debug(format, v...)
}

func HttpError(r *http.Request, status int) {
	lumber.Error("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HttpWarn(r *http.Request, status int) {
	lumber.Warn("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HttpInfo(r *http.Request, status int) {
	lumber.Info("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HttpDebug(r *http.Request, status int) {
	lumber.Debug("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HttpDebugWithDuration(r *http.Request, status int, duration time.Duration) {
	lumber.Debug("%s %s \"%s\" %d \"%s\" \"%s\" %s", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent(), duration)
}

func Configure(prefix, level string) {
	// convert the log level
	logLvl := lumber.LvlInt(level)

	// configure the logger
	lumber.Prefix("[" + prefix + "]")
	lumber.Level(logLvl)
	lumber.Debug("logger configured to console output")
}

type logInfoWriter struct {
	prefix string
}

func NewLogInfoWriter(prefix string) io.Writer {
	lw := &logInfoWriter{prefix: prefix}
	return lw
}

func (lw logInfoWriter) Write(p []byte) (n int, err error) {
	lumber.Info(lw.prefix + ": " + string(p))
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
	lumber.Error(lw.prefix + ": " + string(p))
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
