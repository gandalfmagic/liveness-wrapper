package logger

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const DefaultLogLevel = "INFO"

type Logger struct {
	logger *zap.SugaredLogger
}

func NewLogger(out io.Writer, prefix, level string) (*Logger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02T15:04:05-0700"))
	}
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.CallerKey = ""
	encoderConfig.StacktraceKey = ""

	encoder := &prependEncoder{
		Encoder: zapcore.NewConsoleEncoder(encoderConfig),
		pool:    buffer.NewPool(),
		prefix:  prefix,
	}

	var l zapcore.Level
	if err := l.Set(level); err != nil {
		return nil, err
	}

	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(zapcore.AddSync(out)), zap.NewAtomicLevelAt(l))
	log := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.ErrorLevel))

	return &Logger{logger: log.Sugar()}, nil
}

func (l Logger) Close() {
	_ = l.logger.Sync()
}

func (l Logger) CheckFatal(message string, err error) {
	if err != nil {
		l.Fatalf("%s: %s", message, err)
	}
}

func (l Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatalf(format, v...)
	os.Exit(1)
}

func (l Logger) Errorf(format string, v ...interface{}) {
	l.logger.Errorf(format, v...)
}

func (l Logger) Warnf(format string, v ...interface{}) {
	l.logger.Warnf(format, v...)
}

func (l Logger) Infof(format string, v ...interface{}) {
	l.logger.Infof(format, v...)
}

func (l Logger) Debugf(format string, v ...interface{}) {
	l.logger.Debugf(format, v...)
}

func (l Logger) HTTPError(r *http.Request, status int) {
	l.logger.Errorf("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func (l Logger) HTTPWarn(r *http.Request, status int) {
	l.logger.Warnf("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func (l Logger) HTTPInfo(r *http.Request, status int) {
	l.logger.Infof("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func (l Logger) HTTPDebug(r *http.Request, status int) {
	l.logger.Debugf("%s %s \"%s\" %d \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent())
}

func (l Logger) HTTPDebugWithDuration(r *http.Request, status int, duration time.Duration) {
	l.logger.Debugf("%s %s \"%s\" %d \"%s\" \"%s\" %s", r.RemoteAddr, r.Method, r.RequestURI, status, r.Referer(), r.UserAgent(), duration)
}

type logInfoWriter struct {
	*Logger
	prefix string
}

func NewLogInfoWriter(prefix string, logger *Logger) io.Writer {
	lw := &logInfoWriter{Logger: logger, prefix: prefix}
	return lw
}

func (lw logInfoWriter) Write(p []byte) (n int, err error) {
	lw.logger.Infof("%s: %s", lw.prefix, string(p))

	return len(p), nil
}

type logErrorWriter struct {
	*Logger
	prefix string
}

func NewLogErrorWriter(prefix string, logger *Logger) io.Writer {
	lw := &logErrorWriter{Logger: logger, prefix: prefix}
	return lw
}

func (lw logErrorWriter) Write(p []byte) (n int, err error) {
	lw.logger.Errorf("%s: %s", lw.prefix, string(p))

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

type prependEncoder struct {
	// embed a zapcore encoder
	// this makes prependEncoder implement the interface without extra work
	zapcore.Encoder

	// zap buffer pool
	pool buffer.Pool

	prefix string
}

// implementing only EncodeEntry
func (e *prependEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// new log buffer
	buf := e.pool.Get()

	// prepend the JournalD prefix based on the entry level
	entry.LoggerName = e.toNamePrefix(e.prefix)

	// calling the embedded encoder's EncodeEntry to keep the original encoding format
	consoleBuf, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	// just write the output into your own buffer
	_, err = buf.Write(consoleBuf.Bytes())
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// some mapper function
func (e *prependEncoder) toNamePrefix(prefix string) string {
	if prefix != "" {
		prefix = fmt.Sprintf("[%s]", prefix)
	} else {
		prefix = ""
	}

	return prefix
}
