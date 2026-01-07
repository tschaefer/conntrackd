/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package logger

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newReturnsErrorIfLogLevelIsInvalid(t *testing.T) {
	_, err := NewLogger("unknown")
	assert.Error(t, err)
	assert.EqualError(t, err, `unknown log level: "unknown"`)
}

func newReturnsLoggerIfLogLevelIsValid(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		logger, err := NewLogger(level)
		assert.NoError(t, err)
		assert.NotNil(t, logger)
		assert.IsType(t, logger.Logger, &slog.Logger{})
		assert.Equal(t, strings.ToUpper(level), logger.Level.String())
	}
}

func levelReturnsCorrectLogLevel(t *testing.T) {
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
	t.Run("logger.New returns error if log level is invalid", newReturnsErrorIfLogLevelIsInvalid)
	t.Run("logger.New returns logger if log level is valid", newReturnsLoggerIfLogLevelIsValid)
	t.Run("logger.Level returns correct log level", levelReturnsCorrectLogLevel)
}
