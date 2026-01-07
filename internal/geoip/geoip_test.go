/*
Copyright (c) Tobias Schäfer. All rights reserved.
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

func newReturnsErrorIfDatabaseIsInvalid(t *testing.T) {
	_, err := NewGeoIP("../../README.md")
	assert.EqualError(t, err, "error opening database: invalid MaxMind DB file")
}

func newReturnsInstanceIfDatabaseIsValid(t *testing.T) {
	geo, err := NewGeoIP(geoDatabasePath)
	assert.NoError(t, err)
	assert.NotNil(t, geo)
	defer geo.Close()
	assert.IsType(t, &GeoIP{}, geo)
	assert.Equal(t, geo.Database, geoDatabasePath)
	assert.IsType(t, geo.Reader, &geoip2.Reader{})
}

func locationReturnsNilIfAddressIsUnresolved(t *testing.T) {
	geo, err := NewGeoIP(geoDatabasePath)
	assert.NoError(t, err)
	assert.NotNil(t, geo)
	defer geo.Close()

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

func locationReturnsLocationIfAddressIsResolved(t *testing.T) {
	geo, err := NewGeoIP(geoDatabasePath)
	assert.NoError(t, err)
	assert.NotNil(t, geo)
	defer geo.Close()

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

	t.Run("geoip.NewGeoIP returns error if database is invalid", newReturnsErrorIfDatabaseIsInvalid)
	t.Run("geoip.NewGeoIP returns instance if database is valid", newReturnsInstanceIfDatabaseIsValid)
	t.Run("geoip.Location returns nil if IP is unresolved", locationReturnsNilIfAddressIsUnresolved)
	t.Run("geoip.Location returns location if IP is resolved", locationReturnsLocationIfAddressIsResolved)
}
