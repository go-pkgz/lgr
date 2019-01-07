# lgr - simple logger with basic levels

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