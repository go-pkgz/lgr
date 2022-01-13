package lgr

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoggerNoDbg(t *testing.T) {
	tbl := []struct {
		format     string
		args       []interface{}
		rout, rerr string
	}{
		{"aaa", []interface{}{}, "2018/01/07 13:02:34.000 INFO  aaa\n", ""},
		{"DEBUG something 123 %s", []interface{}{"aaa"}, "", ""},
		{"[DEBUG] something 123 %s", []interface{}{"aaa"}, "", ""},
		{"INFO something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 INFO  something 123 aaa\n", ""},
		{"[INFO] something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 INFO  something 123 aaa\n", ""},
		{"[INFO] something 123 %s", []interface{}{"aaa\n"}, "2018/01/07 13:02:34.000 INFO  something 123 aaa\n", ""},
		{"blah something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 INFO  blah something 123 aaa\n", ""},
		{"WARN something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 WARN  something 123 aaa\n", ""},
		{"ERROR something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 ERROR something 123 aaa\n",
			"2018/01/07 13:02:34.000 ERROR something 123 aaa\n"},
	}
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr), Msec)
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	for i, tt := range tbl {
		tt := tt
		rout.Reset()
		rerr.Reset()
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			l.Logf(tt.format, tt.args...)
			assert.Equal(t, tt.rout, rout.String())
			assert.Equal(t, tt.rerr, rerr.String())
		})
	}
}

func TestLoggerWithDbg(t *testing.T) {
	tbl := []struct {
		format     string
		args       []interface{}
		rout, rerr string
	}{
		{"aaa", []interface{}{},
			"2018/01/07 13:02:34.123 INFO  (lgr/logger_test.go:81 lgr.TestLoggerWithDbg.func2) aaa\n", ""},
		{"DEBUG something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 DEBUG (lgr/logger_test.go:81 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"[DEBUG] something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 DEBUG (lgr/logger_test.go:81 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"INFO something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 INFO  (lgr/logger_test.go:81 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"[INFO] something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 INFO  (lgr/logger_test.go:81 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"blah something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 INFO  (lgr/logger_test.go:81 lgr.TestLoggerWithDbg.func2) blah something 123 aaa\n", ""},
		{"WARN something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 WARN  (lgr/logger_test.go:81 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"ERROR something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 ERROR (lgr/logger_test.go:81 lgr.TestLoggerWithDbg.func2) something 123 aaa\n",
			"2018/01/07 13:02:34.123 ERROR (lgr/logger_test.go:81 lgr.TestLoggerWithDbg.func2) something 123 aaa\n"},
	}

	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Format(FullDebug), Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	for i, tt := range tbl {
		tt := tt
		rout.Reset()
		rerr.Reset()
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			l.Logf(tt.format, tt.args...)
			assert.Equal(t, tt.rout, rout.String())
			assert.Equal(t, tt.rerr, rerr.String())
		})
	}

	l = New(Debug, Out(rout), Err(rerr), Format(WithMsec)) // no caller
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34.000 DEBUG something 123 err\n", rout.String())
	assert.Equal(t, "", rerr.String())

	l = New(Debug, Out(rout), Err(rerr), Format(ShortDebug)) // caller file only
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34.000 DEBUG (lgr/logger_test.go:99) something 123 err\n", rout.String())

	f := `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFunc}}) {{.Message}}`
	l = New(Debug, Out(rout), Err(rerr), Format(f)) // caller func only
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34.000 DEBUG (lgr.TestLoggerWithDbg) something 123 err\n", rout.String())
}

func TestLoggerWithPkg(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Out(rout), Err(rerr), Format(WithPkg))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34.123 DEBUG (lgr) something 123 err\n", rout.String())
}

