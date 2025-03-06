package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/go-pkgz/lgr"
)

func main() {
	// example 1: Using lgr with slog
	out1()

	// example 2: Using slog handlers with lgr
	out2()

	// example 3: Direct slog integration in lgr
	out3()
}

// Example 1: Using lgr with slog
func out1() {
	println("\n--- Example 1: Using lgr with slog ---")

	// create lgr logger
	lgrLogger := lgr.New(lgr.Debug, lgr.Msec)

	// convert to slog handler and create slog logger
	handler := lgr.ToSlogHandler(lgrLogger)
	logger := slog.New(handler)

	// use standard slog API with lgr formatting
	logger.Debug("debug message", "requestID", "123", "user", "john")
	logger.Info("info message", "duration", 42*time.Millisecond)
	logger.Warn("warn message", "status", 429)
	logger.Error("error message", "error", "connection refused")
}

// Example 2: Using slog handlers with lgr
func out2() {
	println("\n--- Example 2: Using slog handlers with lgr ---")

	// create JSON slog handler
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// wrap it with lgr interface
	logger := lgr.FromSlogHandler(jsonHandler)

	// use lgr API with slog JSON backend
	logger.Logf("DEBUG debug message")
	logger.Logf("INFO info message with %s", "parameters")
	logger.Logf("WARN warning message")
	logger.Logf("ERROR error occurred: %v", "database connection failed")
}

// Example 3: Direct slog integration in lgr
func out3() {
	println("\n--- Example 3: Direct slog integration in lgr ---")

	// create a text handler
	textHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// create a logger that uses slog directly
	logger := lgr.New(lgr.SlogHandler(textHandler), lgr.Debug)

	// use lgr API with slog text backend
	logger.Logf("DEBUG debug message")
	logger.Logf("INFO structured logging with %s", "slog")
	logger.Logf("WARN this is a warning")
	logger.Logf("ERROR something bad happened: %v", "timeout")
}
