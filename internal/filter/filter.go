/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package filter

import (
	"slices"
	"syscall"

	"github.com/ti-mo/conntrack"
)

type FilterAddresses struct {
	Destinations []string
	Sources      []string
}

type Filter struct {
	EventTypes   []string
	Protocols    []string
	Destinations []string
	Sources      []string
	Addresses    FilterAddresses
}

func (f *Filter) eventType(event conntrack.Event) bool {
	if len(f.EventTypes) == 0 {
		return true
	}

	types := map[any]string{
		conntrack.EventNew:     "NEW",
		conntrack.EventUpdate:  "UPDATE",
		conntrack.EventDestroy: "DESTROY",
	}
	eventTypeStr, ok := types[event.Type]
	if !ok {
		return false
	}

	return slices.Contains(f.EventTypes, eventTypeStr)
}

func (f *Filter) eventProtocol(event conntrack.Event) bool {
	if len(f.Protocols) == 0 {
		switch event.Flow.TupleOrig.Proto.Protocol {
		case syscall.IPPROTO_TCP, syscall.IPPROTO_UDP:
			return true
		default:
			return false
		}
	}

	protocols := map[int]string{
		syscall.IPPROTO_TCP: "TCP",
		syscall.IPPROTO_UDP: "UDP",
	}
	protocolStr, ok := protocols[int(event.Flow.TupleOrig.Proto.Protocol)]
	if !ok {
		return false
	}

	return slices.Contains(f.Protocols, protocolStr)
}

func (f *Filter) eventSource(event conntrack.Event) bool {
	if len(f.Sources) == 0 {
		return true
	}

	dest := event.Flow.TupleOrig.IP.SourceAddress
	isLocal := dest.IsLoopback()
	isPrivate := dest.IsPrivate()
	isMulticast := dest.IsMulticast()
	isPublic := !isLocal && !isPrivate && !isMulticast

	for _, filterSource := range f.Sources {
		switch filterSource {
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

func (f *Filter) eventDestination(event conntrack.Event) bool {
	if len(f.Destinations) == 0 {
		return true
	}

	dest := event.Flow.TupleOrig.IP.DestinationAddress
	isLocal := dest.IsLoopback()
	isPrivate := dest.IsPrivate()
	isMulticast := dest.IsMulticast()
	isPublic := !isLocal && !isPrivate && !isMulticast

	for _, filterDest := range f.Destinations {
		switch filterDest {
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

func (f *Filter) eventAddressDestination(event conntrack.Event) bool {
	if len(f.Addresses.Destinations) == 0 {
		return true
	}

	return !slices.Contains(f.Addresses.Destinations, event.Flow.TupleOrig.IP.DestinationAddress.String())
}

func (f *Filter) eventAddressSource(event conntrack.Event) bool {
	if len(f.Addresses.Sources) == 0 {
		return true
	}

	return !slices.Contains(f.Addresses.Sources, event.Flow.TupleOrig.IP.SourceAddress.String())
}

func (f *Filter) Apply(event conntrack.Event) bool {
	if !f.eventType(event) {
		return false
	}

	if !f.eventProtocol(event) {
		return false
	}

	if !f.eventSource(event) {
		return false
	}

	if !f.eventDestination(event) {
		return false
	}

	if !f.eventAddressDestination(event) {
		return false
	}

	if !f.eventAddressSource(event) {
		return false
	}

	return true
}