func TestLoggerWithCallerDepth(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l1 := New(Debug, Out(rout), Err(rerr), Format(FullDebug), CallerDepth(1))
	l1.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }

	f := func(l L) {
		l.Logf("[DEBUG] something 123 %s", "err")
	}
	f(l1)

	assert.Equal(t, "2018/01/07 13:02:34.123 DEBUG (lgr/logger_test.go:127 lgr.TestLoggerWithCallerDepth) something 123 err\n",
		rout.String())

	rout.Reset()
	rerr.Reset()
	l2 := New(Debug, Out(rout), Err(rerr), Format(FullDebug), CallerDepth(0))
	l2.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	f(l2)
	assert.Equal(t, "2018/01/07 13:02:34.123 DEBUG (lgr/logger_test.go:125 lgr.TestLoggerWithCallerDepth."+
		"func2) something 123 err\n", rout.String())
}

// nolint dupl
func TestLogger_formatWithOptions(t *testing.T) {
	tbl := []struct {
		opts  []Option
		elems layout
		res   string
	}{
		{
			[]Option{},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "INFO "},
			"2018/01/07 13:02:34 INFO  blah blah",
		},
		{
			[]Option{Msec},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG"},
			"2018/01/07 13:02:34.000 DEBUG blah blah",
		},
		{
			[]Option{Msec, LevelBraces},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG"},
			"2018/01/07 13:02:34.000 [DEBUG] blah blah",
		},
		{
			[]Option{CallerFile, Msec},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG",
				CallerFile: "file1.go", CallerLine: 12},
			"2018/01/07 13:02:34.000 DEBUG {file1.go:12} blah blah",
		},
		{
			[]Option{CallerFunc, CallerPkg},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG",
				CallerFunc: "func1", CallerPkg: "pkg"},
			"2018/01/07 13:02:34 DEBUG {func1 pkg} blah blah",
		},
	}

	for n, tt := range tbl {
		tt := tt
		l := New(tt.opts...)
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			assert.Equal(t, tt.res, l.formatWithOptions(tt.elems))
		})
	}
}

// nolint dupl
func TestLogger_formatWithMapper(t *testing.T) {
	tbl := []struct {
		opts  []Option
		elems layout
		res   string
	}{
		{
			[]Option{},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "INFO "},
			"!TM=2018/01/07 13:02:34=TM! !IF=INFO =IF! !IF=blah blah*=IF!",
		},
		{
			[]Option{Msec},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG"},
			"!TM=2018/01/07 13:02:34.000=TM! !DG=DEBUG=DG! !DG=blah blah*=DG!",
		},
		{
			[]Option{Msec, LevelBraces},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG"},
			"!TM=2018/01/07 13:02:34.000=TM! !DG=[DEBUG]=DG! !DG=blah blah*=DG!",
		},
		{
			[]Option{CallerFile, Msec},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG",
				CallerFile: "file1.go", CallerLine: 12},
			"!TM=2018/01/07 13:02:34.000=TM! !DG=DEBUG=DG! !CL={file1.go:12}=CL! !DG=blah blah*=DG!",
		},
		{
			[]Option{CallerFunc, CallerPkg},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG",
				CallerFunc: "func1", CallerPkg: "pkg"},
			"!TM=2018/01/07 13:02:34=TM! !DG=DEBUG=DG! !CL={func1 pkg}=CL! !DG=blah blah*=DG!",
		},
	}

	mp := Mapper{
		MessageFunc: func(s string) string {
			return s + "*"
		},
		ErrorFunc: func(s string) string {
			return "!ER=" + s + "=ER!"
		},
		WarnFunc: func(s string) string {
			return "!WR=" + s + "=WR!"
		},
		InfoFunc: func(s string) string {
			return "!IF=" + s + "=IF!"
		},
		DebugFunc: func(s string) string {
			return "!DG=" + s + "=DG!"
		},
		CallerFunc: func(s string) string {
			return "!CL=" + s + "=CL!"
		},
		TimeFunc: func(s string) string {
			return "!TM=" + s + "=TM!"
		},
	}

	for n, tt := range tbl {
		tt := tt
		opts := []Option{}
		opts = append(opts, tt.opts...)
		opts = append(opts, Map(mp))
		l := New(opts...)
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			assert.Equal(t, tt.res, l.formatWithOptions(tt.elems))
		})
	}
}

