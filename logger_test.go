package lgr

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			"2018/01/07 13:02:34.123 INFO  (" + testDir + "/logger_test.go:83 lgr.TestLoggerWithDbg.func2) aaa\n", ""},
		{"DEBUG something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 DEBUG (" + testDir + "/logger_test.go:83 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"[DEBUG] something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 DEBUG (" + testDir + "/logger_test.go:83 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"INFO something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 INFO  (" + testDir + "/logger_test.go:83 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"[INFO] something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 INFO  (" + testDir + "/logger_test.go:83 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"blah something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 INFO  (" + testDir + "/logger_test.go:83 lgr.TestLoggerWithDbg.func2) blah something 123 aaa\n", ""},
		{"WARN something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 WARN  (" + testDir + "/logger_test.go:83 lgr.TestLoggerWithDbg.func2) something 123 aaa\n", ""},
		{"ERROR something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.123 ERROR (" + testDir + "/logger_test.go:83 lgr.TestLoggerWithDbg.func2) something 123 aaa\n",
			"2018/01/07 13:02:34.123 ERROR (" + testDir + "/logger_test.go:83 lgr.TestLoggerWithDbg.func2) something 123 aaa\n"},
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
	assert.Empty(t, rerr.String())

	l = New(Debug, Out(rout), Err(rerr), Format(ShortDebug)) // caller file only
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34.000 DEBUG ("+testDir+"/logger_test.go:101) something 123 err\n", rout.String())

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
	assert.Equal(t, "2018/01/07 13:02:34.123 DEBUG ("+testDir+") something 123 err\n", rout.String())
}

func TestLoggerWithCallerDepth(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l1 := New(Debug, Out(rout), Err(rerr), Format(FullDebug), CallerDepth(1))
	l1.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }

	f := func(l L) {
		l.Logf("[DEBUG] something 123 %s", "err")
	}
	f(l1)

	assert.Equal(t, "2018/01/07 13:02:34.123 DEBUG ("+testDir+"/logger_test.go:129 lgr.TestLoggerWithCallerDepth) something 123 err\n",
		rout.String())

	rout.Reset()
	rerr.Reset()
	l2 := New(Debug, Out(rout), Err(rerr), Format(FullDebug), CallerDepth(0))
	l2.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	f(l2)
	assert.Equal(t, "2018/01/07 13:02:34.123 DEBUG ("+testDir+"/logger_test.go:127 lgr.TestLoggerWithCallerDepth."+
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

	t.Log(rerr.String()) //nolint:govet
	assert.True(t, strings.HasPrefix(rerr.String(), "2018/01/07 13:02:34.000 PANIC"))
	assert.Contains(t, rerr.String(), "github.com/go-pkgz/lgr.getDump")
	assert.Contains(t, rerr.String(), "/"+testDir+"/logger.go:")

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
	assert.Contains(t, rerr.String(), "/"+testDir+"/logger.go:")
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
	assert.Contains(t, lines[3], testDir+"/logger_test.go:363")
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

	assert.Len(t, strings.Split(rout.String(), "\n"), 1001)
	assert.Empty(t, rerr.String())
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
	assert.Empty(t, rout.String())

	l = New(Trace, Out(rout), Err(rerr), CallerPkg)
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[TRACE] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34 TRACE {"+testDir+"} something 123 err\n", rout.String())
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
	l := New(Out(rout), Err(rerr), Format(Short), Secret("password", "secret", "", " "))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 123000000, time.Local) }
	l.Logf("INFO something password 123 secret xyz")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  something ****** 123 ****** xyz\n", rout.String(), "secrets secrets")
}

func TestIsStreamsSame(t *testing.T) {
	{ // with stdout and stderr
		sout, serr := os.Stdout, os.Stderr
		assert.True(t, isStreamsSame(sout, serr))
	}
	{ // with same file
		dir := t.TempDir()
		f, err := os.CreateTemp(dir, "test")
		require.NoError(t, err)
		defer os.Remove(f.Name())
		assert.True(t, isStreamsSame(f, f))
	}
	{ // with different files
		dir := t.TempDir()
		f1, err := os.CreateTemp(dir, "test")
		require.NoError(t, err)
		defer os.Remove(f1.Name())
		f2, err := os.CreateTemp(dir, "test")
		require.NoError(t, err)
		defer os.Remove(f2.Name())
		assert.False(t, isStreamsSame(f1, f2))
	}
	{ // with same buffer
		buf := bytes.NewBuffer([]byte{})
		assert.True(t, isStreamsSame(buf, buf))
	}
	{ // with different buffers
		buf1, buf2 := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
		sout, serr := buf1, buf2
		assert.False(t, isStreamsSame(sout, serr))
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

var testDir = func() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd[strings.LastIndex(wd, "/")+1:]
}()

func TestLogger_SecretsLongestFirst(t *testing.T) {
	rout := bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rout), Format(Short), Secret("abc", "abcdef"))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	l.Logf("INFO token abcdef and abc")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  token ****** and ******\n", rout.String())
}

