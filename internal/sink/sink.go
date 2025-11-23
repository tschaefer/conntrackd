/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	slogmulti "github.com/samber/slog-multi"
)

const (
	ExitOnWarningEnv string = "CONNTRACKD_SINK_EXIT_ON_WARNING"
)

type Sink struct {
	Logger *slog.Logger
}

type Config struct {
	Journal Journal
	Syslog  Syslog
	Loki    Loki
	Stream  Stream
}

type SinkTarget func(*slog.HandlerOptions) (slog.Handler, error)

func NewSink(config *Config) (*Sink, error) {
	options := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	exitOnWarning := false
	envExitOnWarning, ok := os.LookupEnv(ExitOnWarningEnv)
	if ok && envExitOnWarning == "1" || envExitOnWarning == "true" {
		exitOnWarning = true
	}

	var handlers []slog.Handler

	targets := []struct {
		name    string
		enabled bool
		init    SinkTarget
	}{
		{"journal", config.Journal.Enable, config.Journal.TargetJournal},
		{"syslog", config.Syslog.Enable, config.Syslog.TargetSyslog},
		{"loki", config.Loki.Enable, config.Loki.TargetLoki},
		{"stream", config.Stream.Enable, config.Stream.TargetStream},
	}

	for _, t := range targets {
		if !t.enabled {
			continue
		}
		handler, err := t.init(options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize sink %q: %v\n", t.name, err)
			if exitOnWarning {
				os.Exit(1)
			}
			continue
		}
		handlers = append(handlers, handler)
	}

	if len(handlers) == 0 {
		return nil, errors.New("no target sink available")
	}

	return &Sink{Logger: slog.New(slogmulti.Fanout(handlers...))}, nil
}