// nolint dupl
func TestLogger_formatWithPartialMapper(t *testing.T) {
	tbl := []struct {
		opts  []Option
		elems layout
		res   string
	}{
		{
			[]Option{},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "INFO "},
			"!TM=2018/01/07 13:02:34=TM! INFO  blah blah*",
		},
		{
			[]Option{Msec},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG"},
			"!TM=2018/01/07 13:02:34.000=TM! DEBUG blah blah*",
		},
		{
			[]Option{Msec, LevelBraces},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG"},
			"!TM=2018/01/07 13:02:34.000=TM! [DEBUG] blah blah*",
		},
		{
			[]Option{CallerFile, Msec},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG",
				CallerFile: "file1.go", CallerLine: 12},
			"!TM=2018/01/07 13:02:34.000=TM! DEBUG {file1.go:12} blah blah*",
		},
		{
			[]Option{CallerFunc, CallerPkg},
			layout{DT: time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local), Message: "blah blah", Level: "DEBUG",
				CallerFunc: "func1", CallerPkg: "pkg"},
			"!TM=2018/01/07 13:02:34=TM! DEBUG {func1 pkg} blah blah*",
		},
	}

	mp := Mapper{
		MessageFunc: func(s string) string {
			return s + "*"
		},

		WarnFunc: func(s string) string {
			return "!WR=" + s + "=WR!"
		},

		TimeFunc: func(s string) string {
			return "!TM=" + s + "=TM!"
		},
	}

	for n, tt := range tbl {
		tt := tt
		opts := []Option{}
		opts = append(opts, tt.opts...)
		opts = append(opts, Map(mp))
		l := New(opts...)
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			assert.Equal(t, tt.res, l.formatWithOptions(tt.elems))
		})
	}
}

func TestLoggerWithPanic(t *testing.T) {
	fatalCalls := 0
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Format(FuncDebug), Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	l.fatal = func() { fatalCalls++ }

	l.Logf("PANIC oh my, panic now! %v", errors.New("bad thing happened"))
	assert.Equal(t, 1, fatalCalls)
	assert.Equal(t, "2018/01/07 13:02:34.000 PANIC (lgr.TestLoggerWithPanic) oh my, panic now! bad thing happened\n", rout.String())

	t.Logf(rerr.String())
	assert.True(t, strings.HasPrefix(rerr.String(), "2018/01/07 13:02:34.000 PANIC"))
	assert.Contains(t, rerr.String(), "github.com/go-pkgz/lgr.getDump")
	assert.Contains(t, rerr.String(), "/lgr/logger.go:")

	rout.Reset()
	rerr.Reset()
	l.Logf("[FATAL] oh my, fatal error! %v", errors.New("bad thing happened"))
	assert.Equal(t, 2, fatalCalls)
	assert.Equal(t, "2018/01/07 13:02:34.000 FATAL (lgr.TestLoggerWithPanic) oh my, fatal error! bad thing happened\n", rout.String())
	assert.Equal(t, "2018/01/07 13:02:34.000 FATAL (lgr.TestLoggerWithPanic) oh my, fatal error! bad thing happened\n", rerr.String())

	rout.Reset()
	rerr.Reset()
	fatalCalls = 0
	l = New(Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	l.fatal = func() { fatalCalls++ }
	l.Logf("[PANIC] oh my, panic now! %v", errors.New("bad thing happened"))
	assert.Equal(t, 1, fatalCalls)
	assert.Equal(t, "2018/01/07 13:02:34 PANIC oh my, panic now! bad thing happened\n", rout.String())
	assert.True(t, strings.HasPrefix(rerr.String(), "2018/01/07 13:02:34 PANIC"))
	assert.Contains(t, rerr.String(), "github.com/go-pkgz/lgr.getDump")
	assert.Contains(t, rerr.String(), "/lgr/logger.go:")
}

func TestLoggerErrorWithDump(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Format(FuncDebug), Out(rout), Err(rerr), StackTraceOnError)
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	l.Logf("ERROR oh my, error now! %v", errors.New("bad thing happened"))
	lines := strings.Split(rout.String(), "\n")
	assert.Equal(t, "2018/01/07 13:02:34.000 ERROR (lgr.TestLoggerErrorWithDump) oh my, error now! bad thing happened", lines[0])
	assert.Equal(t, ">>> stack trace:", lines[1])
	assert.Contains(t, lines[2], "github.com/go-pkgz/lgr.TestLoggerErrorWithDump(")
	assert.Contains(t, lines[3], "lgr/logger_test.go:361")
}

