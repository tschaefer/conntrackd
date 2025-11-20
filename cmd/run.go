/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package cmd

import (
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tschaefer/conntrackd/internal/service"
)

var srv = service.Service{}

var (
	validEventTypes    = []string{"NEW", "UPDATE", "DESTROY"}
	validProtocols     = []string{"TCP", "UDP"}
	validDestinations  = []string{"PUBLIC", "PRIVATE", "LOCAL", "MULTICAST"}
	validLogLevels     = []string{"trace", "debug", "info", "error"}
	validLogFormats    = []string{"text", "json"}
	validSyslogSchemes = []string{"udp", "tcp", "unix", "unixgram", "unixpacket"}
	validLokiSchemes   = []string{"http", "https"}
	validStreamWriters = []string{"stdout", "stderr", "discard"}
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the conntrackd service",
	Run: func(cmd *cobra.Command, args []string) {
		if !srv.Sink.Journal.Enable &&
			!srv.Sink.Syslog.Enable &&
			!srv.Sink.Loki.Enable &&
			!srv.Sink.Stream.Enable {
			cobra.CheckErr(fmt.Errorf("at least one sink must be enabled"))
		}

		err := validateStringFlag("sink.syslog.address", srv.Sink.Syslog.Address, []string{})
		cobra.CheckErr(err)

		err = validateStringFlag("sink.loki.address", srv.Sink.Loki.Address, []string{})
		cobra.CheckErr(err)

		err = validateStringFlag("sink.stream.writer", srv.Sink.Stream.Writer, validStreamWriters)
		cobra.CheckErr(err)

		err = validateStringSliceFlag("filter.include.types", srv.Filter.EventTypes, validEventTypes)
		cobra.CheckErr(err)

		err = validateStringSliceFlag("filter.include.protocols", srv.Filter.Protocols, validProtocols)
		cobra.CheckErr(err)

		err = validateStringSliceFlag("filter.include.destinations", srv.Filter.Destinations, validDestinations)
		cobra.CheckErr(err)

		err = validateStringSliceFlag("filter.exclude.addresses", srv.Filter.Addresses, []string{})
		cobra.CheckErr(err)

		err = validateStringFlag("service.log.level", srv.Logger.Level, validLogLevels)
		cobra.CheckErr(err)

		err = validateStringFlag("service.log.format", srv.Logger.Format, validLogFormats)
		cobra.CheckErr(err)

		if srv.GeoIP.Database != "" {
			if _, err := os.Stat(srv.GeoIP.Database); os.IsNotExist(err) {
				cobra.CheckErr(fmt.Errorf("GeoIP database file does not exist: %s", srv.GeoIP.Database))
			}
		}

		if err := srv.Run(); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	runCmd.Flags().StringSliceVar(&srv.Filter.EventTypes, "filter.include.types", nil, "Filter by event type (NEW,UPDATE,DESTROY)")
	runCmd.Flags().StringSliceVar(&srv.Filter.Protocols, "filter.include.protocols", nil, "Filter by protocol (TCP,UDP)")
	runCmd.Flags().StringSliceVar(&srv.Filter.Destinations, "filter.include.destinations", nil, "Filter by destination IPs (PUBLIC,PRIVATE,LOCAL,MULTICAST)")
	runCmd.Flags().StringSliceVar(&srv.Filter.Addresses, "filter.exclude.addresses", nil, "Exclude specific IP addresses")
	runCmd.Flags().StringVar(&srv.Logger.Format, "service.log.format", "", "Log format (text,json)")
	runCmd.Flags().StringVar(&srv.Logger.Level, "service.log.level", "", "Log level (debug,info)")
	runCmd.Flags().StringVar(&srv.GeoIP.Database, "geoip.database", "", "Path to GeoIP database")

	runCmd.Flags().BoolVar(&srv.Sink.Journal.Enable, "sink.journal.enable", false, "Enable journald sink")
	runCmd.Flags().BoolVar(&srv.Sink.Syslog.Enable, "sink.syslog.enable", false, "Enable syslog sink")
	runCmd.Flags().StringVar(&srv.Sink.Syslog.Address, "sink.syslog.address", "udp://localhost:514", "Syslog address")
	runCmd.Flags().BoolVar(&srv.Sink.Loki.Enable, "sink.loki.enable", false, "Enable Loki sink")
	runCmd.Flags().StringVar(&srv.Sink.Loki.Address, "sink.loki.address", "http://localhost:3100", "Loki address")
	runCmd.Flags().StringSliceVar(&srv.Sink.Loki.Labels, "sink.loki.labels", nil, "Additional labels for Loki sink in key=value format")
	runCmd.Flags().BoolVar(&srv.Sink.Stream.Enable, "sink.stream.enable", false, "Enable stream sink")
	runCmd.Flags().StringVar(&srv.Sink.Stream.Writer, "sink.stream.writer", "stdout", "Stream writer (stdout,stderr,discard)")

	_ = runCmd.RegisterFlagCompletionFunc("service.log.level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validLogLevels, cobra.ShellCompDirectiveNoFileComp
	})

	_ = runCmd.RegisterFlagCompletionFunc("service.log.format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validLogFormats, cobra.ShellCompDirectiveNoFileComp
	})

	_ = runCmd.RegisterFlagCompletionFunc("filter.include.types", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validEventTypes, cobra.ShellCompDirectiveNoFileComp
	})

	_ = runCmd.RegisterFlagCompletionFunc("filter.include.protocols", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validProtocols, cobra.ShellCompDirectiveNoFileComp
	})

	_ = runCmd.RegisterFlagCompletionFunc("filter.include.destinations", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validDestinations, cobra.ShellCompDirectiveNoFileComp
	})

	_ = runCmd.RegisterFlagCompletionFunc("sink.stream.writer", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validStreamWriters, cobra.ShellCompDirectiveNoFileComp
	})
}

