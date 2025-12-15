/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package service

import (
	"bytes"
	"context"
	"log/slog"
	"net/netip"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ti-mo/conntrack"
	"github.com/tschaefer/conntrackd/internal/filter"
	"github.com/tschaefer/conntrackd/internal/logger"
	"github.com/tschaefer/conntrackd/internal/sink"
)

func __setupSinkAndLogger(t *testing.T) (*sink.Sink, *logger.Logger, *bytes.Buffer) {
	var record bytes.Buffer
	target := slog.New(slog.NewTextHandler(&record, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger, err := logger.NewLogger("info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	sink := &sink.Sink{Logger: target}

	return sink, logger, &record
}

func __createEvent(proto uint8) conntrack.Event {
	flow := conntrack.NewFlow(
		proto,
		conntrack.StatusAssured,
		netip.MustParseAddr("9.0.0.1"),
		netip.MustParseAddr("7.8.8.8"),
		12344, 80,
		59, 0,
	)
	return conntrack.Event{Flow: &flow}
}

func newReturnsService(t *testing.T) {
	logger, err := logger.NewLogger("debug")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	svc, err := NewService(logger, nil, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, svc)
}

func processEventDoesNotRecordIfEventNotTCPorUDP(t *testing.T) {
	sink, logger, record := __setupSinkAndLogger(t)
	svc, err := NewService(logger, nil, nil, sink)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	event := __createEvent(syscall.IPPROTO_ICMP)
	svc.processEvent(event)
	assert.Len(t, record.String(), 0, "No log output expected for non-TCP/UDP event")
}

func processEventDoesRecordIfEventTCPorUDP(t *testing.T) {
	sink, logger, record := __setupSinkAndLogger(t)
	svc, err := NewService(logger, nil, nil, sink)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	event := __createEvent(syscall.IPPROTO_TCP)
	svc.processEvent(event)
	assert.Greater(t, len(record.String()), 0, "Log output expected for TCP event")

	record.Reset()
	event = __createEvent(syscall.IPPROTO_UDP)
	svc.processEvent(event)
	assert.Greater(t, len(record.String()), 0, "Log output expected for UDP event")
}

func processEventDoesNotRecordIfFilteredOut(t *testing.T) {
	sink, logger, record := __setupSinkAndLogger(t)
	filter, err := filter.NewFilter([]string{"drop any"})
	if err != nil {
		t.Fatalf("failed to create filter: %v", err)
	}
	svc, err := NewService(logger, nil, filter, sink)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	event := __createEvent(syscall.IPPROTO_TCP)
	svc.processEvent(event)
	assert.Len(t, record.String(), 0, "No log output expected for filtered out event")
}

func startEventProcessorStartsGoroutine(t *testing.T) {
	sink, logger, _ := __setupSinkAndLogger(t)
	svc, err := NewService(logger, nil, nil, sink)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	evCh := make(chan conntrack.Event)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := svc.startEventProcessor(ctx, evCh)
	assert.NotNil(t, g, "Errgroup expected to be returned")
}

func startEventProcessorDoesRecordOnEvent(t *testing.T) {
	sink, logger, record := __setupSinkAndLogger(t)
	svc, err := NewService(logger, nil, nil, sink)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	evCh := make(chan conntrack.Event)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := svc.startEventProcessor(ctx, evCh)

	event := __createEvent(syscall.IPPROTO_TCP)
	evCh <- event

	time.Sleep(100 * time.Millisecond)
	cancel()
	g.Wait()

	assert.Greater(t, len(record.String()), 0, "Log output expected for processed event")
}

func TestService(t *testing.T) {
	t.Run("service.New returns service", newReturnsService)
	t.Run("service.processEvent does not record if event not TCP or UDP", processEventDoesNotRecordIfEventNotTCPorUDP)
	t.Run("service.processEvent does record if event TCP or UDP", processEventDoesRecordIfEventTCPorUDP)
	t.Run("service.processEvent does not record if filtered out", processEventDoesNotRecordIfFilteredOut)
	t.Run("service.startEventProcessor starts goroutine", startEventProcessorStartsGoroutine)
	t.Run("service.startEventProcessor does record on event", startEventProcessorDoesRecordOnEvent)
}
