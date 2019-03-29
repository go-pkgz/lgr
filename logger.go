package lgr

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

var levels = []string{"DEBUG", "INFO", "WARN", "ERROR", "PANIC", "FATAL"}

// Logger provided simple logger with basic support of levels. Thread safe
type Logger struct {
	// set with Option calls
	stdout, stderr io.Writer // destination writes for out and err
	dbg            bool      // allows reporting for DEBUG level
	callerFile     bool      // reports caller file, i.e. /go/src/github.com/go-pkgz/lgr/logger.go
	callerFunc     bool      // reports caller function name, i.e. foo/bar.myFunc
	callerPkg      bool      // reports caller package name
	levelBraces    bool      // encloses level with [], i.e. [INFO]
	callerDepth    int       // how many stack frames to skip

	// internal use
	now   nowFn
	fatal panicFn
	msec  bool
	lock  sync.Mutex
}

// can be redefined internally for testing
type nowFn func() time.Time
type panicFn func()

// New makes new leveled logger. Accepts dbg flag turing on info about the caller and allowing DEBUG messages.
// Two writers can be passed optionally - first for out and second for err
func New(options ...Option) *Logger {
	res := Logger{
		now:         time.Now,
		fatal:       func() { os.Exit(1) },
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		callerDepth: 0,
	}
	for _, opt := range options {
		opt(&res)
	}
	return &res
}

// Logf implements L interface to output with printf style.
// Each line prefixed with ts, level and optionally (dbg mode only) by caller info.
// ERROR and FATAL also send the same line to err writer.
// FATAL adds runtime stack and os.exit(1), like panic.
func (l *Logger) Logf(format string, args ...interface{}) {
	// to align call depth between (*Logger).Logf() and, for example, Printf()
	l.logf(format, args...)
}

func (l *Logger) logf(format string, args ...interface{}) {

	// format timestamp with or without msecs
	ts := func() (res string) {
		if l.msec {
			return l.now().Format("2006/01/02 15:04:05.000")
		}
		return l.now().Format("2006/01/02 15:04:05")
	}

	lv, msg := l.extractLevel(fmt.Sprintf(format, args...))
	if lv == "DEBUG" && !l.dbg {
		return
	}
	var bld strings.Builder
	bld.WriteString(ts())
	bld.WriteString(l.formatLevel(lv))
	bld.WriteString(" ")

	if caller := l.reportCaller(l.callerDepth); caller != "" {
		bld.WriteString("{")
		bld.WriteString(caller)
		bld.WriteString("} ")
	}

	bld.WriteString(msg)

	if !strings.HasSuffix(msg, "\n") { // A newline is appended if the last character of s is not already a newline.
		bld.WriteString("\n")
	}

	l.lock.Lock()
	msgb := []byte(bld.String())
	_, _ = l.stdout.Write(msgb)

	switch lv {
	case "PANIC", "FATAL":
		_, _ = l.stderr.Write(msgb)
		_, _ = l.stderr.Write(getDump())
		l.fatal()
	case "ERROR":
		_, _ = l.stderr.Write(msgb) //nolint
	}

	l.lock.Unlock()
}

// calldepth 0 identifying the caller of reportCaller()
func (l *Logger) reportCaller(calldepth int) string {

	if !(l.callerFile || l.callerFunc || l.callerPkg) {
		return ""
	}

	// caller gets file, line number abd function name via runtime.Callers
	// file looks like /go/src/github.com/go-pkgz/lgr/logger.go
	// file is an empty string if not known.
	// funcName looks like:
	//   main.Test
	//   foo/bar.Test
	//   foo/bar.Test.func1
	//   foo/bar.(*Bar).Test
	//   foo/bar.glob..func1
	// funcName is an empty string if not known.
	// line is a zero if not known.
	caller := func(calldepth int) (file string, line int, funcName string) {
		pcs := make([]uintptr, 1)
		n := runtime.Callers(calldepth, pcs)
		if n != 1 {
			return "", 0, ""
		}

		frame, _ := runtime.CallersFrames(pcs).Next()

		return frame.File, frame.Line, frame.Function
	}

	// add 5 to adjust stack level because it was called from 3 nested functions added by lgr, i.e. caller, reportCaller and logf, plus 2 frames by runtime
	filePath, line, funcName := caller(calldepth + 2 + 3)
	if (filePath == "") || (line <= 0) || (funcName == "") {
		return "???"
	}

	// callerPkg only if no other callers
	if l.callerPkg && !l.callerFile && !l.callerFunc {
		_, pkgInfo := path.Split(path.Dir(filePath))
		return pkgInfo
	}

	res := ""

	if l.callerFile {
		fileInfo := filePath
		if pathElems := strings.Split(filePath, "/"); len(pathElems) > 2 {
			fileInfo = strings.Join(pathElems[len(pathElems)-2:], "/")
		}
		res += fmt.Sprintf("%s:%d", fileInfo, line)
		if l.callerFunc {
			res += " "
		}
	}

	if l.callerFunc {
		funcNameElems := strings.Split(funcName, "/")
		funcInfo := funcNameElems[len(funcNameElems)-1]
		res += funcInfo
	}

	return res
}

func (l *Logger) formatLevel(lv string) string {

	brace := func(b string) string {
		if l.levelBraces {
			return b
		}
		return ""
	}

	if lv == "" {
		return ""
	}

	spaces := ""
	if len(lv) == 4 {
		spaces = " "
	}
	return " " + brace("[") + lv + brace("]") + spaces
}

func (l *Logger) extractLevel(line string) (level, msg string) {
	for _, lv := range levels {
		if strings.HasPrefix(line, lv) {
			return lv, line[len(lv)+1:]
		}
		if strings.HasPrefix(line, "["+lv+"]") {
			return lv, line[len(lv)+3:]
		}
	}
	return "INFO", line
}

// getDump reads runtime stack and returns as a string
func getDump() []byte {
	maxSize := 5 * 1024 * 1024
	stacktrace := make([]byte, maxSize)
	length := runtime.Stack(stacktrace, true)
	if length > maxSize {
		length = maxSize
	}
	return stacktrace[:length]
}

// Option func type
type Option func(l *Logger)

// Out sets out writer
func Out(w io.Writer) Option {
	return func(l *Logger) {
		l.stdout = w
	}
}

// Err sets error writer
func Err(w io.Writer) Option {
	return func(l *Logger) {
		l.stderr = w
	}
}

// Debug turn on dbg mode
func Debug(l *Logger) {
	l.dbg = true
}

// CallerDepth sets number of stack frame skipped for caller reporting
func CallerDepth(n int) Option {
	return func(l *Logger) {
		l.callerDepth = n
	}
}

// CallerFunc adds caller info with function name
func CallerFunc(l *Logger) {
	l.callerFunc = true
}

// CallerPkg adds caller's package name
func CallerPkg(l *Logger) {
	l.callerPkg = true
}

// LevelBraces adds [] to level
func LevelBraces(l *Logger) {
	l.levelBraces = true
}

// CallerFile adds caller info with file, and line number
func CallerFile(l *Logger) {
	l.callerFile = true
}

// Msec adds .msec to timestamp
func Msec(l *Logger) {
	l.msec = true
}
