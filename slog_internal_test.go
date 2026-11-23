package lgr

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
