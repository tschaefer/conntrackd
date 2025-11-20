/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package service

import (
	"context"
	"log/slog"
	"os/signal"
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

type Service struct {
	Filter filter.Filter
	Logger logger.Logger
	GeoIP  geoip.GeoIP
	Sink   sink.Sink
}

func (s *Service) handler(geo *geoip.Reader, sink *slog.Logger) error {
	con, err := conntrack.Dial(nil)
	if err != nil {
		slog.Error("Failed to dial conntrack.", "error", err)
		return err
	}

	evCh := make(chan conntrack.Event, 1024)
	errCh, err := con.Listen(evCh, 4, netfilter.GroupsCT)
	if err != nil {
		_ = con.Close()
		slog.Error("Failed to listen to conntrack events.", "error", err)
		return err
	}

	if err := con.SetOption(netlink.ListenAllNSID|netlink.NoENOBUFS, true); err != nil {
		_ = con.Close()
		slog.Error("Failed to set conntrack listen options.", "error", err)
		return err
	}
	defer func() {
		_ = con.Close()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
				go func() {
					if !s.Filter.Apply(event) {
						record.Record(event, geo, sink)
					}
				}()
			}
		}
	})

	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("Conntrack listener error.", "error", err)
			stop()
			_ = con.Close()
			if gErr := g.Wait(); gErr != nil {
				slog.Error("Event loop returned error during shutdown.", "error", gErr)
			}
			return err
		}
		stop()
		_ = con.Close()
		if gErr := g.Wait(); gErr != nil {
			slog.Error("Event loop returned error during shutdown.", "error", gErr)
		}
		return nil
	case <-ctx.Done():
		slog.Info("Shutting down conntrack listener.")
		_ = con.Close()
		if gErr := g.Wait(); gErr != nil {
			slog.Error("Event loop returned error during shutdown.", "error", gErr)
		}
		return nil
	}
}

func (s *Service) Run() error {
	if err := s.Logger.Initialize(); err != nil {
		slog.Error("Failed to initialize logger.", "error", err)
		return err
	}

	slog.Debug("Running Service.", "data", s)

	slog.Info("Starting conntrack listener.",
		"release", version.Release(), "commit", version.Commit(),
		"filter", s.Filter,
		"geoip", s.GeoIP.Database,
	)

	sink, err := s.Sink.Initialize()
	if err != nil {
		slog.Error("Failed to initialize sink.", "error", err)
		return err
	}

	var geo *geoip.Reader
	if s.GeoIP.Database != "" {
		geo, err = geoip.Open(s.GeoIP.Database)
		if err != nil {
			slog.Error("Failed to open geoip database.", "error", err)
			return err
		}
		defer func() {
			_ = geo.Close()
		}()
	}

	return s.handler(geo, sink)
}
