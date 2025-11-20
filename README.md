# conntrackd

[![Tag](https://img.shields.io/github/tag/tschaefer/conntrackd.svg)](https://github.com/tschaefer/conntrackd/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.25-%23007d9c)
[![Go Report Card](https://goreportcard.com/badge/github.com/tschaefer/conntrackd)](https://goreportcard.com/report/github.com/tschaefer/conntrackd)
[![Coverage](https://img.shields.io/codecov/c/github/tschaefer/conntrackd)](https://codecov.io/gh/tschaefer/conntrackd)
[![Contributors](https://img.shields.io/github/contributors/tschaefer/conntrackd)](https://github.com/tschaefer/conntrackd/graphs/contributors)
[![License](https://img.shields.io/github/license/tschaefer/conntrackd)](./LICENSE)

**conntrackd** is a small, efficient conntrack event fanout logger written in
Go. It listens for Linux conntrack/netfilter connection tracking events and
optional enriches them with GEO location information before emitting structured
logs. It's intended for lightweight monitoring, auditing, and integration with
log pipelines.

## Features

- Listen for conntrack events (new/updated/destroyed connections)
- Enrich IP addresses with GEO location data
- Fanout to multiple log sinks (stream, syslog,
  [journald](https://www.freedesktop.org/software/systemd/man/latest/systemd-journald.html),
  [Loki](https://grafana.com/docs/loki/latest/))

# Getting Started

## Prerequisites

- Linux (netlink/conntrack support required)
- Root privileges
- (Optional) MaxMind GeoIP2/GeoLite2 City database

## Installation and Usage

Download the latest release from the
[releases page](https://github.com/tschaefer/conntrackd/releases).

Start the event listener and logger.

```bash
sudo conntrackd run --sink.journal.enable
```
For further configuration, see the command-line options below.

| Flag                             | Description                        | Options                                |
|----------------------------------|------------------------------------|----------------------------------------|
| `--filter.destinations`          | Filter by destination networks     | PUBLIC,PRIVATE,LOCAL,MULTICAST         |
| `--filter.sourcess`              | Filter by source networks          | PUBLIC,PRIVATE,LOCAL,MULTICAST         |
| `--filter.protocols`             | Filter by protocols                | TCP,UDP                                |
| `--filter.types`                 | Filter by event types              | NEW,UPDATE,DESTROY                     |
| `--filter.destination.addresses` | Filter by destination IP addresses |                                        |
| `--filter.source.addresses`      | Filter by source IP addresses      |                                        |
| `--filter.destination.ports`     | Filter by destination ports        |                                        |
| `--filter.source.ports`          | Filter by source ports             |                                        |
| `--geoip.database`               | Path to GeoIP database             |                                        |
| `--service.log.format`           | Log format                         | json,text; default: text               |
| `--service.log.level`            | Log level                          | trace,debug,info,error; default: info  |
| `--sink.journal.enable`          | Enable journald sink               |                                        |
| `--sink.syslog.enable`           | Enable syslog sink                 |                                        |
| `--sink.enable.loki`             | Enable Loki sink                   |                                        |
| `--sink.stream.enable`           | Enable stream sink                 |                                        |
| `--sink.syslog.address`          | Syslog address                     | default: udp://localhost:514           |
| `--sink.loki.address`            | Loki address                       | default: http://localhost:3100         |
| `--sink.loki.labels`             | Loki labels                        | comma seperated key=value pairs        |
| `--sink.stream.writer`           | Stream writer type                 | stdout,stderr,discard; default: stdout |

All filters are exclusive; if any filter is not set, all related events are processed.

Example run:

```bash
sudo conntrackd run \
  --geoip.database /usr/local/share/GeoLite2-City.mmdb \
  --filter.destination PRIVATE \
  --filter.protocol UDP \
  --filter.destination.addresses 142.250.186.163,2a00:1450:4001:82b::2003
  --sink.journal.enable \
  --service.log.format json \
  --service.log.level debug
```

## Logging format

conntrackd emits structured logs for each conntrack event. A typical log entry
includes:

- type (connection event type)
- flow (connection flow identifier)
- src, dst (IP addresses)
- sport, dport (port numbers)
- prot (transport protocol)

Additionally TCP field:

- state (TCP connection state)

GEO location fields:

- city (city name)
- country (country name)
- lat (latitude)
- lon (longitude)

Example log entry recorded by sink `syslog`:

```json
{
  "event": {
    "dport": 443,
    "dst": "2600:1901:0:b3ea::",
    "flow": 221193769,
    "prot": "TCP",
    "sport": 41348,
    "src": "2003:cf:1716:7b64:da80:83ff:fecd:da51",
    "state": "LAST_ACK",
    "type": "UPDATE"
  },
  "level": "INFO",
  "logger.name": "samber/slog-syslog",
  "logger.version": "v2.5.2",
  "message": "UPDATE TCP connection from 2003:cf:1716:7b64:da80:83ff:fecd:da51/41348...",
  "timestamp": "2025-11-15T09:55:25.647544937Z"
}
```

Example log entry recorded by sink `journal`:

```json
{
	"__CURSOR" : "s=b3c7821dbfce47a59b06797aea9028ca;i=6772d3;b=100da27bd8...",
	"_CAP_EFFECTIVE" : "1ffffffffff",
	"EVENT_SPORT" : "39790",
	"_SOURCE_REALTIME_TIMESTAMP" : "1763200187611509",
	"_SYSTEMD_CGROUP" : "/user.slice/user-1000.slice/session-1.scope",
	"_SYSTEMD_OWNER_UID" : "1000",
	"_SYSTEMD_SESSION" : "1",
	"_EXE" : "/home/tschaefer/.env/bin/conntrackd",
	"_HOSTNAME" : "bullseye",
	"_GID" : "0",
	"PRIORITY" : "6",
	"_SYSTEMD_UNIT" : "session-1.scope",
	"EVENT_DPORT" : "443",
	"SLOG_LOGGER" : "tschaefer/slog-journal:v1.0.0",
	"_TRANSPORT" : "journal",
	"EVENT_SRC" : "2003:cf:1716:7b64:da80:83ff:fecd:da51",
	"_COMM" : "conntrackd",
	"__MONOTONIC_TIMESTAMP" : "352829248481",
	"EVENT_STATE" : "LAST_ACK",
	"_MACHINE_ID" : "75b649379b874beea04d95463e59c3a1",
	"_SYSTEMD_SLICE" : "user-1000.slice",
	"_SYSTEMD_USER_SLICE" : "-.slice",
	"__SEQNUM_ID" : "b3c7821dbfce47a59b06797aea9028ca",
	"__REALTIME_TIMESTAMP" : "1763200187611631",
	"__SEQNUM" : "6779603",
	"_SYSTEMD_INVOCATION_ID" : "021760b3373342b98aaeabf9d12d8d74",
	"EVENT_FLOW" : "3478798157",
	"_PID" : "3794900",
	"_CMDLINE" : "conntrackd run --service.log.level debug --service.log.format ...",
	"EVENT_PROT" : "TCP",
	"_AUDIT_SESSION" : "1",
	"_BOOT_ID" : "100da27bd8b94096b5c80cdac34d6063",
	"_RUNTIME_SCOPE" : "system",
	"_SELINUX_CONTEXT" : "unconfined\n",
	"EVENT_DST" : "2600:1901:0:b3ea::",
	"_AUDIT_LOGINUID" : "1000",
	"_UID" : "0",
	"EVENT_TYPE" : "UPDATE",
	"MESSAGE" : "UPDATE TCP connection from 2003:cf:1716:7b64:da80:83ff:fecd:da51/39790..."
}

```

Example log entry recorded by sink `loki`:

```json
{
  "stream": {
    "city": "Nuremberg",
    "country": "Germany",
    "detected_level": "INFO",
    "dport": "443",
    "dst": "2a01:4f8:1c1c:b751::1",
    "flow": "574674164",
    "host": "core.example.com",
    "lat": "49.4527",
    "level": "INFO",
    "lon": "11.0783",
    "prot": "TCP",
    "service_name": "conntrackd",
    "sport": "44950",
    "src": "2003:cf:1716:7b64:d6e9:8aff:fe4f:7a59",
    "state": "TIME_WAIT",
    "type": "UPDATE"
  },
  "values": [
    [
      "1763537351540294198",
      "UPDATE TCP connection from 2003:cf:1716:7b64:d6e9:8aff:fe4f:7a59/44950..."
    ]
  ]
}
```

## Security Notes

- Observing conntrack/netlink events typically requires elevated privileges.
- Keep GeoIP databases updated.
- Be careful with log storage; connection events may contain sensitive network
  metadata.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.
For major changes, open an issue first to discuss what you would like to change.

Ensure that your code adheres to the existing style and includes appropriate tests.

## License

This project is licensed under the [MIT License](LICENSE).
