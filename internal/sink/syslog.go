/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"log/slog"
	"net"
	"net/url"
	"strings"

	slogsyslog "github.com/samber/slog-syslog/v2"
)

type Syslog struct {
	Enable  bool
	Address string
}

var SyslogProtocols = []string{"udp", "tcp", "unix", "unixgram", "unixpacket"}

func (s *Syslog) TargetSyslog(options *slog.HandlerOptions) (slog.Handler, error) {
	url, err := url.Parse(s.Address)
	if err != nil {
		return nil, err
	}

	network := url.Scheme
	address := url.Host
	if strings.HasPrefix(network, "unix") {
		address = url.Path
	}

	writer, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	slogsyslog.ContextKey = "event"
	o := &slogsyslog.Option{
		Writer: writer,
		Level:  options.Level,
	}
	return o.NewSyslogHandler(), nil
}
