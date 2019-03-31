// Package lgr provides a simple logger with some extras. Primary way to log is Logf method.
// The logger's output can be customized in 2 ways:
//   - by passing formatting template, i.e. lgr.New(lgr.Format(lgr.Short))
//   - by setting individual formatting flags, i.e. lgr.New(lgr.Msec, lgr.CallerFunc)
// Leveled output works for messages based on level prefix, i.e. Logf("INFO some message") means INFO level.
// Debug and trace levels can be filtered based on lgr.Trace and lgr.Debug options.
// ERROR, FATAL and PANIC levels send to err as well. Both FATAL and PANIC also print stack trace and terminate caller application with os.Exit(1)

package lgr

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"
)

var levels = []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "PANIC", "FATAL"}

const (
	Short      = `{{.DT.Format "2006/01/02 15:04:05"}} {{.Level}} {{.Message}}`
	WithMsec   = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} {{.Message}}`
	WithPkg    = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerPkg}}) {{.Message}}`
	ShortDebug = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFile}}:{{.CallerLine}}) {{.Message}}`
	FuncDebug  = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFunc}}) {{.Message}}`
	FullDebug  = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFile}}:{{.CallerLine}} {{.CallerFunc}}) {{.Message}}`
)

// Logger provided simple logger with basic support of levels. Thread safe
type Logger struct {
	// set with Option calls
	stdout, stderr io.Writer // destination writes for out and err
	dbg            bool      // allows reporting for DEBUG level
	trace          bool      // allows reporting for TRACE and DEBUG levels
	callerFile     bool      // reports caller file with line number, i.e. foo/bar.go:89
	callerFunc     bool      // reports caller function name, i.e. bar.myFunc
	callerPkg      bool      // reports caller package name
	levelBraces    bool      // encloses level with [], i.e. [INFO]
	callerDepth    int       // how many stack frames to skip, relative to the real (reported) frame
	format         string    // layout template

	// internal use
	now           nowFn
	fatal         panicFn
	msec          bool
	lock          sync.Mutex
	callerOn      bool
	levelBracesOn bool
	templ         *template.Template
}

// can be redefined internally for testing
type nowFn func() time.Time
type panicFn func()

// layout holds all parts to construct the final message with template
type layout struct {
	DT         time.Time
	Level      string
	Message    string
	CallerPkg  string
	CallerFile string
	CallerFunc string
	CallerLine int
}

// New makes new leveled logger. By default writes to stdout/stderr.
// default format: 2018/01/07 13:02:34.123 DEBUG some message 123
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

	var err error
	if res.format == "" {
		res.format = res.templateFromOptions()
	}

	res.templ, err = template.New("lgr").Parse(res.format)
	if err != nil {
		fmt.Printf("invalid template %s, error %v. switched to %s\n", res.format, err, Short)
		res.format = Short
		res.templ = template.Must(template.New("lgrDefault").Parse(Short))
	}

	buf := bytes.Buffer{}
	if err = res.templ.Execute(&buf, layout{}); err != nil {
		fmt.Printf("failed to execute template %s, error %v. switched to %s\n", res.format, err, Short)
		res.format = Short
		res.templ = template.Must(template.New("lgrDefault").Parse(Short))
	}

	res.callerOn = strings.Contains(res.format, "{{.Caller")
	res.levelBracesOn = strings.Contains(res.format, "[{{.Level}}]")
	return &res
}

// Logf implements L interface to output with printf style.
// DEBUG and TRACE filtered out by dbg and trace flags.
// ERROR and FATAL also send the same line to err writer.
// FATAL and PANIC adds runtime stack and os.exit(1), like panic.
func (l *Logger) Logf(format string, args ...interface{}) {
	// to align call depth between (*Logger).Logf() and, for example, Printf()
	l.logf(format, args...)
}

func (l *Logger) logf(format string, args ...interface{}) {

	lv, msg := l.extractLevel(fmt.Sprintf(format, args...))
	if lv == "DEBUG" && !l.dbg {
		return
	}
	if lv == "TRACE" && !l.trace {
		return
	}

	ci := callerInfo{}
	if l.callerOn { // optimization to avoid expensive caller evaluation if caller info not in the template
		ci = l.reportCaller(l.callerDepth)
	}

	elems := layout{
		DT:         l.now(),
		Level:      l.formatLevel(lv),
		Message:    strings.TrimSuffix(msg, "\n"),
		CallerFunc: ci.FuncName,
		CallerFile: ci.File,
		CallerPkg:  ci.Pkg,
		CallerLine: ci.Line,
	}

	buf := bytes.Buffer{}
	err := l.templ.Execute(&buf, elems) // once constructed, a template may be executed safely in parallel.
	if err != nil {
		fmt.Printf("failed to execute template, %v\n", err)
	}
	buf.WriteString("\n")

	data := buf.Bytes()
	if l.levelBracesOn { // rearrange space in short levels
		data = bytes.Replace(data, []byte("[WARN ]"), []byte("[WARN] "), 1)
		data = bytes.Replace(data, []byte("[INFO ]"), []byte("[INFO] "), 1)
	}

	l.lock.Lock()
	_, _ = l.stdout.Write(data)

	// write to err as well for high levels, exit(1) on fatal and panic and dump stack on panic level
	switch lv {
	case "ERROR":
		_, _ = l.stderr.Write(data)
	case "FATAL":
		_, _ = l.stderr.Write(data)
		l.fatal()
	case "PANIC":
		_, _ = l.stderr.Write(data)
		_, _ = l.stderr.Write(getDump())
		l.fatal()
	}

	l.lock.Unlock()
}

type callerInfo struct {
	File     string
	Line     int
	FuncName string
	Pkg      string
}

// calldepth 0 identifying the caller of reportCaller()
func (l *Logger) reportCaller(calldepth int) (res callerInfo) {

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

	// add 5 to adjust stack level because it was called from 3 nested functions added by lgr, i.e. caller,
	// reportCaller and logf, plus 2 frames by runtime
	filePath, line, funcName := caller(calldepth + 2 + 3)
	if (filePath == "") || (line <= 0) || (funcName == "") {
		return callerInfo{}
	}

	_, pkgInfo := path.Split(path.Dir(filePath))
	res.Pkg = pkgInfo

	res.File = filePath
	if pathElems := strings.Split(filePath, "/"); len(pathElems) > 2 {
		res.File = strings.Join(pathElems[len(pathElems)-2:], "/")
	}
	res.Line = line

	funcNameElems := strings.Split(funcName, "/")
	res.FuncName = funcNameElems[len(funcNameElems)-1]

	return res
}

// make template from option flags
func (l *Logger) templateFromOptions() (res string) {

	const (
		// escape { and } from templates to allow "{some/blah}" output for caller
		openCallerBrace  = `{{"{"}}`
		closeCallerBrace = `{{"}"}}`
	)

	orElse := func(flag bool, value string, elseValue string) string {
		if flag {
			return value
		}
		return elseValue
	}

	var parts []string

	parts = append(parts, orElse(l.msec, `{{.DT.Format "2006/01/02 15:04:05.000"}}`, `{{.DT.Format "2006/01/02 15:04:05"}}`))
	parts = append(parts, orElse(l.levelBraces, `[{{.Level}}]`, `{{.Level}}`))

	if l.callerFile || l.callerFunc || l.callerPkg {
		var callerParts []string
		if v := orElse(l.callerFile, `{{.CallerFile}}:{{.CallerLine}}`, ""); v != "" {
			callerParts = append(callerParts, v)
		}
		if v := orElse(l.callerFunc, `{{.CallerFunc}}`, ""); v != "" {
			callerParts = append(callerParts, v)
		}
		if v := orElse(l.callerPkg, `{{.CallerPkg}}`, ""); v != "" {
			callerParts = append(callerParts, v)
		}
		parts = append(parts, openCallerBrace+strings.Join(callerParts, " ")+closeCallerBrace)
	}
	parts = append(parts, "{{.Message}}")
	return strings.Join(parts, " ")
}

// formatLevel aligns level to 5 chars
func (l *Logger) formatLevel(lv string) string {

	if lv == "" {
		return ""
	}

	spaces := ""
	if len(lv) == 4 {
		spaces = " "
	}
	return lv + spaces
}

// extractLevel parses messages with optional level prefix and returns level and the message with stripped level
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

// Out sets out writer, stdout by default
func Out(w io.Writer) Option {
	return func(l *Logger) {
		l.stdout = w
	}
}

// Err sets error writer, stderr by default
func Err(w io.Writer) Option {
	return func(l *Logger) {
		l.stderr = w
	}
}

// Debug turn on dbg mode
func Debug(l *Logger) {
	l.dbg = true
}

// Trace turn on trace + dbg mode
func Trace(l *Logger) {
	l.dbg = true
	l.trace = true
}

// CallerDepth sets number of stack frame skipped for caller reporting, 0 by default
func CallerDepth(n int) Option {
	return func(l *Logger) {
		l.callerDepth = n
	}
}

// Format sets output layout, overwrites all options for individual parts, i.e. Caller*, Msec and LevelBraces
func Format(f string) Option {
	return func(l *Logger) {
		l.format = f
	}
}

// CallerFunc adds caller info with function name. Ignored if Format option used.
func CallerFunc(l *Logger) {
	l.callerFunc = true
}

// CallerPkg adds caller's package name. Ignored if Format option used.
func CallerPkg(l *Logger) {
	l.callerPkg = true
}

// LevelBraces surrounds level with [], i.e. [INFO]. Ignored if Format option used.
func LevelBraces(l *Logger) {
	l.levelBraces = true
}

// CallerFile adds caller info with file, and line number. Ignored if Format option used.
func CallerFile(l *Logger) {
	l.callerFile = true
}

// Msec adds .msec to timestamp. Ignored if Format option used.
func Msec(l *Logger) {
	l.msec = true
}
