package lgr

import (
	"bytes"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdaptor_ToWriter(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr), Format(WithMsec))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	wr := ToWriter(l, "WARN")
	sz, err := wr.Write([]byte("something blah 123"))
	require.NoError(t, err)
	assert.Equal(t, 18, sz)
	assert.Equal(t, "2018/01/07 13:02:34.000 WARN  something blah 123\n", rout.String())
}

func TestAdaptor_ToWriterNoLevel(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr), Msec, LevelBraces)
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	wr := ToWriter(l, "")
	sz, err := wr.Write([]byte("something blah 123"))
	require.NoError(t, err)
	assert.Equal(t, 18, sz)
	assert.Equal(t, "2018/01/07 13:02:34.000 [INFO]  something blah 123\n", rout.String())

	rout.Reset()
	rerr.Reset()
	_, err = wr.Write([]byte("INFO something blah 123\n"))
	require.NoError(t, err)
	assert.Equal(t, "2018/01/07 13:02:34.000 [INFO]  something blah 123\n", rout.String())
}

func TestAdaptor_ToStdLogger(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	l := New(Out(rout), Err(rerr), Format(WithMsec))
	l.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }

	wr := ToStdLogger(l, "WARN")
	wr.Print("something\n")
	assert.Equal(t, "2018/01/07 13:02:34.000 WARN  something\n", rout.String())

	rout.Reset()
	rerr.Reset()
	wr.Printf("xxx %s", "yyy")
	assert.Equal(t, "2018/01/07 13:02:34.000 WARN  xxx yyy\n", rout.String())
}

func TestSetupStdLogger(t *testing.T) {
	rout, rerr := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})
	SetupStdLogger(Out(rout), Err(rerr), Format(WithMsec))
	log.Print("something\n")
	assert.Contains(t, rout.String(), " INFO  something\n")
	rout.Reset()

	log.Print("[WARN] something\n")
	assert.Contains(t, rout.String(), " WARN  something\n")
	rout.Reset()

	log.Print("[DEBUG] something\n")
	assert.Empty(t, rout.String())
}
