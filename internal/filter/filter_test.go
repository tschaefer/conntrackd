/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

import (
	"net/netip"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ti-mo/conntrack"
)

func createEvent(eventTypeVal, proto uint8) conntrack.Event {
	flow := conntrack.NewFlow(
		proto,
		conntrack.StatusAssured,
		netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("8.8.8.8"),
		1234, 80,
		60, 0,
	)

	event := conntrack.Event{Flow: &flow}
	switch eventTypeVal {
	case 1:
		event.Type = conntrack.EventNew
	case 2:
		event.Type = conntrack.EventUpdate
	case 3:
		event.Type = conntrack.EventDestroy
	}
	return event
}

func createEventWithAddrs(eventTypeVal, proto uint8, srcIP, dstIP string, srcPort, dstPort uint16) conntrack.Event {
	flow := conntrack.NewFlow(
		proto,
		conntrack.StatusAssured,
		netip.MustParseAddr(srcIP), netip.MustParseAddr(dstIP),
		srcPort, dstPort,
		60, 0,
	)

	event := conntrack.Event{Flow: &flow}
	switch eventTypeVal {
	case 1:
		event.Type = conntrack.EventNew
	case 2:
		event.Type = conntrack.EventUpdate
	case 3:
		event.Type = conntrack.EventDestroy
	}
	return event
}