func validateStringSliceFlag(flag string, values []string, validValues []string) error {
	if flag == "filter.exclude.addresses" {
		for _, v := range values {
			if _, err := netip.ParseAddr(v); err != nil {
				return fmt.Errorf("invalid IP address '%s' for '--%s'", v, flag)
			}
		}
		return nil
	}

	for _, v := range values {
		if !slices.Contains(validValues, v) {
			return fmt.Errorf("invalid value '%s' for '--%s' . Valid values are: %s", v, flag, validValues)
		}
	}
	return nil
}

func validateStringFlag(flag string, value string, validValues []string) error {
	if value == "" {
		return nil
	}

	if flag == "sink.syslog.address" || flag == "sink.loki.address" {
		url, err := url.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid URL '%s' for '--%s'", value, flag)
		}

		if flag == "sink.syslog.address" {
			if !slices.Contains(validSyslogSchemes, url.Scheme) {
				return fmt.Errorf("invalid URL scheme '%s' for '--%s'. Valid schemes are: udp, tcp, unix, unixgram unixpacket", url.Scheme, flag)
			}
			if url.Host == "" && !strings.HasPrefix(url.Scheme, "unix") {
				return fmt.Errorf("invalid URL '%s' for '--%s'. Host is missing", value, flag)
			}
			if url.Path == "" && strings.HasPrefix(url.Scheme, "unix") {
				return fmt.Errorf("invalid URL '%s' for '--%s'. Path is missing", value, flag)
			}
		}

		if flag == "sink.loki.address" {
			if !slices.Contains(validLokiSchemes, url.Scheme) {
				return fmt.Errorf("invalid URL scheme '%s' for '--%s'. Valid schemes are: http, https", url.Scheme, flag)
			}
			if url.Host == "" {
				return fmt.Errorf("invalid URL '%s' for '--%s'. Host is missing", value, flag)
			}
		}

		return nil
	}

	if !slices.Contains(validValues, value) {
		return fmt.Errorf("invalid value '%s' for '--%s' . Valid values are: %s", value, flag, validValues)
	}
	return nil
}