func TestLoggerWithErrorSameOutputs(t *testing.T) {
	fatalCalls := 0
	rout := bytes.NewBuffer([]byte{})
	l := New(Debug, Format(FuncDebug), Out(rout), Err(rout))
	l.fatal = func() { fatalCalls++ }
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	l.Logf("ERROR oh my, error now! %v", errors.New("bad thing happened"))
	assert.Equal(t, "2018/01/07 13:02:34.000 ERROR (lgr.TestLoggerWithErrorSameOutputs) oh my, error now! bad thing happened\n", rout.String())

	rout.Reset()
	l.Logf("FATAL oh my, error now! %v", errors.New("bad thing happened"))
	assert.Equal(t, "2018/01/07 13:02:34.000 FATAL (lgr.TestLoggerWithErrorSameOutputs) oh my, error now! bad thing happened\n", rout.String())
	assert.Equal(t, 1, fatalCalls)
}

func TestLoggerConcurrent(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func(i int) {
			l.Logf("[DEBUG] test test 123 debug message #%d, %v", i, errors.New("some error"))
			wg.Done()
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 1001, len(strings.Split(rout.String(), "\n")))
	assert.Equal(t, "", rerr.String())
}

func TestLoggerWithLevelBraces(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Out(rout), Err(rerr), Format(`{{.DT.Format "2006/01/02 15:04:05"}} [{{.Level}}] {{.Message}}`))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	l.Logf("[INFO] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34 [INFO]  something 123 err\n", rout.String())

	l = New(Debug, Out(rout), Err(rerr), LevelBraces)
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[ERROR] some warning 123")
	assert.Equal(t, "2018/01/07 13:02:34 [ERROR] some warning 123\n", rout.String())
	assert.Equal(t, "2018/01/07 13:02:34 [ERROR] some warning 123\n", rerr.String())

	rout.Reset()
	rerr.Reset()
	l.Logf("WARN some warning 123")
	assert.Equal(t, "2018/01/07 13:02:34 [WARN]  some warning 123\n", rout.String())
}

func TestLoggerWithTrace(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Trace, Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }

	l.Logf("[INFO] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  something 123 err\n", rout.String())

	rout.Reset()
	rerr.Reset()
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34 DEBUG something 123 err\n", rout.String())

	rout.Reset()
	rerr.Reset()
	l.Logf("[TRACE] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34 TRACE something 123 err\n", rout.String())

	l = New(Debug, Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[TRACE] something 123 %s", "err")
	assert.Equal(t, "", rout.String())

	l = New(Trace, Out(rout), Err(rerr), CallerPkg)
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[TRACE] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34 TRACE {lgr} something 123 err\n", rout.String())
}

func TestLoggerWithInvalidTemplate(t *testing.T) {

	// invalid template format
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr), Format(`{{.DT.Format "2006/01/02 15:04:05"}} {{{.BadThing}} {{.Message}}`))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	l.Logf("[INFO] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  something 123 err\n", rout.String(), "default format")

	// invalid var
	rout, rerr = bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l1 := New(Out(rout), Err(rerr), Format(`{{.DT.Format "2006/01/02 15:04:05"}} {{.BadThing}} {{.Message}}`))
	l1.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	l1.Logf("[INFO] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  something 123 err\n", rout.String(), "default format")
}

func TestLoggerOverwriteFormat(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Out(rout), Err(rerr), Msec, Format(Short), CallerFile) // mix Format with individual flags
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	l.Logf("INFO something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  something 123 err\n", rout.String(), "short format enforced")
}