func TestCEL_TypePredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		event    conntrack.Event
		expected bool
	}{
		{"match NEW", `log event.type == "NEW"`, createEvent(1, syscall.IPPROTO_TCP), true},
		{"match UPDATE", `log event.type == "UPDATE"`, createEvent(2, syscall.IPPROTO_TCP), true},
		{"match DESTROY", `log event.type == "DESTROY"`, createEvent(3, syscall.IPPROTO_TCP), true},
		{"no match NEW", `log event.type == "UPDATE"`, createEvent(1, syscall.IPPROTO_TCP), false},
		{"match multiple", `log event.type == "NEW" || event.type == "UPDATE"`, createEvent(1, syscall.IPPROTO_TCP), true},
		{"match multiple 2", `log event.type == "NEW" || event.type == "UPDATE"`, createEvent(2, syscall.IPPROTO_TCP), true},
		{"no match multiple", `log event.type == "NEW" || event.type == "UPDATE"`, createEvent(3, syscall.IPPROTO_TCP), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilter([]string{tt.rule})
			require.NoError(t, err)

			matched, _, _ := filter.Evaluate(tt.event)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

func TestCEL_ProtocolPredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		proto    uint8
		expected bool
	}{
		{"match TCP", `log protocol == "TCP"`, syscall.IPPROTO_TCP, true},
		{"match UDP", `log protocol == "UDP"`, syscall.IPPROTO_UDP, true},
		{"no match TCP", `log protocol == "UDP"`, syscall.IPPROTO_TCP, false},
		{"match multiple", `log protocol == "TCP" || protocol == "UDP"`, syscall.IPPROTO_TCP, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilter([]string{tt.rule})
			require.NoError(t, err)

			event := createEvent(1, tt.proto)
			matched, _, _ := filter.Evaluate(event)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

func TestCEL_NetworkPredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		srcIP    string
		dstIP    string
		expected bool
	}{
		{"src private", `log is_network(source.address, "PRIVATE")`, "10.0.0.1", "8.8.8.8", true},
		{"src public", `log is_network(source.address, "PUBLIC")`, "8.8.8.8", "10.0.0.1", true},
		{"dst public", `log is_network(destination.address, "PUBLIC")`, "10.0.0.1", "8.8.8.8", true},
		{"dst private", `log is_network(destination.address, "PRIVATE")`, "8.8.8.8", "10.0.0.1", true},
		{"src local loopback", `log is_network(source.address, "LOCAL")`, "127.0.0.1", "8.8.8.8", true},
		{"dst multicast", `log is_network(destination.address, "MULTICAST")`, "10.0.0.1", "224.0.0.1", true},
		{"no match", `log is_network(source.address, "PUBLIC")`, "10.0.0.1", "8.8.8.8", false},
		{"ipv6 private", `log is_network(source.address, "PRIVATE")`, "fc00::1", "2001:db8::1", true},
		{"ipv6 public", `log is_network(destination.address, "PUBLIC")`, "10.0.0.1", "2001:4860:4860::8888", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilter([]string{tt.rule})
			require.NoError(t, err)

			event := createEventWithAddrs(1, syscall.IPPROTO_TCP, tt.srcIP, tt.dstIP, 1234, 80)
			matched, _, _ := filter.Evaluate(event)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

func TestCEL_AddressPredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		srcIP    string
		dstIP    string
		srcPort  uint16
		dstPort  uint16
		expected bool
	}{
		{"dst exact match", `log destination.address == "8.8.8.8"`, "10.0.0.1", "8.8.8.8", 1234, 80, true},
		{"dst no match", `log destination.address == "8.8.8.8"`, "10.0.0.1", "8.8.4.4", 1234, 80, false},
		{"src exact match", `log source.address == "10.0.0.1"`, "10.0.0.1", "8.8.8.8", 1234, 80, true},
		{"cidr match", `log in_cidr(destination.address, "8.8.8.0/24")`, "10.0.0.1", "8.8.8.100", 1234, 80, true},
		{"cidr no match", `log in_cidr(destination.address, "8.8.8.0/24")`, "10.0.0.1", "8.8.9.1", 1234, 80, false},
		{"with port match", `log destination.address == "10.19.80.100" && destination.port == 53`, "192.168.1.1", "10.19.80.100", 1234, 53, true},
		{"with port no match addr", `log destination.address == "10.19.80.100" && destination.port == 53`, "192.168.1.1", "10.19.80.101", 1234, 53, false},
		{"with port no match port", `log destination.address == "10.19.80.100" && destination.port == 53`, "192.168.1.1", "10.19.80.100", 1234, 80, false},
		{"ipv6 match", `log destination.address == "2001:4860:4860::8888"`, "10.0.0.1", "2001:4860:4860::8888", 1234, 80, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilter([]string{tt.rule})
			require.NoError(t, err)

			event := createEventWithAddrs(1, syscall.IPPROTO_TCP, tt.srcIP, tt.dstIP, tt.srcPort, tt.dstPort)
			matched, _, _ := filter.Evaluate(event)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

func TestCEL_PortPredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		srcPort  uint16
		dstPort  uint16
		expected bool
	}{
		{"dst port match", `log destination.port == 80`, 1234, 80, true},
		{"dst port no match", `log destination.port == 80`, 1234, 443, false},
		{"src port match", `log source.port == 1234`, 1234, 80, true},
		{"either port match dst", `log source.port == 80 || destination.port == 80`, 1234, 80, true},
		{"either port match src", `log source.port == 1234 || destination.port == 1234`, 1234, 80, true},
		{"either port no match", `log source.port == 53 || destination.port == 53`, 1234, 80, false},
		{"port in list", `log destination.port == 80 || destination.port == 443 || destination.port == 8080`, 1234, 443, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilter([]string{tt.rule})
			require.NoError(t, err)

			event := createEventWithAddrs(1, syscall.IPPROTO_TCP, "10.0.0.1", "8.8.8.8", tt.srcPort, tt.dstPort)
			matched, _, _ := filter.Evaluate(event)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

func TestCEL_AnyPredicate(t *testing.T) {
	tests := []struct {
		name  string
		rule  string
		event conntrack.Event
	}{
		{
			"any matches NEW TCP",
			"log true",
			createEvent(1, syscall.IPPROTO_TCP),
		},
		{
			"any matches UPDATE UDP",
			"drop true",
			createEvent(2, syscall.IPPROTO_UDP),
		},
		{
			"any matches DESTROY",
			"log true",
			createEvent(3, syscall.IPPROTO_TCP),
		},
		{
			"any matches with alias",
			`log any`, createEvent(4, syscall.IPPROTO_TCP),
		},
		{
			"any matches with alias",
			`drop any`, createEvent(5, syscall.IPPROTO_UDP),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilter([]string{tt.rule})
			require.NoError(t, err)

			matched, _, _ := filter.Evaluate(tt.event)
			assert.True(t, matched, "true predicate should always match")
		})
	}
}

func TestCEL_BinaryExpressions(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		event    conntrack.Event
		expected bool
	}{
		{
			"and both match",
			`log event.type == "NEW" && protocol == "TCP"`,
			createEvent(1, syscall.IPPROTO_TCP),
			true,
		},
		{
			"and first no match",
			`log event.type == "UPDATE" && protocol == "TCP"`,
			createEvent(1, syscall.IPPROTO_TCP),
			false,
		},
		{
			"and second no match",
			`log event.type == "NEW" && protocol == "UDP"`,
			createEvent(1, syscall.IPPROTO_TCP),
			false,
		},
		{
			"or first match",
			`log event.type == "NEW" || event.type == "UPDATE"`,
			createEvent(1, syscall.IPPROTO_TCP),
			true,
		},
		{
			"or second match",
			`log event.type == "UPDATE" || event.type == "NEW"`,
			createEvent(1, syscall.IPPROTO_TCP),
			true,
		},
		{
			"or both no match",
			`log event.type == "UPDATE" || event.type == "DESTROY"`,
			createEvent(1, syscall.IPPROTO_TCP),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilter([]string{tt.rule})
			require.NoError(t, err)

			matched, _, _ := filter.Evaluate(tt.event)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

func TestCEL_UnaryExpression(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		event    conntrack.Event
		expected bool
	}{
		{
			"not match becomes false",
			`log !(event.type == "NEW")`,
			createEvent(1, syscall.IPPROTO_TCP),
			false,
		},
		{
			"not no-match becomes true",
			`log !(event.type == "UPDATE")`,
			createEvent(1, syscall.IPPROTO_TCP),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilter([]string{tt.rule})
			require.NoError(t, err)

			matched, _, _ := filter.Evaluate(tt.event)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

func TestCELFilter_Evaluate(t *testing.T) {
	rules := []string{
		`drop destination.address == "8.8.8.8"`,
		`log protocol == "TCP" && is_network(destination.address, "PUBLIC")`,
	}

	filter, err := NewFilter(rules)
	require.NoError(t, err)

	tests := []struct {
		name         string
		event        conntrack.Event
		matched      bool
		allow        bool
		matchedIndex int
	}{
		{
			"first rule denies",
			createEventWithAddrs(1, syscall.IPPROTO_TCP, "10.0.0.1", "8.8.8.8", 1234, 80),
			true,
			false,
			0,
		},
		{
			"second rule allows",
			createEventWithAddrs(1, syscall.IPPROTO_TCP, "10.0.0.1", "1.1.1.1", 1234, 80),
			true,
			true,
			1,
		},
		{
			"no match allows by default",
			createEventWithAddrs(1, syscall.IPPROTO_UDP, "10.0.0.1", "192.168.1.1", 1234, 80),
			false,
			true,
			-1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, allow, matchedIndex := filter.Evaluate(tt.event)
			assert.Equal(t, tt.matched, matched)
			assert.Equal(t, tt.allow, allow)
			assert.Equal(t, tt.matchedIndex, matchedIndex)
		})
	}
}

func TestCELFilter_EmptyAllowsByDefault(t *testing.T) {
	filter, err := NewFilter([]string{})
	require.NoError(t, err)

	event := createEventWithAddrs(1, syscall.IPPROTO_TCP, "10.0.0.1", "8.8.8.8", 1234, 80)
	matched, allow, matchedIndex := filter.Evaluate(event)
	assert.False(t, matched)
	assert.True(t, allow)
	assert.Equal(t, -1, matchedIndex)
}

func TestCELFilter_FirstMatchWins(t *testing.T) {
	rules := []string{
		`log protocol == "TCP"`,
		`drop destination.address == "8.8.8.8"`,
	}

	filter, err := NewFilter(rules)
	require.NoError(t, err)

	event := createEventWithAddrs(1, syscall.IPPROTO_TCP, "10.0.0.1", "8.8.8.8", 1234, 80)
	matched, allow, matchedIndex := filter.Evaluate(event)
	assert.True(t, matched)
	assert.True(t, allow)
	assert.Equal(t, 0, matchedIndex)
}

func TestCEL_ComplexExamples(t *testing.T) {
	tests := []struct {
		name  string
		rule  string
		event conntrack.Event
		match bool
	}{
		{
			"complex with negation",
			`log !(event.type == "DESTROY" && is_network(destination.address, "PRIVATE"))`,
			createEventWithAddrs(3, syscall.IPPROTO_TCP, "10.0.0.1", "192.168.1.1", 1234, 80),
			false,
		},
		{
			"multiple conditions",
			`drop is_network(source.address, "LOCAL") && (destination.port == 22 || destination.port == 23 || destination.port == 3389)`,
			createEventWithAddrs(1, syscall.IPPROTO_TCP, "127.0.0.1", "8.8.8.8", 1234, 22),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFilter([]string{tt.rule})
			require.NoError(t, err)
			matched, _, _ := filter.Evaluate(tt.event)
			assert.Equal(t, tt.match, matched)
		})
	}
}
