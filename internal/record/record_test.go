/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package record

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/netip"
	"os"
	"slices"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ti-mo/conntrack"
	"github.com/tschaefer/conntrackd/internal/geoip"
)

const geoDatabasePath = "/tmp/GeoLite2-City.mmdb"
const geoDatabaseUrl = "https://git.io/GeoLite2-City.mmdb"

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

func setupGeoDatabase() {
	if _, err := os.Stat(geoDatabasePath); os.IsNotExist(err) {
		resp, err := http.Get(geoDatabaseUrl)
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		out, err := os.Create(geoDatabasePath)
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = out.Close()
		}()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			panic(err)
		}
	}
}

func recordLogsBasicData(t *testing.T) {
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
		"type", "flow", "prot", "src", "dst", "sport", "dport"}
	got := slices.Sorted(maps.Keys(result))
	assert.ElementsMatch(t, wanted, got, "record basic keys")

	log.Reset()
}

func recordLogsWithGeoIPData(t *testing.T) {
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
		"type", "flow", "prot", "src", "dst", "sport", "dport",
		"city", "country", "lat", "lon"}
	got := slices.Sorted(maps.Keys(result))
	assert.ElementsMatch(t, wanted, got, "record keys with geoip")

	log.Reset()
}

func TestRecord(t *testing.T) {
	setupGeoDatabase()

	t.Run("Record logs basic data", recordLogsBasicData)
	t.Run("Record logs with GeoIP data", recordLogsWithGeoIPData)

	skip, ok := os.LookupEnv("KEEP_GEOIP_DB")
	if ok && skip == "1" || skip == "true" {
		return
	}
	_ = os.Remove(geoDatabasePath)
}
