/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"syscall"

	"github.com/ti-mo/conntrack"
)

// PredicateFunc is a function that evaluates a predicate against an event
type PredicateFunc func(event conntrack.Event) bool

// Compile compiles an expression AST into a predicate function
func Compile(expr ExprNode) (PredicateFunc, error) {
	switch e := expr.(type) {
	case BinaryExpr:
		return compileBinaryExpr(e)
	case UnaryExpr:
		return compileUnaryExpr(e)
	case TypePredicate:
		return compileTypePredicate(e), nil
	case ProtocolPredicate:
		return compileProtocolPredicate(e), nil
	case NetworkPredicate:
		return compileNetworkPredicate(e), nil
	case AddressPredicate:
		return compileAddressPredicate(e)
	case PortPredicate:
		return compilePortPredicate(e), nil
	case AnyPredicate:
		return compileAnyPredicate(e), nil
	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

func compileBinaryExpr(expr BinaryExpr) (PredicateFunc, error) {
	left, err := Compile(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := Compile(expr.Right)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case OpAnd:
		return func(event conntrack.Event) bool {
			return left(event) && right(event)
		}, nil
	case OpOr:
		return func(event conntrack.Event) bool {
			return left(event) || right(event)
		}, nil
	default:
		return nil, fmt.Errorf("unknown binary operator: %v", expr.Op)
	}
}

func compileUnaryExpr(expr UnaryExpr) (PredicateFunc, error) {
	inner, err := Compile(expr.Expr)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case OpNot:
		return func(event conntrack.Event) bool {
			return !inner(event)
		}, nil
	default:
		return nil, fmt.Errorf("unknown unary operator: %v", expr.Op)
	}
}

func compileTypePredicate(pred TypePredicate) PredicateFunc {
	return func(event conntrack.Event) bool {
		var eventType string
		switch event.Type {
		case conntrack.EventNew:
			eventType = "NEW"
		case conntrack.EventUpdate:
			eventType = "UPDATE"
		case conntrack.EventDestroy:
			eventType = "DESTROY"
		default:
			return false
		}
		return slices.Contains(pred.Types, eventType)
	}
}

func compileProtocolPredicate(pred ProtocolPredicate) PredicateFunc {
	return func(event conntrack.Event) bool {
		var protocol string
		switch event.Flow.TupleOrig.Proto.Protocol {
		case syscall.IPPROTO_TCP:
			protocol = "TCP"
		case syscall.IPPROTO_UDP:
			protocol = "UDP"
		default:
			return false
		}
		return slices.Contains(pred.Protocols, protocol)
	}
}

func compileNetworkPredicate(pred NetworkPredicate) PredicateFunc {
	return func(event conntrack.Event) bool {
		var ip netip.Addr
		switch strings.ToLower(pred.Direction) {
		case "source", "src":
			ip = event.Flow.TupleOrig.IP.SourceAddress
		case "destination", "dst", "dest":
			ip = event.Flow.TupleOrig.IP.DestinationAddress
		default:
			return false
		}

		isLocal := ip.IsLoopback() || ip.IsLinkLocalUnicast()
		isPrivate := ip.IsPrivate()
		isMulticast := ip.IsMulticast()
		isPublic := !isLocal && !isPrivate && !isMulticast

		for _, network := range pred.Networks {
			switch network {
			case "LOCAL":
				if isLocal {
					return true
				}
			case "PRIVATE":
				if isPrivate {
					return true
				}
			case "MULTICAST":
				if isMulticast {
					return true
				}
			case "PUBLIC":
				if isPublic {
					return true
				}
			}
		}
		return false
	}
}

func compileAddressPredicate(pred AddressPredicate) (PredicateFunc, error) {
	// Pre-compile address matchers
	var matchers []func(netip.Addr) bool
	for _, addrStr := range pred.Addresses {
		// Try to parse as CIDR first
		if strings.Contains(addrStr, "/") {
			prefix, err := netip.ParsePrefix(addrStr)
			if err == nil {
				// Capture prefix in a local variable to avoid loop variable capture
				pfx := prefix
				matchers = append(matchers, func(ip netip.Addr) bool {
					return pfx.Contains(ip)
				})
				continue
			}
		}
		// Try to parse as IP
		addr, err := netip.ParseAddr(addrStr)
		if err == nil {
			// Capture addr in a local variable to avoid loop variable capture
			a := addr
			matchers = append(matchers, func(ip netip.Addr) bool {
				return ip == a
			})
		}
	}

	// If no addresses parsed successfully, return error
	if len(matchers) == 0 {
		return nil, fmt.Errorf("no valid addresses in predicate")
	}

	return func(event conntrack.Event) bool {
		var ip netip.Addr
		var port uint16

		switch strings.ToLower(pred.Direction) {
		case "source", "src":
			ip = event.Flow.TupleOrig.IP.SourceAddress
			port = event.Flow.TupleOrig.Proto.SourcePort
		case "destination", "dst", "dest":
			ip = event.Flow.TupleOrig.IP.DestinationAddress
			port = event.Flow.TupleOrig.Proto.DestinationPort
		default:
			return false
		}

		// Check if IP matches
		matched := false
		for _, matcher := range matchers {
			if matcher(ip) {
				matched = true
				break
			}
		}

		if !matched {
			return false
		}

		// If ports are specified, check them too
		if len(pred.Ports) > 0 {
			return slices.Contains(pred.Ports, port)
		}

		return true
	}, nil
}

func compilePortPredicate(pred PortPredicate) PredicateFunc {
	return func(event conntrack.Event) bool {
		srcPort := event.Flow.TupleOrig.Proto.SourcePort
		dstPort := event.Flow.TupleOrig.Proto.DestinationPort

		switch strings.ToLower(pred.Direction) {
		case "source", "src":
			return slices.Contains(pred.Ports, srcPort)
		case "destination", "dst", "dest":
			return slices.Contains(pred.Ports, dstPort)
		case "both":
			return slices.Contains(pred.Ports, srcPort) || slices.Contains(pred.Ports, dstPort)
		default:
			return false
		}
	}
}

func compileAnyPredicate(pred AnyPredicate) PredicateFunc {
	// AnyPredicate always matches
	return func(event conntrack.Event) bool {
		return true
	}
}
