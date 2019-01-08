package lgr

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	buff := bytes.NewBufferString("")
	lg := Func(func(format string, args ...interface{}) {
		fmt.Fprintf(buff, format, args...)
	})

	lg.Logf("blah %s %d something", "str", 123)
	assert.Equal(t, "blah str 123 something", buff.String())

	Std.Logf("blah %s %d something", "str", 123)
	Std.Logf("[DEBUG] auth failed, %s", errors.New("blah blah"))
}

func TestStd(t *testing.T) {
	buff := bytes.NewBufferString("")
	log.SetOutput(buff)
	defer log.SetOutput(os.Stdout)

	Std.Logf("blah %s %d something", "str", 123)
	assert.True(t, strings.HasSuffix(buff.String(), "blah str 123 something\n"), buff.String())
}

func TestNoOp(t *testing.T) {
	buff := bytes.NewBufferString("")
	log.SetOutput(buff)
	defer log.SetOutput(os.Stdout)

	NoOp.Logf("blah %s %d something", "str", 123)
	assert.Equal(t, "", buff.String())
}

func TestDefault(t *testing.T) {
	buff := bytes.NewBuffer([]byte{})
	def.stdout = buff
	def.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	defer func() {
		def.stdout = os.Stdout
		def.now = time.Now
	}()

	Printf("[INFO] something 123 %s", "xyz")
	assert.Equal(t, "2018/01/07 13:02:34.000 INFO  something 123 xyz\n", buff.String())

	buff.Reset()
	Printf("[DEBUG] something 123 %s", "xyz")
	assert.Equal(t, "", buff.String())

	buff.Reset()
	Print("[WARN] something 123")
	assert.Equal(t, "2018/01/07 13:02:34.000 WARN  something 123\n", buff.String())
}

func TestDefaultWithSetup(t *testing.T) {
	buff := bytes.NewBuffer([]byte{})
	Setup(Out(buff), Debug, CallerFile, CallerFunc)
	def.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	Printf("[INFO] something 123 %s", "xyz")
	assert.Equal(t, "2018/01/07 13:02:34.000 INFO  {lgr/interface_test.go:72 lgr.TestDefaultWithSetup} something 123 xyz\n", buff.String())
}
