package main

import "github.com/go-pkgz/lgr"

// Logger defines application's logger interface. Note - it doesn't introduce any dependency on lgr
// and can be replaced with anything providing Logf function
type Logger interface {
	Logf(format string, args ...interface{})
}

func main() {
	l := lgr.New(lgr.Format(lgr.FullDebug)) // create lgr instance
	logConsumer(l)                          // pass logger to consumer
	// out: 2019/04/01 02:43:20.590 INFO  (_example/main.go:31 main.logConsumer) test 12345

	l2 := lgr.New(lgr.Debug, lgr.Format(lgr.ShortDebug)) // create lgr instance, debug enabled
	logConsumer(l2)                                      // pass logger to consumer
	// out:
	// 2019/04/01 02:43:20.591 INFO  (_example/main.go:31) test 12345
	// 2019/04/01 02:43:20.591 DEBUG (_example/main.go:32) something

	// define custom output format
	format := lgr.Format(`{{.Level}} - {{.DT.Format "2006-01-02T15:04:05Z07:00"}} - {{.CallerPkg}} - {{.Message}}`)
	l3 := lgr.New(format)
	logConsumer(l3)
	// out: INFO  - 2019-04-02T01:21:33-05:00 - _example - test 12345

	logWithGlobal() // logging with default global logger
	// out: 2019/04/01 02:43:20 WARN  test 9876543

	lgr.Setup(lgr.Msec, lgr.LevelBraces) // change settings of global logger
	logWithGlobal()                      // logging with modified global logger
	// out: 2019/04/01 02:43:20.591 [WARN]  test 9876543
}

// consumer example with Logger passed in
func logConsumer(l Logger) {
	l.Logf("INFO test 12345")
	l.Logf("DEBUG something") // will be printed for logger with Debug enabled
}

func logWithGlobal() {
	lgr.Printf("WARN test 9876543") // print to default logger
}
