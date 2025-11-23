/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func capture(f func()) string {
	originalStderr := os.Stderr

	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	_ = w.Close()
	os.Stderr = originalStderr

	var buf = make([]byte, 5096)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func newReturnsError_NoTargetsEnabled(t *testing.T) {
	config := &Config{
		Journal: Journal{Enable: false},
		Syslog:  Syslog{Enable: false},
		Loki:    Loki{Enable: false},
		Stream:  Stream{Enable: false},
	}

	sink, err := NewSink(config)

	assert.Nil(t, sink)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "no target sink available")
}

func newReturnsSink_TargetsEnabled(t *testing.T) {
	config := &Config{
		Journal: Journal{Enable: false},
		Syslog:  Syslog{Enable: false},
		Loki:    Loki{Enable: false},
		Stream:  Stream{Enable: true, Writer: "discard"},
	}

	sink, err := NewSink(config)
	assert.NotNil(t, sink)
	assert.Nil(t, err)
	assert.IsType(t, &Sink{}, sink)
}

func newPrintsWarning_TargetInitFails(t *testing.T) {
	config := &Config{
		Journal: Journal{Enable: false},
		Syslog:  Syslog{Enable: false},
		Loki:    Loki{Enable: true, Address: "http://invalid-address"},
		Stream:  Stream{Enable: true, Writer: "discard"},
	}
	warning := capture(func() {
		_, _ = NewSink(config)
	})
	assert.Contains(t, warning, "Warning: Failed to initialize sink \"loki\"")
}

func TestSink(t *testing.T) {
	t.Run("NewSink returns error if no targets are enabled", newReturnsError_NoTargetsEnabled)
	t.Run("NewSink returns sink if targets enabled", newReturnsSink_TargetsEnabled)
	t.Run("NewSink prints warning if target init fails", newPrintsWarning_TargetInitFails)
}
