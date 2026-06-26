package logger

import "io"

type Option[T LoggerOptions] func(*T)

type LoggerOptions struct {
	writer        loggerOutputWriter
	minimalLogger bool
}

func SetWriter(w io.Writer) Option[LoggerOptions] {
	return func(o *LoggerOptions) {
		o.writer = loggerOutputWriter{w}
	}
}
func WithMinimal() Option[LoggerOptions] {
	return func(o *LoggerOptions) {
		o.minimalLogger = true
	}
}

type loggerOutputWriter struct{ writer io.Writer }

func (c LoggerOptions) Writer() (loggerOutputWriter, bool) {
	return c.writer, c.writer.writer != nil
}
func (c LoggerOptions) MinimalLogger() bool {
	return c.minimalLogger
}
