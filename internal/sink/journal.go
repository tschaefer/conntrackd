/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"log/slog"

	slogjournal "github.com/tschaefer/slog-journal"
)

// Journal represents a systemd journal logging sink.
type Journal struct {
	Enable bool
}

// TargetJournal creates a sink target for systemd journal logging.
func (j *Journal) TargetJournal(options *slog.HandlerOptions) (slog.Handler, error) {
	slogjournal.FieldPrefix = "EVENT"
	o := &slogjournal.Option{
		Level: options.Level,
	}
	return o.NewJournalHandler(), nil
}
