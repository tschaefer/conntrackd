/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tschaefer/conntrackd/internal/filter"
	"github.com/tschaefer/conntrackd/internal/geoip"
	"github.com/tschaefer/conntrackd/internal/logger"
	"github.com/tschaefer/conntrackd/internal/service"
	"github.com/tschaefer/conntrackd/internal/sink"
)

type Options struct {
	logLevel      string
	geoipDatabase string
	filterRules   []string
	sink          sink.Config
}

var options = Options{}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the conntrackd service",
	Run: func(cmd *cobra.Command, args []string) {
		l, err := logger.NewLogger(options.logLevel)
		if err != nil {
			cobra.CheckErr(fmt.Sprintf("Failed to create logger: %v", err))
		}

		var g *geoip.GeoIP
		if options.geoipDatabase != "" {
			g, err = geoip.NewGeoIP(options.geoipDatabase)
			if err != nil {
				cobra.CheckErr(fmt.Sprintf("Failed to open geoip database: %v", err))
			}
			defer func() {
				_ = g.Close()
			}()
		}

		var f *filter.Filter
		if len(options.filterRules) > 0 {
			f, err = filter.NewFilter(options.filterRules)
			if err != nil {
				cobra.CheckErr(fmt.Sprintf("failed to compile filter rules: %v", err))
			}
		}

		s, err := sink.NewSink(&options.sink)
		if err != nil {
			cobra.CheckErr(fmt.Sprintf("failed to initialize sink: %v", err))
		}

		service, err := service.NewService(l, g, f, s)
		cobra.CheckErr(err)

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		if tranquil := service.Run(ctx); !tranquil {
			os.Exit(1)
		}
	},
}

func init() {
	runCmd.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)

	runCmd.Flags().StringArrayVar(&options.filterRules, "filter", nil, "Filter rules in DSL format (repeatable, first-match wins)")

	runCmd.Flags().StringVar(&options.logLevel, "log.level", "info", fmt.Sprintf("Log level (%s)", strings.Join(logger.Levels, ", ")))
	_ = runCmd.RegisterFlagCompletionFunc("log.level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return logger.Levels, cobra.ShellCompDirectiveNoFileComp
	})

	runCmd.Flags().StringVar(&options.geoipDatabase, "geoip.database", "", "Path to GeoIP database")
	_ = runCmd.RegisterFlagCompletionFunc("geoip.database", cobra.FixedCompletions(nil, cobra.ShellCompDirectiveDefault))

	runCmd.Flags().BoolVar(&options.sink.Journal.Enable, "sink.journal.enable", false, "Enable journald sink")
	runCmd.Flags().BoolVar(&options.sink.Syslog.Enable, "sink.syslog.enable", false, "Enable syslog sink")
	runCmd.Flags().StringVar(&options.sink.Syslog.Address, "sink.syslog.address", "udp://localhost:514", "Syslog address")

	runCmd.Flags().BoolVar(&options.sink.Loki.Enable, "sink.loki.enable", false, "Enable Loki sink")
	runCmd.Flags().StringVar(&options.sink.Loki.Address, "sink.loki.address", "http://localhost:3100", "Loki address")
	runCmd.Flags().StringSliceVar(&options.sink.Loki.Labels, "sink.loki.labels", nil, "Additional labels for Loki sink in key=value format")

	runCmd.Flags().BoolVar(&options.sink.Stream.Enable, "sink.stream.enable", false, "Enable stream sink")
	runCmd.Flags().StringVar(&options.sink.Stream.Writer, "sink.stream.writer", "stdout", fmt.Sprintf("Stream writer (%s)", strings.Join(sink.StreamWriters, ", ")))
	_ = runCmd.RegisterFlagCompletionFunc("sink.stream.writer", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return sink.StreamWriters, cobra.ShellCompDirectiveNoFileComp
	})

}
