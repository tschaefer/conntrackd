/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
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
	// Create event and then set Type using constants to satisfy type system
	event := conntrack.Event{Flow: &flow}
	switch eventTypeVal {
	case 1: // NEW
		event.Type = conntrack.EventNew
	case 2: // UPDATE
		event.Type = conntrack.EventUpdate
	case 3: // DESTROY
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
	// Create event and then set Type using constants to satisfy type system
	event := conntrack.Event{Flow: &flow}
	switch eventTypeVal {
	case 1: // NEW
		event.Type = conntrack.EventNew
	case 2: // UPDATE
		event.Type = conntrack.EventUpdate
	case 3: // DESTROY
		event.Type = conntrack.EventDestroy
	}
	return event
}

func TestEval_TypePredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		event    conntrack.Event
		expected bool
	}{
		{"match NEW", "log type NEW", createEvent(1, syscall.IPPROTO_TCP), true},
		{"match UPDATE", "log type UPDATE", createEvent(2, syscall.IPPROTO_TCP), true},
		{"match DESTROY", "log type DESTROY", createEvent(3, syscall.IPPROTO_TCP), true},
		{"no match NEW", "log type UPDATE", createEvent(1, syscall.IPPROTO_TCP), false},
		{"match multiple", "log type NEW,UPDATE", createEvent(1, syscall.IPPROTO_TCP), true},
		{"match multiple 2", "log type NEW,UPDATE", createEvent(2, syscall.IPPROTO_TCP), true},
		{"no match multiple", "log type NEW,UPDATE", createEvent(3, syscall.IPPROTO_TCP), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.rule)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			pred, err := Compile(rule.Expr)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, pred(tt.event))
		})
	}
}

func TestEval_ProtocolPredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		proto    uint8
		expected bool
	}{
		{"match TCP", "log protocol TCP", syscall.IPPROTO_TCP, true},
		{"match UDP", "log protocol UDP", syscall.IPPROTO_UDP, true},
		{"no match TCP", "log protocol UDP", syscall.IPPROTO_TCP, false},
		{"match multiple", "log protocol TCP,UDP", syscall.IPPROTO_TCP, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.rule)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			pred, err := Compile(rule.Expr)
			require.NoError(t, err)

			event := createEvent(1, tt.proto)
			assert.Equal(t, tt.expected, pred(event))
		})
	}
}

func TestEval_NetworkPredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		srcIP    string
		dstIP    string
		expected bool
	}{
		{"src private", "log source network PRIVATE", "10.0.0.1", "8.8.8.8", true},
		{"src public", "log source network PUBLIC", "8.8.8.8", "10.0.0.1", true},
		{"dst public", "log destination network PUBLIC", "10.0.0.1", "8.8.8.8", true},
		{"dst private", "log destination network PRIVATE", "8.8.8.8", "10.0.0.1", true},
		{"src local loopback", "log source network LOCAL", "127.0.0.1", "8.8.8.8", true},
		{"dst multicast", "log destination network MULTICAST", "10.0.0.1", "224.0.0.1", true},
		{"no match", "log source network PUBLIC", "10.0.0.1", "8.8.8.8", false},
		{"ipv6 private", "log source network PRIVATE", "fc00::1", "2001:db8::1", true},
		{"ipv6 public", "log destination network PUBLIC", "10.0.0.1", "2001:4860:4860::8888", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.rule)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			pred, err := Compile(rule.Expr)
			require.NoError(t, err)

			event := createEventWithAddrs(1, syscall.IPPROTO_TCP, tt.srcIP, tt.dstIP, 1234, 80)
			assert.Equal(t, tt.expected, pred(event))
		})
	}
}

func TestEval_AddressPredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		srcIP    string
		dstIP    string
		srcPort  uint16
		dstPort  uint16
		expected bool
	}{
		{"dst exact match", "log destination address 8.8.8.8", "10.0.0.1", "8.8.8.8", 1234, 80, true},
		{"dst no match", "log destination address 8.8.8.8", "10.0.0.1", "8.8.4.4", 1234, 80, false},
		{"src exact match", "log source address 10.0.0.1", "10.0.0.1", "8.8.8.8", 1234, 80, true},
		{"cidr match", "log destination address 8.8.8.0/24", "10.0.0.1", "8.8.8.100", 1234, 80, true},
		{"cidr no match", "log destination address 8.8.8.0/24", "10.0.0.1", "8.8.9.1", 1234, 80, false},
		{"with port match", "log destination address 10.19.80.100 on port 53", "192.168.1.1", "10.19.80.100", 1234, 53, true},
		{"with port no match addr", "log destination address 10.19.80.100 on port 53", "192.168.1.1", "10.19.80.101", 1234, 53, false},
		{"with port no match port", "log destination address 10.19.80.100 on port 53", "192.168.1.1", "10.19.80.100", 1234, 80, false},
		{"ipv6 match", "log destination address 2001:4860:4860::8888", "10.0.0.1", "2001:4860:4860::8888", 1234, 80, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.rule)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			pred, err := Compile(rule.Expr)
			require.NoError(t, err)

			event := createEventWithAddrs(1, syscall.IPPROTO_TCP, tt.srcIP, tt.dstIP, tt.srcPort, tt.dstPort)
			assert.Equal(t, tt.expected, pred(event))
		})
	}
}

