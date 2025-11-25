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
		slog.String("src_addr", event.Flow.TupleOrig.IP.SourceAddress.String()),
		slog.String("dst_addr", event.Flow.TupleOrig.IP.DestinationAddress.String()),
		slog.Uint64("src_port", uint64(event.Flow.TupleOrig.Proto.SourcePort)),
		slog.Uint64("dst_port", uint64(event.Flow.TupleOrig.Proto.DestinationPort)),
	}

	state, ok := getTCPState(event)
	if ok {
		established = append(established, slog.String("tcp_state", state))
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

	var location []any
	for dir, addr := range map[string]netip.Addr{
		"src_": event.Flow.TupleOrig.IP.SourceAddress,
		"dst_": event.Flow.TupleOrig.IP.DestinationAddress,
	} {
		if loc := geo.Location(addr); loc != nil {
			data := []any{
				slog.String(dir+"city", loc.City),
				slog.String(dir+"country", loc.Country),
				slog.Float64(dir+"lat", loc.Lat),
				slog.Float64(dir+"lon", loc.Lon),
			}
			location = append(location, data...)
		}
	}

	if len(location) == 0 {
		return nil
	}

	return location
}

func formatAddrPort(addr netip.Addr, port uint16) string {
	if addr.Is6() {
		return fmt.Sprintf("[%s]:%d", addr.String(), port)
	}
	return fmt.Sprintf("%s:%d", addr.String(), port)
}
