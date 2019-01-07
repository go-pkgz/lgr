# lgr - simple logger with basic levels [![Build Status](https://travis-ci.org/go-pkgz/lgr.svg?branch=master)](https://travis-ci.org/go-pkgz/lgr) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/lgr/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/lgr?branch=master)

## install

`go get github/go-pkgz/lgr`

## usage

```go
    l := lgr.New(lgr.Debug) // allow debug
    l.Logf("INFO some important err message, %v", err)
    l.Logf("DEBUG some less important err message, %v", err)
```

## details

TODO: interface, options, panics