func TestEval_PortPredicate(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		srcPort  uint16
		dstPort  uint16
		expected bool
	}{
		{"dst port match", "log destination port 80", 1234, 80, true},
		{"dst port no match", "log destination port 80", 1234, 443, false},
		{"src port match", "log source port 1234", 1234, 80, true},
		{"on port dst match", "log on port 80", 1234, 80, true},
		{"on port src match", "log on port 1234", 1234, 80, true},
		{"on port no match", "log on port 53", 1234, 80, false},
		{"port range", "log destination port 80,443,8080", 1234, 443, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.rule)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			pred, err := Compile(rule.Expr)
			require.NoError(t, err)

			event := createEventWithAddrs(1, syscall.IPPROTO_TCP, "10.0.0.1", "8.8.8.8", tt.srcPort, tt.dstPort)
			assert.Equal(t, tt.expected, pred(event))
		})
	}
}

func TestEval_AnyPredicate(t *testing.T) {
	tests := []struct {
		name  string
		rule  string
		event conntrack.Event
	}{
		{
			"any matches NEW TCP",
			"log any",
			createEvent(1, syscall.IPPROTO_TCP),
		},
		{
			"any matches UPDATE UDP",
			"drop any",
			createEvent(2, syscall.IPPROTO_UDP),
		},
		{
			"any matches DESTROY",
			"log any",
			createEvent(3, syscall.IPPROTO_TCP),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.rule)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			pred, err := Compile(rule.Expr)
			require.NoError(t, err)
			result := pred(tt.event)
			assert.True(t, result, "any predicate should always match")
		})
	}
}

func TestEval_BinaryExpressions(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		event    conntrack.Event
		expected bool
	}{
		{
			"and both match",
			"log type NEW and protocol TCP",
			createEvent(1, syscall.IPPROTO_TCP),
			true,
		},
		{
			"and first no match",
			"log type UPDATE and protocol TCP",
			createEvent(1, syscall.IPPROTO_TCP),
			false,
		},
		{
			"and second no match",
			"log type NEW and protocol UDP",
			createEvent(1, syscall.IPPROTO_TCP),
			false,
		},
		{
			"or first match",
			"log type NEW or type UPDATE",
			createEvent(1, syscall.IPPROTO_TCP),
			true,
		},
		{
			"or second match",
			"log type UPDATE or type NEW",
			createEvent(1, syscall.IPPROTO_TCP),
			true,
		},
		{
			"or both no match",
			"log type UPDATE or type DESTROY",
			createEvent(1, syscall.IPPROTO_TCP),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.rule)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			pred, err := Compile(rule.Expr)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, pred(tt.event))
		})
	}
}

func TestEval_UnaryExpression(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		event    conntrack.Event
		expected bool
	}{
		{
			"not match becomes false",
			"log not type NEW",
			createEvent(1, syscall.IPPROTO_TCP),
			false,
		},
		{
			"not no-match becomes true",
			"log not type UPDATE",
			createEvent(1, syscall.IPPROTO_TCP),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.rule)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			pred, err := Compile(rule.Expr)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, pred(tt.event))
		})
	}
}

func TestFilter_Evaluate(t *testing.T) {
	rules := []string{
		"drop destination address 8.8.8.8",
		"log protocol TCP and destination network PUBLIC",
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

func TestFilter_EmptyAllowsByDefault(t *testing.T) {
	filter, err := NewFilter([]string{})
	require.NoError(t, err)

	event := createEventWithAddrs(1, syscall.IPPROTO_TCP, "10.0.0.1", "8.8.8.8", 1234, 80)
	matched, allow, matchedIndex := filter.Evaluate(event)
	assert.False(t, matched)
	assert.True(t, allow)
	assert.Equal(t, -1, matchedIndex)
}

func TestFilter_FirstMatchWins(t *testing.T) {
	rules := []string{
		"log protocol TCP",
		"drop destination address 8.8.8.8",
	}

	filter, err := NewFilter(rules)
	require.NoError(t, err)

	// TCP to 8.8.8.8 should be allowed by first rule (first match wins)
	event := createEventWithAddrs(1, syscall.IPPROTO_TCP, "10.0.0.1", "8.8.8.8", 1234, 80)
	matched, allow, matchedIndex := filter.Evaluate(event)
	assert.True(t, matched)
	assert.True(t, allow)
	assert.Equal(t, 0, matchedIndex)
}
