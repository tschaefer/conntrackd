/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package logger

import (
	"fmt"
	"log/slog"
	"os"
)

type Logger struct {
	Logger *slog.Logger
	Level  slog.Level
}

var (
	Levels = []string{"debug", "info", "warn", "error"}
	level  slog.Level
)

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

func Level() slog.Level {
	return level
}
