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
	assert.False(t, matched, "no filters")

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
		Networks: FilterNetworks{
			Destinations: []string{"PUBLIC"},
		},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "destination network filter PUBLIC")

	f = &Filter{
		Networks: FilterNetworks{
			Destinations: []string{"PRIVATE"},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "destination network filter PRIVATE")

	f = &Filter{
		Networks: FilterNetworks{
			Destinations: []string{"LOCAL"},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "destination network filter LOCAL")

	f = &Filter{
		Networks: FilterNetworks{
			Destinations: []string{"MULTICAST"},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "source network filter MULTICAST")

	f = &Filter{
		Networks: FilterNetworks{
			Sources: []string{"PUBLIC"},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "source network filter PUBLIC")

	f = &Filter{
		Networks: FilterNetworks{
			Sources: []string{"PRIVATE"},
		},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "source network filter PRIVATE")

	f = &Filter{
		Networks: FilterNetworks{
			Sources: []string{"LOCAL"},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "source network filter LOCAL")

	f = &Filter{
		Networks: FilterNetworks{
			Sources: []string{"MULTICAST"},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "source network filter MULTICAST")

	f = &Filter{
		Addresses: FilterAddresses{
			Destinations: []string{"78.47.60.169"},
		},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "destination address filter match")

	f = &Filter{
		Addresses: FilterAddresses{
			Destinations: []string{"78.47.60.170"},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "destination address filter no match")

	f = &Filter{
		Addresses: FilterAddresses{
			Sources: []string{"10.19.80.100"},
		},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "source address filter match")

	f = &Filter{
		Addresses: FilterAddresses{
			Sources: []string{"10.19.80.200"},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "source address filter no match")

	f = &Filter{
		Ports: FilterPorts{
			Destinations: []uint{443},
		},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "destination port filter match")

	f = &Filter{
		Ports: FilterPorts{
			Destinations: []uint{80},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "destination port filter no match")

	f = &Filter{
		Ports: FilterPorts{
			Sources: []uint{4711},
		},
	}
	matched = f.Apply(event)
	assert.True(t, matched, "source port filter match")

	f = &Filter{
		Ports: FilterPorts{
			Sources: []uint{1234},
		},
	}
	matched = f.Apply(event)
	assert.False(t, matched, "source port filter no match")
}
