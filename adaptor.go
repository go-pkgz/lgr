package lgr

import (
	"log"
	"strings"
)

// logWriter is an L wrapper for use as an output writer for the standard logger
type logWriter struct {
	l L
}

// Write implements io.Writer
func (w logWriter) Write(p []byte) (n int, err error) {
	w.l.Logf(strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}

// ToStdLogger returns l wrapped into the standard logger.
// level is an optional logging level.
// It is assumed that the returned logger will not be further adjusted
// (i.e. (*Logger).SetOutput() method will not be called).
func ToStdLogger(l L, level string) *log.Logger {
	if level != "" && !strings.HasSuffix(level, " ") {
		level += " "
	}
	return log.New(logWriter{l}, level, 0)
}
