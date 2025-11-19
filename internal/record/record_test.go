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
const geoDatabaseUrl = "https://github.com/P3TERX/GeoLite.mmdb/releases/latest/download/GeoLite2-City.mmdb"

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

func Test_Record(t *testing.T) {
	logger := setupLogger()
	setupGeoDatabase()

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

	var geo *geoip.Reader
	Record(event, geo, logger)
	var result map[string]any
	err := json.Unmarshal(log.Bytes(), &result)
	assert.NoError(t, err)

	wanted := []string{"level", "time",
		"type", "flow", "prot", "src", "dst", "sport", "dport"}
	got := slices.Sorted(maps.Keys(result))
	assert.ElementsMatch(t, wanted, got, "record keys without geoip")

	geo, err = geoip.Open(geoDatabasePath)
	assert.NoError(t, err)
	defer func() {
		_ = geo.Close()
	}()

	log.Reset()
	Record(event, geo, logger)
	err = json.Unmarshal(log.Bytes(), &result)
	assert.NoError(t, err)

	wanted = append(wanted, []string{"city", "country", "lat", "lon"}...)
	got = slices.Sorted(maps.Keys(result))
	assert.ElementsMatch(t, wanted, got, "record keys with geoip")

}
