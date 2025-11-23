/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package geoip

import (
	"io"
	"net/http"
	"net/netip"
	"os"
	"testing"

	"github.com/oschwald/geoip2-golang/v2"
	"github.com/stretchr/testify/assert"
)

const geoDatabasePath = "/tmp/GeoLite2-City.mmdb"
const geoDatabaseUrl = "https://git.io/GeoLite2-City.mmdb"

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

func newReturnsError_InvalidDatabase(t *testing.T) {
	_, err := NewGeoIP("../../README.md")
	assert.EqualError(t, err, "error opening database: invalid MaxMind DB file")
}

func newReturnsInstance_ValidDatabase(t *testing.T) {
	geoIP, err := NewGeoIP(geoDatabasePath)
	assert.NoError(t, err)
	assert.NotNil(t, geoIP)
	assert.IsType(t, &GeoIP{}, geoIP)
	assert.Equal(t, geoIP.Database, geoDatabasePath)
	assert.IsType(t, geoIP.Reader, &geoip2.Reader{})
}

func locationReturnsNil_UnresolvedIP(t *testing.T) {
	geo, err := NewGeoIP(geoDatabasePath)
	assert.NoError(t, err)
	assert.NotNil(t, geo)

	for ipStr, desc := range map[string]string{
		"::1":           "local address",
		"10.19.80.12":   "private address",
		"224.0.1.1":     "multicast address",
		"172.66.43.195": "unresolved address",
	} {
		ip, _ := netip.ParseAddr(ipStr)
		location := geo.Location(ip)
		assert.Nil(t, location, desc)
	}
}

func locationReturnsLocation_ResolvedIP(t *testing.T) {
	geo, err := NewGeoIP(geoDatabasePath)
	assert.NoError(t, err)
	assert.NotNil(t, geo)

	ip, _ := netip.ParseAddr("63.176.75.230")
	location := geo.Location(ip)
	assert.NotNil(t, location, "resolved address")
	assert.IsType(t, &Location{}, location, "location type")
}

func TestGeoIP(t *testing.T) {
	setupGeoDatabase()

	t.Run("New returns error for invalid database", newReturnsError_InvalidDatabase)
	t.Run("New returns instance for valid database", newReturnsInstance_ValidDatabase)
	t.Run("Location returns nil for unresolved IPs", locationReturnsNil_UnresolvedIP)
	t.Run("Location returns location for resolved IPs", locationReturnsLocation_ResolvedIP)

	skip, ok := os.LookupEnv("KEEP_GEOIP_DB")
	if ok && skip == "1" || skip == "true" {
		return
	}
	_ = os.Remove(geoDatabasePath)
}
