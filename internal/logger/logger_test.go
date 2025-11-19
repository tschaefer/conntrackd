/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Logger(t *testing.T) {
	testees := []Logger{
		{Format: "json", Level: "trace"},
		{Format: "json", Level: "debug"},
		{Format: "json", Level: "info"},
		{Format: "json", Level: "error"},
		{Format: "text", Level: "trace"},
		{Format: "text", Level: "debug"},
		{Format: "text", Level: "info"},
		{Format: "text", Level: "error"},
	}

	for _, logger := range testees {
		err := logger.Initialize()
		assert.NoError(t, err, "valid logger config")
	}

	logger := Logger{}
	err := logger.Initialize()
	assert.NoError(t, err, "default logger config")

	logger = Logger{Format: "xml", Level: "info"}
	err = logger.Initialize()
	assert.Errorf(t, err, "unknown log format: %q", "xml")

	logger = Logger{Format: "json", Level: "panic"}
	err = logger.Initialize()
	assert.Errorf(t, err, "unknown log level: %q", "error")

	logger = Logger{Format: "json", Level: "debug"}
	err = logger.Initialize()
	assert.NoError(t, err, "valid logger config")
	assert.Equal(t, "json", logger.Format, "logger format set by Initialize args")
	assert.Equal(t, "debug", logger.Level, "logger level set by Initialize args")
}
