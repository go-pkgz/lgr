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
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	buff := bytes.NewBufferString("")
	lg := Func(func(format string, args ...interface{}) {
		_, err := fmt.Fprintf(buff, format, args...)
		require.NoError(t, err)
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
	assert.Empty(t, buff.String())
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
	assert.Equal(t, "2018/01/07 13:02:34 INFO  something 123 xyz\n", buff.String())

	buff.Reset()
	Printf("[DEBUG] something 123 %s", "xyz")
	assert.Empty(t, buff.String())

	buff.Reset()
	Print("[WARN] something 123 % %% %3A%2F%")
	assert.Equal(t, "2018/01/07 13:02:34 WARN  something 123 % %% %3A%2F%\n", buff.String())
}

func TestDefaultWithSetup(t *testing.T) {
	buff := bytes.NewBuffer([]byte{})
	Setup(Out(buff), Debug, Format(FullDebug))
	def.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	Printf("[DEBUG] something 123 %s", "xyz")
	assert.Equal(t, "2018/01/07 13:02:34.000 DEBUG ("+testDir+"/interface_test.go:74 lgr.TestDefaultWithSetup) something 123 xyz\n",
		buff.String())
}

func TestDefaultFuncWithSetup(t *testing.T) {
	buff := bytes.NewBuffer([]byte{})
	Setup(Out(buff), Debug, Format(FullDebug))
	def.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	Default().Logf("[INFO] something 123 %s", "xyz")
	assert.Equal(t, "2018/01/07 13:02:34.000 INFO  ("+testDir+"/interface_test.go:83 lgr."+
		"TestDefaultFuncWithSetup) something 123 xyz\n", buff.String())
}

func TestDefaultFatal(t *testing.T) {
	var fatal int
	buff := bytes.NewBuffer([]byte{})
	Setup(Out(buff), Format(Short))
	def.stdout = buff
	def.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	def.fatal = func() { fatal++ }
	defer func() {
		def.stdout = os.Stdout
		def.now = time.Now
	}()

	Fatalf("ERROR something 123 %s", "xyz")
	assert.Equal(t, "2018/01/07 13:02:34 ERROR something 123 xyz\n", buff.String())
	assert.Equal(t, 1, fatal)
}

func TestAddRemoveSecrets(t *testing.T) {
	buff := bytes.NewBuffer([]byte{})
	Setup(Out(buff), Err(buff), Format(Short), Secret("own-secret"))
	def.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	defer Setup()
	defer RemoveSecrets("token1", "token2", "own-secret-long")

	AddSecrets("token1", "token2", "", " ")
	Printf("INFO own-secret token1 token2")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  ****** ****** ******\n", buff.String())

	buff.Reset()
	Default().Logf("INFO own-secret token1 token2")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  ****** ****** ******\n", buff.String())

	buff.Reset()
	AddSecrets("own-secret-long")
	Printf("INFO own-secret-long")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  ******\n", buff.String(), "longer package secret wins over the logger's own")

	buff.Reset()
	RemoveSecrets("token1", "own-secret")
	Printf("INFO own-secret token1 token2")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  ****** token1 ******\n", buff.String(), "package remove keeps the logger's own secret")
}

func TestAddSecretsSurvivesSetup(t *testing.T) {
	AddSecrets("token1")
	defer RemoveSecrets("token1")
	defer Setup()

	buff := bytes.NewBuffer([]byte{})
	Setup(Out(buff), Err(buff), Format(Short))
	def.now = func() time.Time { return time.Date(2018, 1, 7, 13, 2, 34, 0, time.Local) }
	Printf("INFO with token1")
	assert.Equal(t, "2018/01/07 13:02:34 INFO  with ******\n", buff.String())
}

func TestAddSecretsWhilePrinting(t *testing.T) {
	rout := &syncBuffer{}
	Setup(Out(rout), Err(rout))
	defer Setup()
	defer RemoveSecrets("kept")
	AddSecrets("kept")

	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 500; j++ {
				Printf("INFO kept %d", j)
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 4; i++ {
		go func(i int) {
			for j := 0; j < 200; j++ {
				v := fmt.Sprintf("tmp-%d-%d", i, j)
				AddSecrets(v, "kept")
				RemoveSecrets(v)
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 12; i++ {
		<-done
	}
	assert.NotContains(t, rout.String(), "kept")
	assert.Equal(t, []string{"kept"}, secretStrings(sharedSecrets.list()))
}
