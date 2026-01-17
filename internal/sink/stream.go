/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Stream represents a standard output stream sink.
type Stream struct {
	Enable bool
	Writer string
}

// Available stream writers
var StreamWriters = []string{"stdout", "stderr", "discard"}

// TargetStream creates a sink target for standard output streams.
func (s *Stream) TargetStream(options *slog.HandlerOptions) (slog.Handler, error) {
	writer := map[string]io.Writer{
		"stdout":  os.Stdout,
		"stderr":  os.Stderr,
		"discard": io.Discard,
	}

	if w, ok := writer[s.Writer]; ok {
		return slog.NewJSONHandler(w, options), nil
	}

	return nil, fmt.Errorf("invalid stream writer specified: %q", s.Writer)
}