func TestLogger_AddRemoveSecrets(t *testing.T) {
	rout := bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rout), Format(Short), Secret("password"))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	l.Logf("INFO password token1 token2")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  ****** token1 token2\n", rout.String())

	rout.Reset()
	l.AddSecrets("token1", "token2", "", " ")
	l.AddSecrets("token1")
	l.Logf("INFO password token1 token2")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  ****** ****** ******\n", rout.String())

	rout.Reset()
	l.RemoveSecrets("token1", "password")
	l.Logf("INFO password token1 token2")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  password token1 ******\n", rout.String())
}

func TestLogger_SecretsReadOncePerLine(t *testing.T) {
	rout := bytes.NewBuffer([]byte{})
	var l *Logger
	removeAndLeak := func(s string) string {
		l.RemoveSecrets("token1")
		return s + " token1"
	}
	mapper := nopMapper
	mapper.MessageFunc = removeAndLeak
	l = New(Out(rout), Err(rout), Secret("token1"), Map(mapper))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	l.Logf("INFO first")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  first ******\n", rout.String(), "secret removed mid-line is still hidden in that line")

	rout.Reset()
	l.Logf("INFO second")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  second token1\n", rout.String())
}

func TestLogger_NewIgnoresPackageSecrets(t *testing.T) {
	AddSecrets("pkg-secret")
	defer RemoveSecrets("pkg-secret")

	rout := bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rout), Format(Short))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	l.Logf("INFO with pkg-secret")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  with pkg-secret\n", rout.String())
}

func TestLogger_SecretsChangedWhileLogging(t *testing.T) {
	rout := &syncBuffer{}
	l := New(Out(rout), Err(rout), Format(Short), Secret("initial"))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				l.Logf("INFO initial kept dropped %d", j)
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				v := "tmp-" + strconv.Itoa(i) + "-" + strconv.Itoa(j)
				l.AddSecrets(v, "kept", "dropped")
				l.RemoveSecrets(v)
			}
		}(i)
	}
	wg.Wait()
	assert.NotContains(t, rout.String(), "initial")

	l.RemoveSecrets("dropped")
	rout.Reset()
	l.Logf("INFO initial kept dropped tmp-0-0")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  ****** ****** dropped tmp-0-0\n", rout.String())
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *syncBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Reset()
}

func TestLogger_MapperWithNilFuncs(t *testing.T) {
	tbl := []struct {
		name   string
		mapper Mapper
		want   string
	}{
		{name: "message func only", mapper: Mapper{MessageFunc: strings.ToUpper}, want: "2018/01/07 13:02:34 INFO  SOMETHING 123\n"},
		{name: "zero mapper", mapper: Mapper{}, want: "2018/01/07 13:02:34 INFO  something 123\n"},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			rout := bytes.NewBuffer([]byte{})
			l := New(Out(rout), Err(rout), Map(tt.mapper))
			l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
			require.NotPanics(t, func() { l.Logf("INFO something 123") })
			assert.Equal(t, tt.want, rout.String())
		})
	}
}

func TestLogger_TraceRegexDropsLoggerFrames(t *testing.T) {
	caller := "main.doWork(...)\n\t/app/main.go:16\nmain.main()\n\t/app/main.go:12 +0xe0\n"
	printf := "github.com/go-pkgz/lgr.Printf(...)\n\t/src/lgr/interface.go:31\n"
	tbl := []struct {
		name  string
		stack string
		want  string
	}{
		{name: "module cache", want: caller, stack: "goroutine 1 [running]:\n" +
			"github.com/go-pkgz/lgr.(*Logger).logf(0x1, {0x2?, 0x3?}, {0x4?, 0x5?, 0x6?})\n" +
			"\t/home/user/go/pkg/mod/github.com/go-pkgz/lgr@v0.12.4/logger.go:238 +0x944\n" +
			"github.com/go-pkgz/lgr.(*Logger).Logf(...)\n" +
			"\t/home/user/go/pkg/mod/github.com/go-pkgz/lgr@v0.12.4/logger.go:147\n" + caller},
		{name: "vendored with prefix", want: caller, stack: "goroutine 1 [running]:\n" +
			"app/vendor/github.com/go-pkgz/lgr.(*Logger).logf(0x1, {0x2, 0x3}, {0x4, 0x5, 0x6})\n" +
			"\t/app/vendor/github.com/go-pkgz/lgr/logger.go:238 +0x944\n" +
			"app/vendor/github.com/go-pkgz/lgr.(*Logger).Logf(...)\n" +
			"\t/app/vendor/github.com/go-pkgz/lgr/logger.go:147\n" + caller},
		{name: "renamed checkout", want: caller, stack: "goroutine 1 [running]:\n" +
			"github.com/go-pkgz/lgr.(*Logger).logf(0x1, {0x2, 0x3}, {0x4, 0x5, 0x6})\n" +
			"\t/src/some-other-name/logger.go:238 +0x944\n" + caller},
		{name: "package Printf frame is kept", want: printf + caller, stack: "goroutine 1 [running]:\n" +
			"github.com/go-pkgz/lgr.(*Logger).logf(0x1, {0x2, 0x3}, {0x4, 0x5, 0x6})\n" +
			"\t/src/lgr/logger.go:238 +0x944\n" + printf + caller},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			parts := reTraceDefault.Split(tt.stack, -1)
			assert.Equal(t, tt.want, parts[len(parts)-1])
		})
	}
}
