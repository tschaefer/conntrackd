/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"errors"
	"log/slog"

	slogmulti "github.com/samber/slog-multi"
)

type Sink struct {
	Journal Journal
	Syslog  Syslog
	Loki    Loki
}

type SinkTarget func(*slog.HandlerOptions) (slog.Handler, error)

func (s *Sink) Initialize() (*slog.Logger, error) {
	slog.Debug("Initializing sink targets.", "data", s)

	options := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	var handlers []slog.Handler
	if s.Journal.Enable {
		handler, err := s.Journal.TargetJournal(options)
		if err != nil {
			slog.Warn("Failed to initialize journal sink", "error", err)
		} else {
			handlers = append(handlers, handler)
		}
	}

	if s.Syslog.Enable {
		handler, err := s.Syslog.TargetSyslog(options)
		if err != nil {
			slog.Warn("Failed to initialize syslog sink", "error", err)
		} else {
			handlers = append(handlers, handler)
		}
	}

	if s.Loki.Enable {
		handler, err := s.Loki.TargetLoki(options)
		if err != nil {
			slog.Warn("Failed to initialize loki sink", "error", err)
		} else {
			handlers = append(handlers, handler)
		}
	}

	if len(handlers) == 0 {
		return nil, errors.New("no target sink available")
	}

	return slog.New(slogmulti.Fanout(handlers...)), nil
}
