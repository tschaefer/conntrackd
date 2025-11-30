/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func targetStream_WriterValid(t *testing.T) {
	for _, writer := range []string{"discard", "stdout", "stderr"} {
		stream := &Stream{
			Enable: true,
			Writer: writer,
		}
		handler, err := stream.TargetStream(&slog.HandlerOptions{})
		assert.Nil(t, err)
		assert.NotNil(t, handler)
		assert.IsType(t, &slog.JSONHandler{}, handler)
	}
}

func targetStream_WriterInvalid(t *testing.T) {
	stream := &Stream{
		Enable: true,
		Writer: "invalid-writer",
	}
	handler, err := stream.TargetStream(&slog.HandlerOptions{})
	assert.NotNil(t, err)
	assert.Nil(t, handler)
	assert.EqualError(t, err, "invalid stream writer specified: \"invalid-writer\"")
}

func TestSinkTargetStream(t *testing.T) {
	t.Run("TargetStream returns valid handler if writer valid", targetStream_WriterValid)
	t.Run("TargetStream returns error if writer invalid", targetStream_WriterInvalid)
}
