/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package service

import (
	"context"
	"log/slog"
	"syscall"

	"github.com/mdlayher/netlink"
	"github.com/ti-mo/conntrack"
	"github.com/ti-mo/netfilter"
	"github.com/tschaefer/conntrackd/internal/filter"
	"github.com/tschaefer/conntrackd/internal/geoip"
	"github.com/tschaefer/conntrackd/internal/logger"
	"github.com/tschaefer/conntrackd/internal/record"
	"github.com/tschaefer/conntrackd/internal/sink"
	"github.com/tschaefer/conntrackd/internal/version"
	"golang.org/x/sync/errgroup"
)

// Service represents the conntrack service.
type Service struct {
	Filter *filter.Filter
	GeoIP  *geoip.GeoIP
	Sink   *sink.Sink
	Logger *slog.Logger
}

// NewService creates a new conntrack service.
func NewService(logger *logger.Logger, geoip *geoip.GeoIP, filter *filter.Filter, sink *sink.Sink) (*Service, error) {
	slog.SetDefault(logger.Logger)

	return &Service{
		Filter: filter,
		GeoIP:  geoip,
		Sink:   sink,
		Logger: logger.Logger,
	}, nil
}

// Run starts the conntrack listener service.
func (s *Service) Run(ctx context.Context) bool {
	slog.Info("Starting conntrack listener.",
		"release", version.Release(), "commit", version.Commit(),
	)

	con, err := s.setupConntrack()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	evCh, errCh, err := s.startEventListener(con)
	if err != nil {
		_ = con.Close()
		cancel()
		return false
	}

	g := s.startEventProcessor(ctx, evCh)

	return s.handleShutdown(ctx, cancel, con, g, errCh)
}

// setupConntrack initializes the conntrack connection and sets options.
func (s *Service) setupConntrack() (*conntrack.Conn, error) {
	con, err := conntrack.Dial(nil)
	if err != nil {
		slog.Error("Failed to dial conntrack.", "error", err)
		return nil, err
	}

	if err := con.SetOption(netlink.ListenAllNSID|netlink.NoENOBUFS, true); err != nil {
		_ = con.Close()
		slog.Error("Failed to set conntrack listen options.", "error", err)
		return nil, err
	}

	return con, nil
}

// startEventListener starts listening for conntrack events.
func (s *Service) startEventListener(con *conntrack.Conn) (chan conntrack.Event, chan error, error) {
	evCh := make(chan conntrack.Event, 1024)
	errCh, err := con.Listen(evCh, 4, netfilter.GroupsCT)
	if err != nil {
		slog.Error("Failed to listen to conntrack events.", "error", err)
		return nil, nil, err
	}

	return evCh, errCh, nil
}

// startEventProcessor starts the event processing goroutine.
func (s *Service) startEventProcessor(ctx context.Context, evCh chan conntrack.Event) *errgroup.Group {
	var g errgroup.Group
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case event, ok := <-evCh:
				if !ok {
					return nil
				}
				go s.processEvent(event)
			}
		}
	})
	return &g
}

// processEvent processes a single conntrack event.
func (s *Service) processEvent(event conntrack.Event) {
	// Only process TCP and UDP events, ignore all other protocols (ICMP, etc.)
	protocol := event.Flow.TupleOrig.Proto.Protocol
	if protocol != syscall.IPPROTO_TCP && protocol != syscall.IPPROTO_UDP {
		return
	}

	shouldRecord := true
	if s.Filter != nil {
		_, shouldLog, _ := s.Filter.Evaluate(event)
		shouldRecord = shouldLog
	}

	if shouldRecord {
		record.Record(event, s.GeoIP, s.Sink.Logger)
	}
}

// handleShutdown manages graceful shutdown of the service.
func (s *Service) handleShutdown(ctx context.Context, cancel context.CancelFunc, con *conntrack.Conn, g *errgroup.Group, errCh chan error) bool {
	select {
	case err := <-errCh:
		if err != nil {
			cancel()
			slog.Error("Conntrack listener error.", "error", err)
			_ = con.Close()
			if gErr := g.Wait(); gErr != nil {
				slog.Error("Event loop returned error during shutdown.", "error", gErr)
			}
			return false
		}
		cancel()
		_ = con.Close()
		if gErr := g.Wait(); gErr != nil {
			slog.Error("Event loop returned error during shutdown.", "error", gErr)
		}
		return true
	case <-ctx.Done():
		slog.Info("Shutting down conntrack listener.")
		cancel()
		_ = con.Close()
		if gErr := g.Wait(); gErr != nil {
			slog.Error("Event loop returned error during shutdown.", "error", gErr)
		}
		return true
	}
}