func TestLoggerNoSpaceLevel(t *testing.T) {
	tbl := []struct {
		format     string
		args       []interface{}
		rout, rerr string
	}{
		{"INFOsomething 123 %s", []interface{}{"aaa1"}, "2018/01/07 13:02:34.000 INFO  something 123 aaa1\n", ""},
		{"[INFO]something 123 %s", []interface{}{"aaa1"}, "2018/01/07 13:02:34.000 INFO  something 123 aaa1\n", ""},
		{"[INFO]something 123 %s", []interface{}{"aaa1\n"}, "2018/01/07 13:02:34.000 INFO  something 123 aaa1\n", ""},
		{"WARNsomething 123 %s", []interface{}{"aaa1"}, "2018/01/07 13:02:34.000 WARN  something 123 aaa1\n", ""},
		{"ERRORsomething 123 %s", []interface{}{"aaa1"}, "2018/01/07 13:02:34.000 ERROR something 123 aaa1\n",
			"2018/01/07 13:02:34.000 ERROR something 123 aaa1\n"},
	}
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr), Msec)
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	for i, tt := range tbl {
		tt := tt
		rout.Reset()
		rerr.Reset()
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			l.Logf(tt.format, tt.args...)
			assert.Equal(t, tt.rout, rout.String())
			assert.Equal(t, tt.rerr, rerr.String())
		})
	}
}

func TestLoggerHidden(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr), Format(Short), Secret("password", "secret"))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	l.Logf("INFO something password 123 secret xyz")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  something ****** 123 ****** xyz\n", rout.String(), "secrets secrets")
}

func TestLogger_Shortcuts(t *testing.T) {
	{
		rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
		lg := New(Debug, Out(rout), Err(rerr), Format(FullDebug))
		lg.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }

		lg.Infof("the message %s blah %d", "test", 123)
		assert.Equal(t, "2018/01/07 13:02:34.123 INFO  (lgr/logger_test.go:525 lgr.TestLogger_Shortcuts) the message test blah 123\n",
			rout.String())
	}

	{
		rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
		lg := New(Out(rout), Err(rerr), LevelBraces)
		lg.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }

		lg.Infof("the message %s blah %d", "test", 123)
		assert.Equal(t, "2018/01/07 13:02:34 [INFO]  the message test blah 123\n", rout.String())
	}

	{
		rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
		lg := New(Out(rout), Err(rerr), LevelBraces)
		lg.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }

		lg.Infof("the message")
		assert.Equal(t, "2018/01/07 13:02:34 [INFO]  the message\n", rout.String())
	}
	{
		rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
		lg := New(Out(rout), Err(rerr), LevelBraces, Debug)
		lg.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }

		lg.Debugf("the message %s blah %d", "test", 123)
		assert.Equal(t, "2018/01/07 13:02:34 [DEBUG] the message test blah 123\n", rout.String())
	}

	{
		rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
		lg := New(Out(rout), Err(rerr), LevelBraces, Trace)
		lg.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }

		lg.Tracef("the message %s blah %d", "test", 123)
		assert.Equal(t, "2018/01/07 13:02:34 [TRACE] the message test blah 123\n", rout.String())
	}

	{
		rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
		lg := New(Out(rout), Err(rerr), Msec)
		lg.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }

		lg.Warnf("the message %s blah %d", "test", 123)
		assert.Equal(t, "2018/01/07 13:02:34.123 WARN  the message test blah 123\n", rout.String())
	}
}

func BenchmarkNoDbgNoFormat(b *testing.B) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	e := errors.New("some error")
	for n := 0; n < b.N; n++ {
		l.Logf("[INFO] test test 123 debug message #%d, %v", n, e)
	}
}

func BenchmarkNoDbgFormat(b *testing.B) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr), Format(Short))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	e := errors.New("some error")
	for n := 0; n < b.N; n++ {
		l.Logf("[INFO] test test 123 debug message #%d, %v", n, e)
	}
}

func BenchmarkWithDbgNoFormat(b *testing.B) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Out(rout), Err(rerr), CallerFile, CallerFunc, CallerPkg)
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	e := errors.New("some error")
	for n := 0; n < b.N; n++ {
		l.Logf("INFO test test 123 debug message #%d, %v", n, e)
	}
}

func BenchmarkWithDbgAndFormat(b *testing.B) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Format(FullDebug), Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	e := errors.New("some error")
	for n := 0; n < b.N; n++ {
		l.Logf("INFO test test 123 debug message #%d, %v", n, e)
	}
}
