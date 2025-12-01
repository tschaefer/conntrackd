/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package record

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"maps"
	"net/netip"
	"os"
	"slices"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ti-mo/conntrack"
	"github.com/tschaefer/conntrackd/internal/geoip"
)

var geoDatabasePath string

var log bytes.Buffer

func setupLogger() *slog.Logger {
	loggerOptions := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "type"
			}
			return a
		},
	}
	return slog.New(slog.NewJSONHandler(&log, loggerOptions))
}

func recordLogsBasicDataIfNoLocationIsGiven(t *testing.T) {
	logger := setupLogger()

	flow := conntrack.NewFlow(
		syscall.IPPROTO_TCP,
		conntrack.StatusAssured,
		netip.MustParseAddr("10.19.80.100"), netip.MustParseAddr("78.47.60.169"),
		4711, 443,
		60, 0,
	)

	event := conntrack.Event{
		Type: conntrack.EventNew,
		Flow: &flow,
	}

	var geo *geoip.GeoIP
	Record(event, geo, logger)
	var result map[string]any
	err := json.Unmarshal(log.Bytes(), &result)
	assert.NoError(t, err)

	wanted := []string{"level", "time",
		"type", "flow", "prot",
		"src_addr", "dst_addr", "src_port", "dst_port"}
	got := slices.Sorted(maps.Keys(result))
	assert.ElementsMatch(t, wanted, got, "record basic keys")

	log.Reset()
}

func recordLogsAllDataIfLocationIsGiven(t *testing.T) {
	logger := setupLogger()

	flow := conntrack.NewFlow(
		syscall.IPPROTO_TCP,
		conntrack.StatusAssured,
		netip.MustParseAddr("78.47.60.169"), netip.MustParseAddr("78.47.60.169"),
		4711, 443,
		60, 0,
	)

	event := conntrack.Event{
		Type: conntrack.EventNew,
		Flow: &flow,
	}

	geo, err := geoip.NewGeoIP(geoDatabasePath)
	assert.NoError(t, err)
	defer func() {
		_ = geo.Close()
	}()

	Record(event, geo, logger)
	var result map[string]any
	err = json.Unmarshal(log.Bytes(), &result)
	assert.NoError(t, err)

	wanted := []string{"level", "time",
		"type", "flow", "prot",
		"src_addr", "dst_addr", "src_port", "dst_port",
		"dst_country", "dst_city", "dst_lat", "dst_lon",
		"src_country", "src_city", "src_lat", "src_lon"}
	got := slices.Sorted(maps.Keys(result))
	assert.ElementsMatch(t, wanted, got, "record keys with geoip")

	log.Reset()
}

func TestRecord(t *testing.T) {
	var ok bool
	geoDatabasePath, ok = os.LookupEnv("CONNTRACKD_GEOIP_DATABASE")
	if !ok || geoDatabasePath == "" {
		geoDatabasePath = "/tmp/GeoLite2-City.mmdb"
	}
	if _, err := os.Stat(geoDatabasePath); os.IsNotExist(err) {
		t.Skip("GeoIP database not found, skipping GeoIP tests")
	}

	t.Run("record.Record logs basic data without location data", recordLogsBasicDataIfNoLocationIsGiven)
	t.Run("record.Record logs all data with location data", recordLogsAllDataIfLocationIsGiven)
}
