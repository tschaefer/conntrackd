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

	"github.com/stretchr/testify/assert"
)

const geoDatabasePath = "/tmp/GeoLite2-City.mmdb"
const geoDatabaseUrl = "https://github.com/P3TERX/GeoLite.mmdb/releases/latest/download/GeoLite2-City.mmdb"

func setup() {
	setupGeoDatabase()
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

func Test_GeoIP(t *testing.T) {
	setup()

	geo, err := Open("invalid-path.mmdb")
	assert.Error(t, err, "open invalid path")
	assert.Nil(t, geo, "no geoip instance")

	geo, err = Open(geoDatabasePath)
	assert.NoError(t, err, "open valid path")
	assert.NotNil(t, geo, "geoip instance")
	defer func() {
		_ = geo.Close()
	}()

	testees := map[string]string{
		"::1":           "local address",
		"10.19.80.12":   "private address",
		"224.0.1.1":     "multicast address",
		"172.66.43.195": "unresolved address",
	}

	for ipStr, desc := range testees {
		ip, _ := netip.ParseAddr(ipStr)
		location := geo.Location(ip)
		assert.Nil(t, location, desc, "no location")
	}

	ip, _ := netip.ParseAddr("63.176.75.230")
	location := geo.Location(ip)
	assert.NotNil(t, location, "resolved address")
	assert.IsType(t, &Location{}, location, "location type")
}
