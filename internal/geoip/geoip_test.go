/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package geoip

import (
	"net/netip"
	"os"
	"testing"

	"github.com/oschwald/geoip2-golang/v2"
	"github.com/stretchr/testify/assert"
)

var geoDatabasePath string

func newReturnsError_InvalidDatabase(t *testing.T) {
	_, err := NewGeoIP("../../README.md")
	assert.EqualError(t, err, "error opening database: invalid MaxMind DB file")
}

func newReturnsInstance_ValidDatabase(t *testing.T) {
	t.Logf("Using GeoIP database at %s", geoDatabasePath)
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
	var ok bool
	geoDatabasePath, ok = os.LookupEnv("CONNTRACKD_GEOIP_DATABASE")
	if !ok || geoDatabasePath == "" {
		geoDatabasePath = "/tmp/GeoLite2-City.mmdb"
	}
	if _, err := os.Stat(geoDatabasePath); os.IsNotExist(err) {
		t.Skip("GeoIP database not found, skipping GeoIP tests")
	}

	t.Run("New returns error for invalid database", newReturnsError_InvalidDatabase)
	t.Run("New returns instance for valid database", newReturnsInstance_ValidDatabase)
	t.Run("Location returns nil for unresolved IPs", locationReturnsNil_UnresolvedIP)
	t.Run("Location returns location for resolved IPs", locationReturnsLocation_ResolvedIP)
}
