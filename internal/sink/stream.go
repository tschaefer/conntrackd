/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

type Stream struct {
	Enable bool
	Writer string
}

func (s *Stream) TargetStream(options *slog.HandlerOptions) (slog.Handler, error) {
	slog.Debug("Initializing stream sink.")

	switch s.Writer {
	case "stdout":
		return slog.NewJSONHandler(os.Stdout, options), nil
	case "stderr":
		return slog.NewJSONHandler(os.Stderr, options), nil
	case "discard":
		return slog.NewJSONHandler(io.Discard, options), nil
	}

	return nil, fmt.Errorf("invalid stream writer specified: %q", s.Writer)
}
