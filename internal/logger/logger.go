/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

type Logger struct {
	Format string
	Level  string
}

const (
	LevelTrace = slog.Level(-8)
)

var format string
var level slog.Level

func (l *Logger) Initialize() error {
	var logLevel slog.Level
	switch l.Level {
	case "trace":
		logLevel = LevelTrace
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "error":
		logLevel = slog.LevelError
	case "":
		logLevel = slog.LevelInfo
	default:
		return fmt.Errorf("unknown log level: %q", l.Level)
	}
	level = logLevel

	loggerOptions := &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				if a.Value.String() == "DEBUG-4" {
					a.Value = slog.StringValue("TRACE")
				}
			}
			return a
		},
	}
	if logLevel == LevelTrace {
		loggerOptions.AddSource = true
	}

	var logger *slog.Logger
	switch l.Format {
	case "json":
		logger = slog.New(slog.NewJSONHandler(os.Stderr, loggerOptions))
	case "text", "":
		logger = slog.New(slog.NewTextHandler(os.Stderr, loggerOptions))
	default:
		return fmt.Errorf("unknown log format: %q", l.Format)
	}
	format = l.Format

	slog.SetDefault(logger)

	return nil
}

func Trace(msg string, args ...any) {
	slog.Log(context.Background(), LevelTrace, msg, args...)
}

func Format() string {
	return format
}

func Level() slog.Level {
	return level
}
