/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	slogjournal "github.com/tschaefer/slog-journal"
)

func targetJournal(t *testing.T) {
	journal := &Journal{
		Enable: true,
	}
	handler, err := journal.TargetJournal(&slog.HandlerOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, handler)
	assert.IsType(t, &slogjournal.JournalHandler{}, handler)
}

func TestSinkTargetJournal(t *testing.T) {
	t.Run("TargetJournal returns valid handler", targetJournal)
}
