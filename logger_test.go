package lgr

import (
	"bytes"
	"errors"
	"fmt"
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
		{"", []interface{}{}, "2018/01/07 13:02:34.000 \n", ""},
		{"DEBUG something 123 %s", []interface{}{"aaa"}, "", ""},
		{"[DEBUG] something 123 %s", []interface{}{"aaa"}, "", ""},
		{"INFO something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 INFO  something 123 aaa\n", ""},
		{"[INFO] something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 INFO  something 123 aaa\n", ""},
		{"blah something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 blah something 123 aaa\n", ""},
		{"WARN something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 WARN  something 123 aaa\n", ""},
		{"ERROR something 123 %s", []interface{}{"aaa"}, "2018/01/07 13:02:34.000 ERROR something 123 aaa\n",
			"2018/01/07 13:02:34.000 ERROR something 123 aaa\n"},
	}
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	for i, tt := range tbl {
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
		{"", []interface{}{},
			"2018/01/07 13:02:34.000 {lgr/logger_test.go:79 lgr.TestLoggerWithDbg.func2} \n", ""},
		{"DEBUG something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.000 DEBUG {lgr/logger_test.go:79 lgr.TestLoggerWithDbg.func2} something 123 aaa\n", ""},
		{"[DEBUG] something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.000 DEBUG {lgr/logger_test.go:79 lgr.TestLoggerWithDbg.func2} something 123 aaa\n", ""},
		{"INFO something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.000 INFO  {lgr/logger_test.go:79 lgr.TestLoggerWithDbg.func2} something 123 aaa\n", ""},
		{"[INFO] something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.000 INFO  {lgr/logger_test.go:79 lgr.TestLoggerWithDbg.func2} something 123 aaa\n", ""},
		{"blah something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.000 {lgr/logger_test.go:79 lgr.TestLoggerWithDbg.func2} blah something 123 aaa\n", ""},
		{"WARN something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.000 WARN  {lgr/logger_test.go:79 lgr.TestLoggerWithDbg.func2} something 123 aaa\n", ""},
		{"ERROR something 123 %s", []interface{}{"aaa"},
			"2018/01/07 13:02:34.000 ERROR {lgr/logger_test.go:79 lgr.TestLoggerWithDbg.func2} something 123 aaa\n",
			"2018/01/07 13:02:34.000 ERROR {lgr/logger_test.go:79 lgr.TestLoggerWithDbg.func2} something 123 aaa\n"},
	}

	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, CallerFile, CallerFunc, Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	for i, tt := range tbl {
		rout.Reset()
		rerr.Reset()
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			l.Logf(tt.format, tt.args...)
			assert.Equal(t, tt.rout, rout.String())
			assert.Equal(t, tt.rerr, rerr.String())
		})
	}

	l = New(Debug, Out(rout), Err(rerr)) // no caller
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34.000 DEBUG something 123 err\n", rout.String())
	assert.Equal(t, "", rerr.String())

	l = New(Debug, Out(rout), Err(rerr), CallerFile) // caller file only
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34.000 DEBUG {lgr/logger_test.go:97} something 123 err\n", rout.String())

	l = New(Debug, Out(rout), Err(rerr), CallerFunc) // caller func only
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	rout.Reset()
	rerr.Reset()
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34.000 DEBUG {lgr.TestLoggerWithDbg} something 123 err\n", rout.String())
}

func TestLoggerWithLevelBraces(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, Out(rout), Err(rerr), LevelBraces)
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	l.Logf("[DEBUG] something 123 %s", "err")
	assert.Equal(t, "2018/01/07 13:02:34.000 [DEBUG] something 123 err\n", rout.String())
}

func TestLoggerWithPanic(t *testing.T) {
	fatalCalls := 0
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, CallerFunc, Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	l.fatal = func() { fatalCalls++ }

	l.Logf("[PANIC] oh my, panic now! %v", errors.New("bad thing happened"))
	assert.Equal(t, 1, fatalCalls)
	assert.Equal(t, "2018/01/07 13:02:34.000 PANIC {lgr.TestLoggerWithPanic} oh my, panic now! bad thing happened\n", rout.String())

	t.Logf(rerr.String())
	assert.True(t, strings.HasPrefix(rerr.String(), "2018/01/07 13:02:34.000 PANIC"))
	assert.True(t, strings.Contains(rerr.String(), "github.com/go-pkgz/lgr.getDump"))
	assert.True(t, strings.Contains(rerr.String(), "go-pkgz/lgr/logger.go:"))
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

func BenchmarkNoDbg(b *testing.B) {

	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	e := errors.New("some error")
	for n := 0; n < b.N; n++ {
		l.Logf("[INFO] test test 123 debug message #%d, %v", n, e)
	}
}

func BenchmarkWithDbg(b *testing.B) {

	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Debug, CallerFile, CallerFunc, Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	e := errors.New("some error")
	for n := 0; n < b.N; n++ {
		l.Logf("[INFO] test test 123 debug message #%d, %v", n, e)
	}
}
