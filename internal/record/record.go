/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package record

import (
	"fmt"
	"log/slog"
	"net/netip"
	"syscall"

	"github.com/ti-mo/conntrack"
	"github.com/tschaefer/conntrackd/internal/geoip"
)

func Record(event conntrack.Event, geo *geoip.GeoIP, logger *slog.Logger) {
	slog.Debug("Conntrack Event", "data", event)

	prot := getProtocol(event)
	eType := getType(event)

	established := []any{
		slog.String("type", eType),
		slog.Uint64("flow", uint64(event.Flow.ID)),
		slog.String("prot", prot),
		slog.String("src", event.Flow.TupleOrig.IP.SourceAddress.String()),
		slog.String("dst", event.Flow.TupleOrig.IP.DestinationAddress.String()),
		slog.Uint64("sport", uint64(event.Flow.TupleOrig.Proto.SourcePort)),
		slog.Uint64("dport", uint64(event.Flow.TupleOrig.Proto.DestinationPort)),
	}

	state, ok := getTCPState(event)
	if ok {
		established = append(established, slog.String("state", state))
	}

	location := getLocation(event, geo)

	msg := fmt.Sprintf("%s %s connection from %s to %s",
		eType, prot,
		formatAddrPort(
			event.Flow.TupleOrig.IP.SourceAddress,
			event.Flow.TupleOrig.Proto.SourcePort,
		),
		formatAddrPort(
			event.Flow.TupleOrig.IP.DestinationAddress,
			event.Flow.TupleOrig.Proto.DestinationPort,
		),
	)

	logger.Info(msg, append(established, location...)...)
}

func getProtocol(event conntrack.Event) string {
	protocols := map[int]string{
		syscall.IPPROTO_TCP: "TCP",
		syscall.IPPROTO_UDP: "UDP",
	}
	if prot, ok := protocols[int(event.Flow.TupleOrig.Proto.Protocol)]; ok {
		return prot
	}
	return ""
}

func getType(event conntrack.Event) string {
	switch event.Type {
	case conntrack.EventNew:
		return "NEW"
	case conntrack.EventUpdate:
		return "UPDATE"
	case conntrack.EventDestroy:
		return "DESTROY"
	default:
		return ""
	}
}

func getTCPState(event conntrack.Event) (string, bool) {
	if event.Flow.ProtoInfo.TCP == nil {
		return "", false
	}

	state := event.Flow.ProtoInfo.TCP.State
	states := map[uint8]string{
		0: "NONE",
		1: "SYN_SENT",
		2: "SYN_RECV",
		3: "ESTABLISHED",
		4: "FIN_WAIT",
		5: "CLOSE_WAIT",
		6: "LAST_ACK",
		7: "TIME_WAIT",
		8: "CLOSE",
	}
	if s, ok := states[state]; ok {
		return s, true
	}

	return "", true
}

func getLocation(event conntrack.Event, geo *geoip.GeoIP) []any {
	if geo == nil {
		return nil
	}

	var locGroups []any
	for dir, addr := range map[string]netip.Addr{
		"src": event.Flow.TupleOrig.IP.SourceAddress,
		"dst": event.Flow.TupleOrig.IP.DestinationAddress,
	} {
		if loc := geo.Location(addr); loc != nil {
			locGroups = append(locGroups, slog.Group(dir,
				slog.String("city", loc.City),
				slog.String("country", loc.Country),
				slog.Float64("lat", loc.Lat),
				slog.Float64("lon", loc.Lon),
			))
		}
	}

	if len(locGroups) == 0 {
		return nil
	}

	return []any{slog.Group("location", locGroups...)}
}

func formatAddrPort(addr netip.Addr, port uint16) string {
	if addr.Is6() {
		return fmt.Sprintf("[%s]:%d", addr.String(), port)
	}
	return fmt.Sprintf("%s:%d", addr.String(), port)
}
