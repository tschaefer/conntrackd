/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package profiler

import (
	"log/slog"
	"runtime"

	"github.com/grafana/pyroscope-go"
	"github.com/tschaefer/conntrackd/internal/logger"
)

// Profiler is a wrapper around the pyroscope profiler.
type Profiler struct {
	Instance *pyroscope.Profiler
	Config   pyroscope.Config
}

// NewProfiler creates a new Profiler instance with the given server address.
func NewProfiler(address string) *Profiler {
	var pylogger pyroscope.Logger
	if logger.Level() == slog.LevelDebug {
		pylogger = pyroscope.StandardLogger
	} else {
		pylogger = nil
	}

	cfg := pyroscope.Config{
		ApplicationName: "github.com/tschaefer/conntrackd",
		ServerAddress:   address,
		Logger:          pylogger,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	}
	return &Profiler{
		Config: cfg,
	}
}

// Start starts the profiler.
func (p *Profiler) Start() error {
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	profiler, err := pyroscope.Start(p.Config)
	if err != nil {
		p.Instance = nil
		return err
	}
	p.Instance = profiler

	return nil
}

// Stop stops the profiler.
func (p *Profiler) Stop() error {
	if p.Instance == nil {
		return nil
	}

	return p.Instance.Stop()
}
