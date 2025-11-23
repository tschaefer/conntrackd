/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package record

import (
	"fmt"
	"log/slog"
	"syscall"

	"github.com/ti-mo/conntrack"
	"github.com/tschaefer/conntrackd/internal/geoip"
)

func Record(event conntrack.Event, geo *geoip.GeoIP, logger *slog.Logger) {
	slog.Debug("Conntrack Event", "data", event)

	protocols := map[int]string{
		syscall.IPPROTO_TCP: "TCP",
		syscall.IPPROTO_UDP: "UDP",
	}
	prot := protocols[int(event.Flow.TupleOrig.Proto.Protocol)]

	var eType string
	switch event.Type {
	case conntrack.EventNew:
		eType = "NEW"
	case conntrack.EventUpdate:
		eType = "UPDATE"
	case conntrack.EventDestroy:
		eType = "DESTROY"
	}

	established := []any{
		slog.String("type", eType),
		slog.Uint64("flow", uint64(event.Flow.ID)),
		slog.String("prot", prot),
		slog.String("src", event.Flow.TupleOrig.IP.SourceAddress.String()),
		slog.String("dst", event.Flow.TupleOrig.IP.DestinationAddress.String()),
		slog.Uint64("sport", uint64(event.Flow.TupleOrig.Proto.SourcePort)),
		slog.Uint64("dport", uint64(event.Flow.TupleOrig.Proto.DestinationPort)),
	}

	if event.Flow.ProtoInfo.TCP != nil {
		states := map[int]string{
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
		state := states[int(event.Flow.ProtoInfo.TCP.State)]
		established = append(established, slog.String("state", state))
	}

	var location []any
	if geo != nil {
		loc := geo.Location(event.Flow.TupleOrig.IP.DestinationAddress)
		if loc != nil {
			location = []any{
				slog.String("city", loc.City),
				slog.String("country", loc.Country),
				slog.Float64("lat", loc.Lat),
				slog.Float64("lon", loc.Lon),
			}
		}
	}

	msg := fmt.Sprintf("%s %s connection from %s/%d to %s/%d",
		eType, prot,
		event.Flow.TupleOrig.IP.SourceAddress.String(),
		event.Flow.TupleOrig.Proto.SourcePort,
		event.Flow.TupleOrig.IP.DestinationAddress.String(),
		event.Flow.TupleOrig.Proto.DestinationPort,
	)

	logger.Info(msg, append(established, location...)...)
}
