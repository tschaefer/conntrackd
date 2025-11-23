/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package logger

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newReturnsError_UnknownLogLevel(t *testing.T) {
	_, err := NewLogger("unknown")
	assert.Error(t, err)
	assert.EqualError(t, err, `unknown log level: "unknown"`)
}

func newReturnsLogger_KnownLogLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		logger, err := NewLogger(level)
		assert.NoError(t, err)
		assert.NotNil(t, logger)
		assert.IsType(t, logger.Logger, &slog.Logger{})
		assert.Equal(t, strings.ToUpper(level), logger.Level.String())
	}
}

func levelReturnsCorrectLevel(t *testing.T) {
	for str, level := range map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	} {
		_, err := NewLogger(str)
		assert.NoError(t, err)

		assert.Equal(t, level, Level())
	}
}

func TestLogger(t *testing.T) {
	t.Run("New returns error for unknown log level", newReturnsError_UnknownLogLevel)
	t.Run("New returns logger for known log levels", newReturnsLogger_KnownLogLevels)
	t.Run("Level returns correct level", levelReturnsCorrectLevel)
}
