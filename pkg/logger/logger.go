package logger

import (
	"net/http"
	"os"

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

func Configure(prefix, level string) {
	// convert the log level
	logLvl := lumber.LvlInt(level)

	// configure the logger
	lumber.Prefix("[" + prefix + "]")
	lumber.Level(logLvl)
	lumber.Debug("logger configured to console output")
}

type LogInfoWriter struct{}

func NewLogInfoWriter() *LogInfoWriter {
	lw := &LogInfoWriter{}
	return lw
}

func (lw LogInfoWriter) Write(p []byte) (n int, err error) {
	lumber.Info(string(p))
	return len(p), nil
}

type LogErrorWriter struct{}

func NewLogErrorWriter() *LogErrorWriter {
	lw := &LogErrorWriter{}
	return lw
}

func (lw LogErrorWriter) Write(p []byte) (n int, err error) {
	lumber.Error(string(p), nil)
	return len(p), nil
}
