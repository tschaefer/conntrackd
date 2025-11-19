/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package geoip

import (
	"log/slog"
	"net/netip"

	"github.com/oschwald/geoip2-golang/v2"
)

type Reader struct {
	reader *geoip2.Reader
}

type GeoIP struct {
	Database string
}

type Location struct {
	Country string
	City    string
	Lat     float64
	Lon     float64
}

func Open(path string) (*Reader, error) {
	slog.Debug("Opening GeoIP2 database", "path", path)

	reader, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}

	return &Reader{reader: reader}, nil
}

func (r *Reader) Close() error {
	return r.reader.Close()
}

func (r *Reader) Location(ip netip.Addr) *Location {
	record, err := r.reader.City(ip)
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
