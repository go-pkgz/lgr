package lgr

import (
	"bytes"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLevelPrefixes(t *testing.T) {
	tbl := []struct {
		line, level, msg string
		found            bool
	}{
		{"WARN disk almost full", "WARN", "disk almost full", true},
		{"WARN: disk almost full", "WARN", ": disk almost full", true},
		{"WARN\tdisk almost full", "WARN", "disk almost full", true},
		{"WARN", "WARN", "", true},
		{"[WARN]disk almost full", "WARN", "disk almost full", true},
		{"WARNING: disk almost full", "INFO", "WARNING: disk almost full", false},
		{"DEBUGGING started", "INFO", "DEBUGGING started", false},
		{"TRACEROUTE done", "INFO", "TRACEROUTE done", false},
		{"ERRORS: 3", "INFO", "ERRORS: 3", false},
		{"FATALITY count is 3", "INFO", "FATALITY count is 3", false},
		{"PANICKED users: 0", "INFO", "PANICKED users: 0", false},
		{"no prefix", "INFO", "no prefix", false},
	}
	for _, tt := range tbl {
		t.Run(tt.line, func(t *testing.T) {
			level, msg, found := extractLevel(tt.line)
			assert.Equal(t, tt.level, level)
			assert.Equal(t, tt.msg, msg)
			assert.Equal(t, tt.found, found)
		})
	}
}

func TestLoggerLevelNeedsBoundary(t *testing.T) {
	fatalCalls := 0
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	l.fatal = func() { fatalCalls++ }

	l.Logf("FATALITY count is 3")
	l.Logf("PANICKED users: 0")
	l.Logf("ERRORS: 3")
	l.Logf("DEBUGGING started")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  FATALITY count is 3\n2018/01/07 13:02:34 INFO  PANICKED users: 0\n"+
		"2018/01/07 13:02:34 INFO  ERRORS: 3\n2018/01/07 13:02:34 INFO  DEBUGGING started\n", rout.String())
	assert.Empty(t, rerr.String())
	assert.Equal(t, 0, fatalCalls)
}

func TestLoggerSlogHandlerLevel(t *testing.T) {
	fatalCalls := 0
	hout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Err(rerr), Debug, SlogHandler(slog.NewTextHandler(hout, nil))) // handler at INFO
	l.fatal = func() { fatalCalls++ }

	l.Logf("DEBUG below the handler level")
	l.Logf("INFO at the handler level")
	assert.NotContains(t, hout.String(), "below the handler level")
	assert.Contains(t, hout.String(), `level=INFO msg="at the handler level"`)

	hout.Reset()
	l = New(Err(rerr), SlogHandler(slog.NewTextHandler(hout, &slog.HandlerOptions{Level: slog.LevelError + 4})))
	l.fatal = func() { fatalCalls++ }
	l.Logf("FATAL not taken by the handler")
	assert.Equal(t, 1, fatalCalls, "FATAL exits whether the handler takes the record or not")
	l.Logf("PANIC not taken by the handler")
	assert.Equal(t, 2, fatalCalls, "PANIC exits whether the handler takes the record or not")
	assert.Contains(t, rerr.String(), "*** PANIC: not taken by the handler")
	assert.Empty(t, hout.String())
}

func TestFromSlogHandlerFatalAndPanic(t *testing.T) {
	fatalCalls := 0
	hout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	a, ok := FromSlogHandler(slog.NewTextHandler(hout, nil)).(*slogLgrAdapter)
	require.True(t, ok)
	a.stderr = rerr
	a.fatal = func() { fatalCalls++ }

	a.Logf("FATAL fromslog fatal")
	assert.Equal(t, 1, fatalCalls)
	assert.Contains(t, hout.String(), `level=ERROR msg="fromslog fatal"`)
	assert.Empty(t, rerr.String())

	hout.Reset()
	a.handler = ToSlogHandler(New(Out(hout), Secret("tok123")))
	a.Logf("[PANIC] failed with %s", "tok123")
	assert.Equal(t, 2, fatalCalls)
	assert.Contains(t, hout.String(), "ERROR failed with ******\n", "the handler gets the message and masks it")
	assert.True(t, strings.HasPrefix(rerr.String(), "\n*** PANIC\n\ngoroutine "), rerr.String())
	assert.NotContains(t, rerr.String(), "tok123", "stderr gets the heading and the dump, not the message")

	hout.Reset()
	a.handler = slog.NewTextHandler(hout, &slog.HandlerOptions{Level: slog.LevelError + 4})
	a.Logf("FATAL not taken by the handler")
	assert.Equal(t, 3, fatalCalls, "FATAL exits whether the handler takes the record or not")
	assert.Empty(t, hout.String())
}

func TestFatalfLevel(t *testing.T) {
	var fatal int
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	Setup(Out(rout), Err(rerr), CallerFile, CallerFunc)
	def.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	def.fatal = func() { fatal++ }
	defer Setup()

	_, _, line, _ := runtime.Caller(0)
	Fatalf("no prefix %s", "here")
	want := fmt.Sprintf("2018/01/07 13:02:34 FATAL {%s/levels_test.go:%d lgr.TestFatalfLevel} no prefix here\n", testDir, line+1)
	assert.Equal(t, want, rout.String(), "a message without a level is FATAL, with the caller of Fatalf")
	assert.Equal(t, want, rerr.String(), "FATAL is also written to the err writer")
	assert.Equal(t, 1, fatal)

	rout.Reset()
	rerr.Reset()
	Fatalf("INFO with a level")
	assert.Contains(t, rout.String(), "INFO  {")
	assert.Empty(t, rerr.String())
	assert.Equal(t, 2, fatal)

	rout.Reset()
	Fatalf("FATAL with the fatal level")
	assert.Contains(t, rout.String(), "FATAL {")
	assert.Equal(t, 3, fatal, "exits once")
}
