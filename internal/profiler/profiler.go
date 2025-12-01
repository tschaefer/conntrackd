/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package profiler

import (
	"log/slog"
	"runtime"

	"github.com/grafana/pyroscope-go"
	"github.com/tschaefer/conntrackd/internal/logger"
)

type Profiler interface {
	Start() error
	Stop() error
}

type profiler struct {
	instance *pyroscope.Profiler
	config   pyroscope.Config
	address  string
}

func NewProfiler(address string) Profiler {
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
	return &profiler{
		config:  cfg,
		address: address,
	}
}

func (p *profiler) Start() error {
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	profiler, err := pyroscope.Start(p.config)
	if err != nil {
		p.instance = nil
		return err
	}
	p.instance = profiler

	return nil
}

func (p *profiler) Stop() error {
	if p.instance == nil {
		return nil
	}

	return p.instance.Stop()
}
