/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package geoip

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/oschwald/geoip2-golang/v2"
)

// GeoIP is a wrapper around the GeoIP2 City database reader.
type GeoIP struct {
	Reader   *geoip2.Reader
	Database string
}

// Location represents geographical location information.
type Location struct {
	Country string
	City    string
	Lat     float64
	Lon     float64
}

// NewGeoIP creates a new GeoIP instance by loading the specified GeoIP2 City
// database file.
func NewGeoIP(database string) (*GeoIP, error) {
	reader, err := geoip2.Open(database)
	if err != nil {
		return nil, err
	}

	metadata := reader.Metadata()
	if !strings.HasSuffix(metadata.DatabaseType, "City") {
		_ = reader.Close()
		return nil, fmt.Errorf("invalid GeoIP2 database type: %s, expected City", metadata.DatabaseType)
	}

	return &GeoIP{
		Reader:   reader,
		Database: database,
	}, nil
}

// Close closes the GeoIP database reader.
func (g *GeoIP) Close() error {
	return g.Reader.Close()
}

// Location retrieves the geographical location information for the given IP
// address. If no location data is found, it returns nil.
func (g *GeoIP) Location(ip netip.Addr) *Location {
	record, err := g.Reader.City(ip)
	if err != nil {
		return nil
	}
	if !record.HasData() {
		return nil
	}

	var country, city string
	if record.Country.HasData() {
		country = record.Country.Names.English
	}
	if record.City.HasData() {
		city = record.City.Names.English
	}

	var lat, lon float64
	if record.Location.HasCoordinates() {
		lat = *record.Location.Latitude
		lon = *record.Location.Longitude
	}

	if country == "" && city == "" &&
		lat == 0 && lon == 0 {
		return nil
	}

	return &Location{
		Country: country,
		City:    city,
		Lat:     lat,
		Lon:     lon,
	}
}
