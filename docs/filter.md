# Filter CEL Documentation

## Overview

The conntrackd filter uses CEL (Common Expression Language) to control
which conntrack events are **logged** to your configured sinks
(journal, syslog, Loki, etc.).

**Protocol Support:** Only TCP and UDP events are processed. All other protocols
(ICMP, IGMP, etc.) are automatically ignored and never logged, regardless of
filter rules.

**Important:** Filters do not affect network traffic - they only control which
conntrack events are logged. All network traffic flows normally regardless of
filter rules.

Rules are evaluated in order (first-match wins), and events are
**logged by default** when no rule matches.

## Command-Line Usage

Use the `--filter` flag to specify filter rules. This flag can be repeated
multiple times:

```bash
conntrackd run \
    --filter 'drop destination.address == "8.8.8.8"' \
    --filter 'log protocol == "TCP" && is_network(destination.address, "PUBLIC")' \
    --filter "drop any"
```

## Understanding Allow-by-Default

By default, conntrackd logs all conntrack events. This means:

- If no filters match an event, it **is logged** (allow-by-default)
- A `log` rule means "log this event"
- A `drop` rule means "don't log this event"

To log **only** specific events, use a `log` rule followed by `drop any` or
`drop true` to prevent logging of all other events.

```bash
# Log ONLY NEW TCP connections
--filter 'log event.type == "NEW" && protocol == "TCP"'
--filter "drop any"
```

Without the `drop any`, all non-matching events would still be logged.

## CEL Syntax

### Basic Structure

Each filter rule has two parts:
1. **Action**: `log` or `drop`
2. **Expression**: A CEL boolean expression

```
log <expression>
drop <expression>
```

### Available Variables

| Variable | Type | Description | Example Values |
|----------|------|-------------|----------------|
| `event.type` | string | Event type | "NEW", "UPDATE", "DESTROY" |
| `protocol` | string | Protocol | "TCP", "UDP" |
| `source.address` | string | Source IP address | "10.0.0.1", "2001:db8::1" |
| `destination.address` | string | Destination IP address | "8.8.8.8", "2600:1901::1" |
| `source.port` | int | Source port | 12345 |
| `destination.port` | int | Destination port | 80, 443 |

### Custom Functions

#### `is_network(ip, network_type)`

Checks if an IP address belongs to a network category.

**Parameters:**
- `ip` (string): IP address to check
- `network_type` (string): Network category

**Network Categories:**
- `"LOCAL"` - Loopback addresses (127.0.0.0/8, ::1) and link-local addresses
- `"PRIVATE"` - RFC1918 private addresses (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) and IPv6 ULA (fc00::/7)
- `"PUBLIC"` - Public addresses (not LOCAL, PRIVATE, or MULTICAST)
- `"MULTICAST"` - Multicast addresses (224.0.0.0/4, ff00::/8)

**Examples:**
```cel
is_network(source.address, "PRIVATE")
is_network(destination.address, "PUBLIC")
```

#### `in_cidr(ip, cidr)`

Checks if an IP address is within a CIDR range.

**Parameters:**
- `ip` (string): IP address to check
- `cidr` (string): CIDR notation (e.g., "192.168.1.0/24")

**Examples:**
```cel
in_cidr(destination.address, "8.8.8.0/24")
in_cidr(source.address, "2001:db8::/32")
```

#### `in_range(value, min, max)`

Checks if a numeric value is within a range (inclusive).

**Parameters:**
- `value` (int): Value to check
- `min` (int): Minimum value (inclusive)
- `max` (int): Maximum value (inclusive)

**Examples:**
```cel
in_range(destination.port, 8000, 8999)
in_range(source.port, 1024, 65535)
```

## Operators

### Comparison Operators

- `==` - Equal to
- `!=` - Not equal to
- `<` - Less than
- `<=` - Less than or equal to
- `>` - Greater than
- `>=` - Greater than or equal to

### Logical Operators

- `&&` - AND (both conditions must be true)
- `||` - OR (at least one condition must be true)
- `!` - NOT (negates a condition)

### Grouping

Use parentheses `()` to group expressions and control evaluation order.

## Examples

### Example 1: Don't Log Specific Destination

Don't log events to a specific IP address, but log all TCP traffic to public networks:

```bash
conntrackd run \
  --filter 'drop destination.address == "8.8.8.8"' \
  --filter 'log protocol == "TCP" && is_network(destination.address, "PUBLIC")'
```

**Evaluation:**
- Traffic to 8.8.8.8: Matches first rule → **NOT LOGGED**
- TCP to public IP (not 8.8.8.8): Matches second rule → **LOGGED**
- UDP to private network: No match → **LOGGED** (default)

### Example 2: Don't Log DNS to Specific Server

Don't log DNS traffic to a specific IP, log all other TCP/UDP:

```bash
conntrackd run \
  --filter 'drop destination.address == "10.19.80.100" && destination.port == 53' \
  --filter 'log protocol == "TCP" || protocol == "UDP"'
```

