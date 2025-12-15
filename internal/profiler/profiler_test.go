/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package profiler

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"testing"

	"github.com/grafana/pyroscope-go"
	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/conntrackd/internal/logger"
)

func __setupLogger(t *testing.T, level string) {
	logger, err := logger.NewLogger(level)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	slog.SetDefault(logger.Logger)
}

func __address(t *testing.T) string {
	for port := 4096; port <= 65535; port++ {
		c, _ := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if c != nil {
			c.Close()
			continue
		}
		return fmt.Sprintf("http://localhost:%d", port)
	}
	t.Fatalf("failed to find free port")

	return ""
}

func newReturnsProfiler(t *testing.T) {
	__setupLogger(t, "info")
	address := __address(t)

	profiler := NewProfiler(address)
	assert.NotNil(t, profiler)
	assert.Equal(t, address, profiler.Config.ServerAddress)
	assert.Nil(t, profiler.Config.Logger)
	assert.Nil(t, profiler.Instance)
	assert.Equal(t, "github.com/tschaefer/conntrackd", profiler.Config.ApplicationName)
}

func newSetsLoggerIfLogLevelIsDebug(t *testing.T) {
	__setupLogger(t, "debug")
	address := __address(t)

	profiler := NewProfiler(address)
	assert.NotNil(t, profiler)
	assert.Equal(t, address, profiler.Config.ServerAddress)
	assert.IsType(t, pyroscope.StandardLogger, profiler.Config.Logger)
	assert.Nil(t, profiler.Instance)
	assert.Equal(t, "github.com/tschaefer/conntrackd", profiler.Config.ApplicationName)
}

func startSetsInstance(t *testing.T) {
	__setupLogger(t, "info")
	address := __address(t)

	profiler := NewProfiler(address)
	err := profiler.Start()
	if err != nil {
		t.Fatalf("failed to start profiler: %v", err)
	}
	assert.NotNil(t, profiler.Instance)
	assert.IsType(t, &pyroscope.Profiler{}, profiler.Instance)
	defer profiler.Stop()
}

func startReturnsErrorIfAddressIsInvalid(t *testing.T) {
	__setupLogger(t, "info")
	address := "http://invalid:address"

	profiler := NewProfiler(address)
	err := profiler.Start()
	assert.NotNil(t, err)
	assert.Nil(t, profiler.Instance)
	assert.Error(t, err)
	assert.IsType(t, &url.Error{}, err)
}

func TestProfiler(t *testing.T) {
	t.Run("profiler.New returns Profiler", newReturnsProfiler)
	t.Run("profiler.New sets logger if log level is debug", newSetsLoggerIfLogLevelIsDebug)
	t.Run("profiler.Start sets Instance", startSetsInstance)
	t.Run("profiler.Start returns error if address is invalid", startReturnsErrorIfAddressIsInvalid)
}
