/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

import (
	"io"
	"net/http"
	"net/netip"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ti-mo/conntrack"
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

func Test_Filter(t *testing.T) {
	setup()

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

	f := &Filter{}
	matched := f.Apply(event)
	assert.True(t, matched, "no filters")

	f = &Filter{
		Protocols: []string{"TCP"},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "protocol filter TCP")

	f = &Filter{
		EventTypes: []string{"NEW"},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "event type filter NEW")

	f = &Filter{
		EventTypes: []string{"UPDATE"},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "event type filter UPDATE")

	f = &Filter{
		EventTypes: []string{"DESTROY"},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "event type filter DESTROY")

	f = &Filter{
		EventTypes: []string{"NEW", "DESTROY"},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "event type filter NEW, DESTROY")

	f = &Filter{
		Protocols: []string{"UDP"},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "protocol filter UDP")

	f = &Filter{
		Protocols: []string{"UDP", "TCP"},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "protocol filter UDP, TCP")

	f = &Filter{
		Protocols: []string{"ICMP"},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "bad protocol filter")

	f = &Filter{
		Destinations: []string{"PUBLIC"},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "destination filter PUBLIC")

	f = &Filter{
		Destinations: []string{"PRIVATE"},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "destination filter PRIVATE")

	f = &Filter{
		Destinations: []string{"LOCAL"},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "destination filter LOCAL")

	f = &Filter{
		Destinations: []string{"MULTICAST"},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "destination filter MULTICAST")
}
