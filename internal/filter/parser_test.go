/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_BasicActions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		action Action
		valid  bool
	}{
		{"log type", "log type NEW", ActionLog, true},
		{"drop type", "drop type NEW", ActionDrop, true},
		{"LOG uppercase", "LOG type NEW", ActionLog, true},
		{"DROP uppercase", "DROP type NEW", ActionDrop, true},
		{"missing action", "type NEW", ActionLog, false},
		{"invalid action", "permit type NEW", ActionLog, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			if !tt.valid {
				if err == nil {
					_, err = parser.ParseRule()
				}
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)
			assert.Equal(t, tt.action, rule.Action)
		})
	}
}

func TestParser_TypePredicate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"single type", "log type NEW", []string{"NEW"}},
		{"multiple types", "log type NEW,UPDATE", []string{"NEW", "UPDATE"}},
		{"all types", "log type NEW,UPDATE,DESTROY", []string{"NEW", "UPDATE", "DESTROY"}},
		{"lowercase", "log type new,update", []string{"NEW", "UPDATE"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			typePred, ok := rule.Expr.(TypePredicate)
			require.True(t, ok, "expected TypePredicate")
			assert.ElementsMatch(t, tt.expected, typePred.Types)
		})
	}
}

func TestParser_ProtocolPredicate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"single protocol", "log protocol TCP", []string{"TCP"}},
		{"multiple protocols", "log protocol TCP,UDP", []string{"TCP", "UDP"}},
		{"lowercase", "log protocol tcp", []string{"TCP"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			protoPred, ok := rule.Expr.(ProtocolPredicate)
			require.True(t, ok, "expected ProtocolPredicate")
			assert.ElementsMatch(t, tt.expected, protoPred.Protocols)
		})
	}
}

func TestParser_NetworkPredicate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		direction string
		networks  []string
	}{
		{"source private", "log source network PRIVATE", "source", []string{"PRIVATE"}},
		{"destination public", "log destination network PUBLIC", "destination", []string{"PUBLIC"}},
		{"dst abbreviation", "log dst network LOCAL", "dst", []string{"LOCAL"}},
		{"src abbreviation", "log src network MULTICAST", "src", []string{"MULTICAST"}},
		{"multiple networks", "log source network PRIVATE,LOCAL", "source", []string{"PRIVATE", "LOCAL"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			netPred, ok := rule.Expr.(NetworkPredicate)
			require.True(t, ok, "expected NetworkPredicate")
			assert.Equal(t, tt.direction, netPred.Direction)
			assert.ElementsMatch(t, tt.networks, netPred.Networks)
		})
	}
}

func TestParser_AddressPredicate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		direction string
		addresses []string
		ports     []uint16
	}{
		{"ipv4 address", "log destination address 8.8.8.8", "destination", []string{"8.8.8.8"}, nil},
		{"ipv6 address", "log source address 2001:db8::1", "source", []string{"2001:db8::1"}, nil},
		{"cidr", "log destination address 192.168.1.0/24", "destination", []string{"192.168.1.0/24"}, nil},
		{"address with port", "log destination address 10.19.80.100 on port 53", "destination", []string{"10.19.80.100"}, []uint16{53}},
		{"multiple addresses", "log source address 1.1.1.1,8.8.8.8", "source", []string{"1.1.1.1", "8.8.8.8"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			addrPred, ok := rule.Expr.(AddressPredicate)
			require.True(t, ok, "expected AddressPredicate")
			assert.Equal(t, tt.direction, addrPred.Direction)
			assert.ElementsMatch(t, tt.addresses, addrPred.Addresses)
			if tt.ports != nil {
				assert.ElementsMatch(t, tt.ports, addrPred.Ports)
			}
		})
	}
}

func TestParser_PortPredicate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		direction string
		ports     []uint16
	}{
		{"single port", "log destination port 80", "destination", []uint16{80}},
		{"multiple ports", "log source port 80,443", "source", []uint16{80, 443}},
		{"port range", "log destination port 8000-8005", "destination", []uint16{8000, 8001, 8002, 8003, 8004, 8005}},
		{"on port both", "log on port 53", "both", []uint16{53}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			portPred, ok := rule.Expr.(PortPredicate)
			require.True(t, ok, "expected PortPredicate")
			assert.Equal(t, tt.direction, portPred.Direction)
			assert.ElementsMatch(t, tt.ports, portPred.Ports)
		})
	}
}

