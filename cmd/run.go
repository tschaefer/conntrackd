/*
Copyright (c) Tobias Schäfer. All rights reserved.
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
	"github.com/spf13/viper"
	"github.com/tschaefer/conntrackd/internal/config"
	"github.com/tschaefer/conntrackd/internal/filter"
	"github.com/tschaefer/conntrackd/internal/geoip"
	"github.com/tschaefer/conntrackd/internal/logger"
	"github.com/tschaefer/conntrackd/internal/profiler"
	"github.com/tschaefer/conntrackd/internal/service"
	"github.com/tschaefer/conntrackd/internal/sink"
)

var cfgFile string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the conntrackd service",
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := config.InitConfig(cfgFile); err != nil {
			cobra.CheckErr(fmt.Sprintf("Failed to initialize configuration: %v", err))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("profiler.enable") {
			profiler := profiler.NewProfiler(viper.GetString("profiler.address"))
			err := profiler.Start()
			if err != nil {
				cobra.CheckErr(fmt.Sprintf("Failed to start profiler: %v", err))
			}
			defer func() {
				_ = profiler.Stop()
			}()
		}

		l, err := logger.NewLogger(viper.GetString("log.level"))
		if err != nil {
			cobra.CheckErr(fmt.Sprintf("Failed to create logger: %v", err))
		}

		var g *geoip.GeoIP
		geoipDatabase := viper.GetString("geoip.database")
		if geoipDatabase != "" {
			g, err = geoip.NewGeoIP(geoipDatabase)
			if err != nil {
				cobra.CheckErr(fmt.Sprintf("Failed to open geoip database: %v", err))
			}
			defer func() {
				_ = g.Close()
			}()
		}

		var f *filter.Filter
		filterRules := viper.GetStringSlice("filter")
		if len(filterRules) > 0 {
			f, err = filter.NewFilter(filterRules)
			if err != nil {
				cobra.CheckErr(fmt.Sprintf("failed to compile filter rules: %v", err))
			}
		}

		s, err := sink.NewSink(getSinkConfig())
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

func getSinkConfig() *sink.Config {
	return &sink.Config{
		Journal: sink.Journal{
			Enable: viper.GetBool("sink.journal.enable"),
		},
		Syslog: sink.Syslog{
			Enable:  viper.GetBool("sink.syslog.enable"),
			Address: viper.GetString("sink.syslog.address"),
		},
		Loki: sink.Loki{
			Enable:  viper.GetBool("sink.loki.enable"),
			Address: viper.GetString("sink.loki.address"),
			Labels:  viper.GetStringSlice("sink.loki.labels"),
		},
		Stream: sink.Stream{
			Enable: viper.GetBool("sink.stream.enable"),
			Writer: viper.GetString("sink.stream.writer"),
		},
	}
}

func init() {
	runCmd.CompletionOptions.SetDefaultShellCompDirective(cobra.ShellCompDirectiveNoFileComp)

	runCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is /etc/conntrackd/conntrackd.{yaml,json,toml})")

	runCmd.Flags().StringArray("filter", nil, "Filter rules in CEL format (repeatable, first-match wins)")
	_ = viper.BindPFlag("filter", runCmd.Flags().Lookup("filter"))

	runCmd.Flags().String("log.level", "info", fmt.Sprintf("Log level (%s)", strings.Join(logger.Levels, ", ")))
	_ = viper.BindPFlag("log.level", runCmd.Flags().Lookup("log.level"))
	_ = runCmd.RegisterFlagCompletionFunc("log.level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return logger.Levels, cobra.ShellCompDirectiveNoFileComp
	})

	runCmd.Flags().String("geoip.database", "", "Path to GeoIP database")
	_ = viper.BindPFlag("geoip.database", runCmd.Flags().Lookup("geoip.database"))
	_ = runCmd.RegisterFlagCompletionFunc("geoip.database", cobra.FixedCompletions(nil, cobra.ShellCompDirectiveDefault))

	runCmd.Flags().Bool("sink.journal.enable", false, "Enable journald sink")
	_ = viper.BindPFlag("sink.journal.enable", runCmd.Flags().Lookup("sink.journal.enable"))

	runCmd.Flags().Bool("sink.syslog.enable", false, "Enable syslog sink")
	_ = viper.BindPFlag("sink.syslog.enable", runCmd.Flags().Lookup("sink.syslog.enable"))

	runCmd.Flags().String("sink.syslog.address", "udp://localhost:514", "Syslog address")
	_ = viper.BindPFlag("sink.syslog.address", runCmd.Flags().Lookup("sink.syslog.address"))

	runCmd.Flags().Bool("sink.loki.enable", false, "Enable Loki sink")
	_ = viper.BindPFlag("sink.loki.enable", runCmd.Flags().Lookup("sink.loki.enable"))

	runCmd.Flags().String("sink.loki.address", "http://localhost:3100", "Loki address")
	_ = viper.BindPFlag("sink.loki.address", runCmd.Flags().Lookup("sink.loki.address"))

	runCmd.Flags().StringSlice("sink.loki.labels", nil, "Additional labels for Loki sink in key=value format")
	_ = viper.BindPFlag("sink.loki.labels", runCmd.Flags().Lookup("sink.loki.labels"))

	runCmd.Flags().Bool("sink.stream.enable", false, "Enable stream sink")
	_ = viper.BindPFlag("sink.stream.enable", runCmd.Flags().Lookup("sink.stream.enable"))

	runCmd.Flags().String("sink.stream.writer", "stdout", fmt.Sprintf("Stream writer (%s)", strings.Join(sink.StreamWriters, ", ")))
	_ = viper.BindPFlag("sink.stream.writer", runCmd.Flags().Lookup("sink.stream.writer"))
	_ = runCmd.RegisterFlagCompletionFunc("sink.stream.writer", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return sink.StreamWriters, cobra.ShellCompDirectiveNoFileComp
	})

	runCmd.Flags().Bool("profiler.enable", false, "Enable profiler")
	_ = viper.BindPFlag("profiler.enable", runCmd.Flags().Lookup("profiler.enable"))

	runCmd.Flags().String("profiler.address", "http://localhost:4040", "Profiler server address")
	_ = viper.BindPFlag("profiler.address", runCmd.Flags().Lookup("profiler.address"))

}
