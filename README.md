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

## Filtering

conntrackd logs conntrack events to various sinks.

**Protocol Support:** Only TCP and UDP events are processed. All other protocols
(ICMP, IGMP, etc.) are automatically ignored and never logged, regardless of filter rules.

You can use filters to control which TCP/UDP events are logged using a
Domain-Specific Language (DSL).
The `--filter` flag lets you specify filter rules:

```bash
sudo conntrackd run \
  --filter "drop destination address 8.8.8.8" \
  --filter "log protocol TCP and destination network PUBLIC" \
  --filter "drop any" \
  --sink.journal.enable
```

**Filter Rules:**
- Rules are evaluated in order (first-match wins)
- Events are **logged by default** when no rule matches
- `--filter` flag can be repeated for multiple rules
- Use `drop any` as a final rule to block all non-matching events from being logged

**Important:** Filters control which conntrack events are **logged**,
not network traffic. Traffic always flows normally; filters only affect logging.

**Common Filter Examples:**

```bash
# Don't log events to a specific IP
--filter "drop destination address 8.8.8.8"

# Log only NEW TCP connections (deny everything else)
--filter "log type NEW and protocol TCP"
--filter "drop any"

# Don't log DNS to specific server
--filter "drop destination address 10.19.80.100 on port 53"

# Don't log any traffic to private networks
--filter "drop destination network PRIVATE"

# Log only traffic from public IPs using TCP
--filter "log source network PUBLIC and protocol TCP"
--filter "drop any"
```

See [docs/filter.md](docs/filter.md) for complete DSL documentation,
including grammar, operators, and advanced examples.

## Configuration

conntrackd can be configured via command-line flags, configuration files,
environment variables, or a combination of these methods.

### Configuration Files

By default, conntrackd searches for a configuration file named
`conntrackd.(yaml|yml|json|toml)` in `/etc/conntrackd` directory.

You can also specify a custom config file using the `--config` flag:

```bash
sudo conntrackd run --config /path/to/config.yaml
```
Configuration files support YAML, JSON, and TOML formats.
See [contrib/config.yaml](contrib/config.yaml) for a complete example
configuration file.

### Environment Variables

Configuration values can be set via environment variables with the
`CONNTRACKD_` prefix:

```bash
export CONNTRACKD_LOG_LEVEL=debug
export CONNTRACKD_SINK_STREAM_WRITER=discard
sudo -E conntrackd run
```

Use underscores (`_`) to represent nested keys:
`sink.stream.writer` → `CONNTRACKD_SINK_STREAM_WRITER`

### Priority Order

Configuration values are applied in the following order
(later overrides earlier):

1. Default values
2. Configuration file
3. Environment variables
4. Command-line flags

**Note:** Command-line flags always have the highest priority.

## Configuration Flags

| Flag                    | Description                                       | Default                  |
|-------------------------|---------------------------------------------------|--------------------------|
| `--config`              | Path to configuration file                        |                          |
| `--filter`              | Filter rule in DSL format (repeatable)            |                          |
| `--geoip.database`      | Path to GeoIP database                            |                          |
| `--log.level`           | Log level (debug, info, warn, error)              | info                     |
| `--sink.journal.enable` | Enable journald sink                              |                          |
| `--sink.syslog.enable`  | Enable syslog sink                                |                          |
| `--sink.loki.enable`    | Enable Loki sink                                  |                          |
| `--sink.stream.enable`  | Enable stream sink                                |                          |
| `--sink.syslog.address` | Syslog address                                    | udp://localhost:514      |
| `--sink.loki.address`   | Loki address                                      | http://localhost:3100    |
| `--sink.loki.labels`    | Loki labels (comma-separated key=value pairs)     |                          |
| `--sink.stream.writer`  | Stream writer (stdout, stderr, discard)           | stdout                   |

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

GEO location fields for source and destination if applicable:

- city (city name)
- country (country name)
- lat (latitude)
- lon (longitude)

<details>
<summary>Example log entry recorded by sink `syslog`</summary>

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
  "message": "UPDATE TCP connection from [2003:cf:1716:7b64:da80:83ff:fecd...",
  "timestamp": "2025-11-15T09:55:25.647544937Z"
}
```
</details>

<details>
<summary>Example log entry recorded by sink `journal`</summary>

```json
{
	"__CURSOR" : "s=b3c7821dbfce47a59b06797aea9028ca;i=6772d3;b=100da27bd...",
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
	"_CMDLINE" : "conntrackd run --service.log.level debug --service.log....",
	"EVENT_PROT" : "TCP",
	"_AUDIT_SESSION" : "1",
	"_BOOT_ID" : "100da27bd8b94096b5c80cdac34d6063",
	"_RUNTIME_SCOPE" : "system",
	"_SELINUX_CONTEXT" : "unconfined\n",
	"EVENT_DST" : "2600:1901:0:b3ea::",
	"_AUDIT_LOGINUID" : "1000",
	"_UID" : "0",
	"EVENT_TYPE" : "UPDATE",
	"MESSAGE" : "UPDATE TCP connection from [2003:cf:1716:7b64:da80:83ff:fe..."
}

```
</details>

<details>
<summary>Example log entry recorded by sink `loki`</summary>

```json
{
  "stream": {
    "dcity": "Falkenstein",
    "dcountry": "Germany",
    "detected_level": "INFO",
    "dlat": "50.4777",
    "dlon": "12.3649",
    "dport": "443",
    "dst": "2a01:4f8:160:5372::2",
    "flow": "122448605",
    "host": "core.example",
    "level": "INFO",
    "prot": "TCP",
    "scity": "Falkenstein",
    "scountry": "Germany",
    "service_name": "conntrackd",
    "slat": "50.4777",
    "slon": "12.3649",
    "sport": "54756",
    "src": "2003:cf:1716:7b64:da80:83ff:fecd:da51",
    "state": "SYN_SENT",
    "type": "NEW"
  },
  "values": [
    [
      "1764068411229712376",
      "NEW TCP connection from [2003:cf:1716:7b64:da80:83ff:fecd:da51]:54756..."
    ]
  ]
}

```
</details>

<details>
<summary>Example log entry recorded by sink `stream`</summary>

```json
{
  "time": "2025-11-25T11:58:21.99256561+01:00",
  "level": "INFO",
  "msg": "NEW TCP connection from [2003:cf:1716:7b64:da80:83ff:fecd:da51]:4...",
  "type": "NEW",
  "flow": 901634022,
  "prot": "TCP",
  "src": "2003:cf:1716:7b64:da80:83ff:fecd:da51",
  "dst": "2a01:4f8:160:5372::2",
  "sport": 44646,
  "dport": 443,
  "state": "SYN_SENT",
  "scity": "Falkenstein",
  "scountry": "Germany",
  "slat": 50.4777,
  "slon": 12.3649,
  "dcity": "Falkenstein",
  "dcountry": "Germany",
  "dlat": 50.4777,
  "dlon": 12.3649
}
```
</details>


## Security Notes

- Observing conntrack/netlink events typically requires elevated privileges.
- Keep GeoIP databases updated.
- Be careful with log storage; connection events may contain sensitive network
  metadata.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.
For major changes, open an issue first to discuss what you would like to change.

Ensure that your code adheres to the existing style and includes appropriate
tests.

## License

This project is licensed under the [MIT License](LICENSE).
