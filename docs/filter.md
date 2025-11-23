# Filter DSL Documentation

## Overview

The conntrackd filter DSL (Domain-Specific Language) allows you to control
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
    --filter "drop destination address 8.8.8.8" \
    --filter "log protocol TCP" \
    --filter "drop ANY"
```

## Understanding Allow-by-Default

By default, conntrackd logs all conntrack events. This means:

- If no filters match an event, it **is logged** (allow-by-default)
- An `log` rule means "log this event"
- A `drop` rule means "don't log this event"

To log **only** specific events, use an `log` rule followed by `drop ANY`:

```bash
# Log ONLY NEW TCP connections
--filter "log type NEW and protocol TCP"
--filter "drop ANY"
```

Without the `drop ANY`, all non-matching events would still be logged.

## Grammar

The filter DSL follows this grammar (EBNF notation):

```ebnf
rule       ::= action expression
action     ::= "log" | "drop"
expression ::= orExpr
orExpr     ::= andExpr { "or" andExpr }
andExpr    ::= notExpr { "and" notExpr }
notExpr    ::= [ "not" | "!" ] primary
primary    ::= predicate | "(" expression ")"

predicate  ::= eventPred | protoPred | addrPred | networkPred | portPred | anyPred

eventPred  ::= "type" identList
protoPred  ::= "protocol" identList
addrPred   ::= direction "address" addrList [ "on" "port" portSpec ]
networkPred::= direction "network" identList
portPred   ::= [ direction ] "port" portSpec
             | "on" "port" portSpec
anyPred    ::= "ANY"

direction  ::= "source" | "src" | "destination" | "dst" | "dest"
identList  ::= IDENT { "," IDENT }
addrList   ::= ADDRESS { "," ADDRESS }
portSpec   ::= NUMBER | NUMBER "-" NUMBER | NUMBER { "," NUMBER }
```

## Predicates

### Any (Catch-All)

The `ANY` predicate matches all events. It's typically used with `drop` to
block all non-matching events:

```bash
# Log only NEW TCP connections (deny everything else)
log type NEW and protocol TCP
drop ANY

# Log only traffic to specific IPs (deny everything else)
log destination address 1.2.3.4,5.6.7.8
drop ANY
```

### Event Type

Match on conntrack event types:

```bash
# Deny NEW events
drop type NEW

# Allow UPDATE or DESTROY events
log type UPDATE,DESTROY
```

Valid types: `NEW`, `UPDATE`, `DESTROY`

### Protocol

Match on protocol:

```bash
# Deny TCP events
drop protocol TCP

# Allow TCP or UDP
log protocol TCP,UDP
```

Valid protocols: `TCP`, `UDP`

### Network Classification

Match on network categories:

```bash
# Deny traffic to private networks
drop destination network PRIVATE

# Allow traffic from public networks
log source network PUBLIC
```

Valid network types:
- `LOCAL` - Loopback addresses (127.0.0.0/8, ::1) and link-local addresses
- `PRIVATE` - RFC1918 private addresses (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) and IPv6 ULA (fc00::/7)
- `PUBLIC` - Public addresses (not LOCAL, PRIVATE, or MULTICAST)
- `MULTICAST` - Multicast addresses (224.0.0.0/4, ff00::/8)

### IP Address

Match on specific IP addresses or CIDR ranges:

```bash
# Deny traffic to specific IP
drop destination address 8.8.8.8

# Allow traffic from a CIDR range
log source address 192.168.1.0/24

# Match IPv6 addresses
drop destination address 2001:4860:4860::8888

# Multiple addresses
drop destination address 1.1.1.1,8.8.8.8
```

### Port

Match on ports:

```bash
# Deny traffic to port 22
drop destination port 22

# Allow traffic from ports 1024-65535
log source port 1024-65535

# Match multiple ports
drop destination port 22,23,3389

