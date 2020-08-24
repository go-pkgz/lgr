package lgr

// Mapper defines optional functions to change elements of the logged message for each part, based on levels.
// Only some mapFunc can be defined, by default does nothing. Can be used to alter the output, for example making some
// part of the output colorful.
type Mapper struct {
	ErrorFunc mapFunc
	WarnFunc  mapFunc
	InfoFunc  mapFunc
	DebugFunc mapFunc

	CallerFunc mapFunc
	TimeFunc   mapFunc
}

type mapFunc func(string) string

// nopMapper is a default, doing nothing
var nopMapper = Mapper{
	ErrorFunc:  func(s string) string { return s },
	WarnFunc:   func(s string) string { return s },
	InfoFunc:   func(s string) string { return s },
	DebugFunc:  func(s string) string { return s },
	CallerFunc: func(s string) string { return s },
	TimeFunc:   func(s string) string { return s },
}
