package logger

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jcelliott/lumber"
)

type consoleLogger *lumber.ConsoleLogger

var logger consoleLogger

func CheckFatal(message string, err error) {
	if err != nil {
		Fatal(message+": ", err)
	}
}

func Fatal(format string, v ...interface{}) {
	getLogger().Fatal(format, v...)
	os.Exit(1)
}

func Error(format string, v ...interface{}) {
	getLogger().Error(format, v...)
}

func Warn(format string, v ...interface{}) {
	getLogger().Warn(format, v...)
}

func Info(format string, v ...interface{}) {
	getLogger().Info(format, v...)
}

func Debug(format string, v ...interface{}) {
	getLogger().Debug(format, v...)
}

func HttpError(r *http.Request, status int) {
	getLogger().Error("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HttpWarn(r *http.Request, status int) {
	getLogger().Warn("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HttpInfo(r *http.Request, status int) {
	getLogger().Info("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HttpDebug(r *http.Request, status int) {
	getLogger().Debug("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func HttpDebugWithDuration(r *http.Request, status int, duration time.Duration) {
	getLogger().Debug("%s %s \"%s\" %d \"%s\" \"%s\" %s", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent(), duration)
}

func Configure(out io.WriteCloser, prefix, level string) {
	// convert the log level
	logLvl := lumber.LvlInt(level)

	if logger == nil {
		logger = lumber.NewBasicLogger(out, logLvl)
	}
	(*lumber.ConsoleLogger)(logger).Level(logLvl)
	(*lumber.ConsoleLogger)(logger).Prefix("[" + prefix + "]")
}

func getLogger() *lumber.ConsoleLogger {
	if logger == nil {
		logger = lumber.NewConsoleLogger(lumber.INFO)
		return logger
	}

	return logger
}

type logInfoWriter struct {
	prefix string
}

func NewLogInfoWriter(prefix string) io.Writer {
	lw := &logInfoWriter{prefix: prefix}
	return lw
}

func (lw logInfoWriter) Write(p []byte) (n int, err error) {
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
