/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger is a wrapper around slog.Logger with a predefined log level.
type Logger struct {
	Logger *slog.Logger
	Level  slog.Level
}

var (
	// Supported log levels
	Levels = []string{"debug", "info", "warn", "error"}
	// Current log level
	level slog.Level
)

// NewLogger creates a new Logger with the specified log level.
func NewLogger(levelStr string) (*Logger, error) {
	err := level.UnmarshalText([]byte(levelStr))
	if err != nil {
		return nil, fmt.Errorf("unknown log level: %q", levelStr)
	}

	o := &slog.HandlerOptions{Level: level}
	if level == slog.LevelDebug {
		o.AddSource = true
	}

	return &Logger{
		Logger: slog.New(slog.NewJSONHandler(os.Stderr, o)),
		Level:  level,
	}, nil
}

// Level returns the current log level.
func Level() slog.Level {
	return level
}
