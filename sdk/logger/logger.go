// TODO: Implement a standardised logger that ensures flushing logs with traces in a structured way to datadog
package logger

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	WithContext(ctx context.Context) *logrus.Entry
	// WithField(key string, value interface{}) *logrus.Entry
	// WithFields(fields logrus.Fields) *logrus.Entry
	// WithError(err error) *logrus.Entry
	// WithTrace()

	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Printf(format string, args ...any)
	Warnf(format string, args ...any)
	Warningf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Panicf(format string, args ...any)

	Debug(args ...any)
	Info(args ...any)
	Print(args ...any)
	Warn(args ...any)
	Warning(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Panic(args ...any)

	Debugln(args ...any)
	Infoln(args ...any)
	Println(args ...any)
	Warnln(args ...any)
	Warningln(args ...any)
	Errorln(args ...any)
	Fatalln(args ...any)
	Panicln(args ...any)
	SetOutput(w io.Writer)
}

type StdLogger struct {
	*logrus.Logger
}

type logType struct {
	logType   string
	formatter logrus.Formatter
}

// Format implements logrus.Formatter.
func (l logType) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Data["log_type"] = l.logType
	return l.formatter.Format(entry)
}

func New(opts ...Option[LoggerOptions]) StdLogger {
	var opt LoggerOptions
	var writer io.Writer = os.Stdout
	for _, o := range opts {
		o(&opt)
	}
	if w, ok := opt.Writer(); ok {
		writer = w.writer
	}

	if opt.MinimalLogger() {
		return minimalLogger(writer)
	}
	return newStdLogger(writer)
}

func minimalLogger(w io.Writer) StdLogger {
	l := logrus.New()
	l.SetOutput(w)
	l.SetLevel(logrus.TraceLevel)
	return StdLogger{l}
}

func newStdLogger(w io.Writer) StdLogger {
	l := logrus.New()
	l.SetOutput(w)
	l.SetFormatter(logType{
		logType: "service",
		formatter: &logrus.JSONFormatter{
			DisableHTMLEscape: true,
		},
	})
	l.SetLevel(logrus.TraceLevel)
	l.SetReportCaller(true)
	l.SetFormatter(&logrus.JSONFormatter{})
	return StdLogger{l}
}
