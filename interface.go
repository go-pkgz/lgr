package lgr

import stdlog "log"

// L defines minimal interface used to log things
type L interface {
	Logf(format string, args ...interface{})
}

// Func type is an adapter to allow the use of ordinary functions as Logger.
type Func func(format string, args ...interface{})

// Logf calls f(id)
func (f Func) Logf(format string, args ...interface{}) { f(format, args...) }

// NoOp logger
var NoOp = Func(func(format string, args ...interface{}) {})

// Std logger
var Std = Func(func(format string, args ...interface{}) { stdlog.Printf(format, args...) })

// Printf simplifies replacement of std logger
func Printf(format string, args ...interface{}) {
	Std.Logf(format, args...)
}