**Evaluation:**
- DNS to 10.19.80.100: Matches first rule → **NOT LOGGED**
- TCP/UDP to other destinations: Matches second rule → **LOGGED**
- Other protocols: No match → **LOGGED** (default)

### Example 3: Log Only Specific Traffic

Log only NEW TCP connections (don't log anything else):

```bash
conntrackd run \
  --filter 'log event.type == "NEW" && protocol == "TCP"' \
  --filter "drop any"
```

**Evaluation:**
- NEW TCP: Matches first rule → **LOGGED**
- NEW UDP: Matches second rule → **NOT LOGGED**
- UPDATE/DESTROY: Matches second rule → **NOT LOGGED**

**Note:** Without `drop any`, all non-matching events would still be logged.

### Example 4: Complex Filtering

Don't log outbound traffic to private networks on specific ports:

```bash
conntrackd run \
  --filter 'drop is_network(destination.address, "PRIVATE") && (destination.port == 22 || destination.port == 23 || destination.port == 3389)' \
  --filter 'log is_network(source.address, "PRIVATE")'
```

**Evaluation:**
- Private network destination on port 22: Matches first rule → **NOT LOGGED**
- Private network source: Matches second rule → **LOGGED**
- Other traffic: No match → **LOGGED** (default)

### Example 5: Port Range Filtering

Log only events to web ports:

```bash
conntrackd run \
  --filter 'log destination.port == 80 || destination.port == 443 || in_range(destination.port, 8000, 8999)' \
  --filter "drop any"
```

### Example 6: CIDR-based Filtering

Log traffic to specific subnets:

```bash
conntrackd run \
  --filter 'log in_cidr(destination.address, "192.168.1.0/24") || in_cidr(destination.address, "10.0.0.0/8")' \
  --filter "drop any"
```

### Example 7: IPv6 Filtering

Handle IPv6 addresses:

```bash
conntrackd run \
  --filter 'drop destination.address == "2001:4860:4860::8888"' \
  --filter 'log is_network(destination.address, "PUBLIC")'
```

### Example 8: Complex Negation

Log everything except traffic to private networks on SSH port:

```bash
conntrackd run \
  --filter 'drop is_network(destination.address, "PRIVATE") && destination.port == 22' \
  --filter "log any"
```

Alternatively using negation:

```bash
conntrackd run \
  --filter 'log !(is_network(destination.address, "PRIVATE") && destination.port == 22)'
```

## Migration from Old DSL

If you're migrating from the old DSL syntax, here are the key changes:

| Old DSL | New CEL |
|---------|---------|
| `type NEW` | `event.type == "NEW"` |
| `protocol TCP` | `protocol == "TCP"` |
| `source address 10.0.0.1` | `source.address == "10.0.0.1"` |
| `destination address 8.8.8.8` | `destination.address == "8.8.8.8"` |
| `source port 80` | `source.port == 80` |
| `destination port 443` | `destination.port == 443` |
| `source network PRIVATE` | `is_network(source.address, "PRIVATE")` |
| `destination network PUBLIC` | `is_network(destination.address, "PUBLIC")` |
| `destination address 8.8.8.0/24` | `in_cidr(destination.address, "8.8.8.0/24")` |
| `destination address 10.19.80.100 on port 53` | `destination.address == "10.19.80.100" && destination.port == 53` |
| `on port 53` | `source.port == 53 \|\| destination.port == 53` |
| `type NEW,UPDATE` | `event.type == "NEW" \|\| event.type == "UPDATE"` |
| `and` | `&&` |
| `or` | `\|\|` |
| `not type NEW` | `!(event.type == "NEW")` |
| `drop any` | `drop true` alias `drop any` |
| `log any` | `log true` alias `log any` |


## Best Practices

1. **Order Matters**: Place more specific rules before general rules
2. **Use `drop any` for Exclusive Logging**: When you want to log ONLY specific events, end with `drop any`
3. **Use `&&` for Precision**: Combine multiple conditions to create precise filters
4. **Test Incrementally**: Start with simple rules and add complexity
5. **Document Complex Rules**: Add comments in your deployment scripts
6. **Use Parentheses**: Make precedence explicit in complex expressions
7. **Quote Strings**: Always use double quotes for string literals

## Troubleshooting

### Common Errors

**Error: "Syntax error: extraneous input"**
- Make sure you're using CEL syntax, not the old DSL
- Check that string comparisons use `==` instead of space-separated values

**Error: "overlapping identifier for name 'type'"**
- `type` is a reserved keyword in CEL
- Use `event.type` instead

**Error: "no matching overload"**
- Check function arguments match the expected types
- Ensure custom functions are called with correct parameters

### Testing Filters

Start with simple filters and gradually add complexity:

```bash
# Start simple
--filter 'log protocol == "TCP"'

# Add conditions
--filter 'log protocol == "TCP" && destination.port == 443'

# Add network checks
--filter 'log protocol == "TCP" && is_network(destination.address, "PUBLIC")'

# Add final drop rule
--filter 'log protocol == "TCP" && is_network(destination.address, "PUBLIC")'
--filter "drop any"
```