# Match traffic on any port (source or destination)
deny on port 53
```

### Address with Port

Combine address and port matching:

```bash
# Deny traffic to specific IP on specific port
drop destination address 10.19.80.100 on port 53
```

## Operators

### AND

Both conditions must be true:

```bash
log type NEW and protocol TCP
```

### OR

At least one condition must be true:

```bash
log type NEW or type UPDATE
```

### NOT

Negate a condition:

```bash
log not type DESTROY
```

### Parentheses

Group expressions:

```bash
log (type NEW and protocol TCP) or (type UPDATE and protocol UDP)
```

## Operator Precedence

1. NOT (highest)
2. AND
3. OR (lowest)

Example: `log type NEW or type UPDATE and protocol TCP` parses as `log type NEW or (type UPDATE and protocol TCP)`

## Evaluation Semantics

### First-Match Wins

Rules are evaluated in the order they are specified. The first rule whose predicate matches determines the action (allow or deny).

```bash
# First rule matches TCP traffic - allows it
# Second rule never evaluated for TCP to 8.8.8.8
conntrackd run \
  --filter "log protocol TCP" \
  --filter "drop destination address 8.8.8.8"
```

### Allow-by-Default

If no rule matches an event, it is **logged** (allowed by default):

```bash
# Don't log events to 8.8.8.8
# All other events ARE logged
conntrackd run --filter "drop destination address 8.8.8.8"
```

To change this behavior and log **only** specific events, use `drop ANY` as the final rule:

```bash
# Log ONLY events to 8.8.8.8
# All other events are NOT logged
conntrackd run \
  --filter "log destination address 8.8.8.8" \
  --filter "drop ANY"
```

## Examples

### Example 1: Don't Log Specific Destination

Don't log events to a specific IP address, but log all TCP traffic to public networks:

```bash
conntrackd run \
  --filter "drop destination address 8.8.8.8" \
  --filter "log protocol TCP and destination network PUBLIC"
```

**Evaluation:**
- Traffic to 8.8.8.8: Matches first rule → **NOT LOGGED**
- TCP to public IP (not 8.8.8.8): Matches second rule → **LOGGED**
- UDP to private network: No match → **LOGGED** (default)

### Example 2: Don't Log DNS to Specific Server

Don't log DNS traffic to a specific IP, log all other TCP/UDP:

```bash
conntrackd run \
  --filter "drop destination address 10.19.80.100 on port 53" \
  --filter "log protocol TCP,UDP"
```

**Evaluation:**
- DNS to 10.19.80.100: Matches first rule → **NOT LOGGED**
- TCP/UDP to other destinations: Matches second rule → **LOGGED**
- Other protocols: No match → **LOGGED** (default)

### Example 3: Log Only Specific Traffic

Log only NEW TCP connections (don't log anything else):

```bash
conntrackd run \
  --filter "log type NEW and protocol TCP" \
  --filter "drop ANY"
```

**Evaluation:**
- NEW TCP: Matches first rule → **LOGGED**
- NEW UDP: Matches second rule → **NOT LOGGED**
- UPDATE/DESTROY: Matches second rule → **NOT LOGGED**

**Note:** Without `drop ANY`, all non-matching events would still be logged.

### Example 4: Complex Filtering

Don't log outbound traffic to private networks on specific ports:

```bash
conntrackd run \
  --filter "drop destination network PRIVATE and destination port 22,23,3389" \
  --filter "log source network PRIVATE"
```

**Evaluation:**
- Private network destination on port 22: Matches first rule → **NOT LOGGED**
- Private network source: Matches second rule → **LOGGED**
- Other traffic: No match → **LOGGED** (default)

## Best Practices

1. **Order Matters**: Place more specific rules before general rules
2. **Use `drop ANY` for Exclusive Logging**: When you want to log ONLY specific events, end with `drop ANY`
3. **Use AND for Precision**: Combine multiple conditions to create precise filters
4. **Test Incrementally**: Start with simple rules and add complexity
5. **Document Complex Rules**: Add comments in your deployment scripts
6. **Use Parentheses**: Make precedence explicit in complex expressions

## Case Insensitivity

Keywords and identifiers are case-insensitive:

```bash
# These are all equivalent
log type NEW
ALLOW TYPE NEW
Allow Type New
```

## Abbreviations

Supported abbreviations:
- `src` = `source`
- `dst` = `dest` = `destination`

```bash
# These are equivalent
deny src network PRIVATE
drop source network PRIVATE
```
