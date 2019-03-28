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
	stdout, stderr io.Writer
	dbg            bool
	lock           sync.Mutex
	callerFile     bool
	callerFunc     bool
	callerPkg      bool
	ignoredCallers map[string]struct{}
	now            nowFn
	fatal          panicFn
	levelBraces    bool
	msec           bool
}

type nowFn func() time.Time
type panicFn func()

// New makes new leveled logger. Accepts dbg flag turing on info about the caller and allowing DEBUG messages/
// Two writers can be passed optionally - first for out and second for err
func New(options ...Option) *Logger {
	res := Logger{
		now:            time.Now,
		fatal:          func() { os.Exit(1) },
		stdout:         os.Stdout,
		stderr:         os.Stderr,
		ignoredCallers: make(map[string]struct{}),
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

	if caller := l.reportCaller(2); caller != "" {
		bld.WriteString("{")
		bld.WriteString(caller)
		bld.WriteString("} ")
	}

	bld.WriteString(msg)  //nolint
	bld.WriteString("\n") //nolint

	l.lock.Lock()
	msgb := []byte(bld.String())
	l.stdout.Write(msgb) //nolint

	switch lv {
	case "PANIC", "FATAL":
		l.stderr.Write(msgb)      //nolint
		bld.WriteString("\n")     //nolint
		l.stderr.Write(getDump()) //nolint
		l.fatal()
	case "ERROR":
		l.stderr.Write(msgb) //nolint
	}

	l.lock.Unlock()
}

// calldepth 0 identifying the caller of reportCaller()
func (l *Logger) reportCaller(calldepth int) string {
	if !(l.callerFile || l.callerFunc || l.callerPkg) {
		return ""
	}

	filePath, line, funcName, ok := l.caller(calldepth + 1)
	if !ok {
		return ""
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

// caller searches for the caller while skipping ignored functions.
// calldepth is a starting call depth for the search.
// calldepth 0 identifying the caller of caller()
// file looks like:
//   /go/src/github.com/go-pkgz/lgr/logger.go
// funcName looks like:
//   main.Test
//   foo/bar.Test
//   foo/bar.Test.func1
//   foo/bar.(*Bar).Test
//   foo/bar.glob..func1
// line is always > 0
func (l *Logger) caller(calldepth int) (file string, line int, funcName string, ok bool) {

	check := func(frame runtime.Frame) (string, int, string, bool) {
		if frame.File == "" || frame.Line <= 0 || frame.Function == "" {
			return "", 0, "", false
		}
		return frame.File, frame.Line, frame.Function, true
	}

	calldepth += 2

	for i := 0; i < 10; i++ {
		// len(pcs): we want the caller to be likely fetched by the first scan
		//           while trying not to compromise performance a lot
		var pcs [5]uintptr
		n := runtime.Callers(calldepth, pcs[:])
		if n < 1 {
			return "", 0, "", false
		}
		frames := runtime.CallersFrames(pcs[:n])
		for {
			frame, more := frames.Next()
			calldepth++
			if _, present := l.ignoredCallers[frame.Function]; !present {
				return check(frame)
			}
			if !more {
				break
			}
		}
	}

	return "", 0, "", false
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

// CallerFile adds caller info with file, and line number
func CallerFile(l *Logger) {
	l.callerFile = true
}

// CallerFunc adds caller info with function name
func CallerFunc(l *Logger) {
	l.callerFunc = true
}

// CallerPkg adds caller's package name
func CallerPkg(l *Logger) {
	l.callerPkg = true
}

// CallerIgnore sets functions that must be skipped when searching for a caller.
// Function names must be package path-qualified. Examples:
//   "main.Test"
//   "foo.Test"
//   "foo/bar.Test"
//   "foo/bar.Test.func1"
//   "foo/bar.(*Bar).Test"
// If the project is in the $GOPATH and uses vendoring but doesn't use modules
// then to ignore a function in the "$GOPATH/foo/vendor/bar/" package:
//   "foo/vendor/bar.Test"
func CallerIgnore(ignores ...string) Option {
	return func(l *Logger) {
		for _, f := range ignores {
			l.ignoredCallers[f] = struct{}{}
		}
	}
}

// LevelBraces adds [] to level
func LevelBraces(l *Logger) {
	l.levelBraces = true
}

// Msec adds .msec to timestamp
func Msec(l *Logger) {
	l.msec = true
}