func TestParser_AnyPredicate(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"log any", "log any"},
		{"drop any", "drop any"},
		{"ANY uppercase", "log ANY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			_, ok := rule.Expr.(AnyPredicate)
			assert.True(t, ok, "expected AnyPredicate")
		})
	}
}

func TestParser_BinaryExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		op    BinaryOp
	}{
		{"and operator", "log type NEW and protocol TCP", OpAnd},
		{"or operator", "log type NEW or type UPDATE", OpOr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			binExpr, ok := rule.Expr.(BinaryExpr)
			require.True(t, ok, "expected BinaryExpr")
			assert.Equal(t, tt.op, binExpr.Op)
		})
	}
}

func TestParser_UnaryExpression(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not keyword", "log not type NEW"},
		{"exclamation", "log ! protocol TCP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)

			unaryExpr, ok := rule.Expr.(UnaryExpr)
			require.True(t, ok, "expected UnaryExpr")
			assert.Equal(t, OpNot, unaryExpr.Op)
		})
	}
}

func TestParser_Parentheses(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple grouping", "log (type NEW)"},
		{"complex grouping", "log (type NEW and protocol TCP) or type UPDATE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			_, err = parser.ParseRule()
			require.NoError(t, err)
		})
	}
}

func TestParser_Precedence(t *testing.T) {
	// Test that AND binds tighter than OR
	input := "log type NEW or type UPDATE and protocol TCP"
	parser, err := NewParser(input)
	require.NoError(t, err)
	rule, err := parser.ParseRule()
	require.NoError(t, err)

	// Should parse as: (type NEW) OR (type UPDATE AND protocol TCP)
	orExpr, ok := rule.Expr.(BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, OpOr, orExpr.Op)

	// Right side should be AND
	andExpr, ok := orExpr.Right.(BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, OpAnd, andExpr.Op)
}

func TestParser_ComplexExamples(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"example 1",
			"drop destination address 8.8.8.8",
		},
		{
			"example 2",
			"log protocol TCP and destination network PUBLIC",
		},
		{
			"example 3",
			"drop destination address 10.19.80.100 on port 53",
		},
		{
			"example 4",
			"log protocol TCP,UDP",
		},
		{
			"complex with negation",
			"log not (type DESTROY and destination network PRIVATE)",
		},
		{
			"multiple conditions",
			"drop source network LOCAL and destination port 22,23,3389",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)
			rule, err := parser.ParseRule()
			require.NoError(t, err)
			assert.NotNil(t, rule)
		})
	}
}

func TestParser_InvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only action", "log"},
		{"missing action", "type NEW"},
		{"invalid keyword", "log typo NEW"},
		{"unclosed paren", "log (type NEW"},
		{"extra tokens", "log type NEW extra stuff"},
		{"invalid port", "log destination port abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			if err == nil {
				_, err = parser.ParseRule()
			}
			assert.Error(t, err)
		})
	}
}

func TestParser_InvalidTypeValidation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid type single", "log type FOO"},
		{"invalid type in list", "log type NEW,BAR"},
		{"invalid type complex", "log (destination address 78.47.60.169/32) and type NE"},
		{"typo in type", "log type NWE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			if err == nil {
				_, err = parser.ParseRule()
			}
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid event type")
		})
	}
}

func TestParser_InvalidProtocolValidation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid protocol single", "log protocol ICMP"},
		{"invalid protocol in list", "log protocol TCP,ICMP"},
		{"invalid protocol complex", "log destination address 1.2.3.4 and protocol FOO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			if err == nil {
				_, err = parser.ParseRule()
			}
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid protocol")
		})
	}
}

func TestParser_InvalidNetworkValidation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid network single", "log destination network INVALID"},
		{"invalid network in list", "log source network LOCAL,BOGUS"},
		{"invalid network complex", "log type NEW and destination network FOO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			if err == nil {
				_, err = parser.ParseRule()
			}
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid network type")
		})
	}
}
