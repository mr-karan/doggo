---
title: CLI Reference Guide
description: Comprehensive guide to all command-line options and flags for Doggo DNS client
---

This guide provides a comprehensive list of all command-line options and flags available in Doggo.

## Basic Syntax

```
doggo [--] [query options] [arguments...]
```

Free-form decimal arguments and `TYPE<number>` are interpreted as record
types. Use `--query=65` when querying a numeric single-label hostname.
Invalid, reserved, and non-question record types are rejected before a DNS
request is sent; the lookup API reports the same validation failures as HTTP
400 responses.

## Query Options

| Option                  | Description                                                                  |
| ----------------------- | ---------------------------------------------------------------------------- |
| `-q, --query=HOSTNAME`  | Hostname to query the DNS records for (e.g., example.com)                    |
| `-t, --type=TYPE`       | DNS record type by name, number, or RFC 3597 notation (e.g., `HTTPS`, `65`, `TYPE65`) |
| `-n, --nameserver=ADDR` | Address of a specific nameserver to send queries to (e.g., 9.9.9.9, 8.8.8.8) |
| `-c, --class=CLASS`     | Network class of the DNS record (IN, CH, HS, etc.)                           |
| `-x, --reverse`         | Performs a reverse DNS lookup for an IPv4 or IPv6 address                    |

## Resolver Options

| Option                         | Description                                                                 |
| ------------------------------ | --------------------------------------------------------------------------- |
| `--strategy=STRATEGY`          | Specify strategy to query nameservers (all, random, first)                  |
| `--ndots=INT`                  | Specify ndots parameter                                                     |
| `--search`                     | Use the search list defined in resolv.conf (default: true)                  |
| `--timeout=DURATION`           | Specify timeout for the resolver to return a response (e.g., 5s, 400ms, 1m) |
| `-4, --ipv4`                   | Use IPv4 only                                                               |
| `-6, --ipv6`                   | Use IPv6 only                                                               |
| `--http3`                      | Use HTTP/3 for HTTPS (DoH) nameservers                                      |
| `--tls-hostname=HOSTNAME`      | Provide a hostname for TLS certificate verification                         |
| `--skip-hostname-verification` | Skip TLS Hostname Verification for DoT lookups                              |

## Query Flags

| Flag   | Description                                |
| ------ | ------------------------------------------ |
| `--aa` | Set Authoritative Answer flag              |
| `--ad` | Set Authenticated Data flag                |
| `--cd` | Set Checking Disabled flag                 |
| `--rd` | Set Recursion Desired flag (default: true) |
| `--z`  | Set Z flag (reserved for future use)       |
| `--do` | Set DNSSEC OK flag                         |

## EDNS Options

EDNS (Extension Mechanisms for DNS) provides additional capabilities beyond basic DNS queries.

| Option        | Description                                                                                                |
| ------------- | ---------------------------------------------------------------------------------------------------------- |
| `--nsid`      | Request Name Server Identifier (NSID) to identify which nameserver responded                                |
| `--cookie`    | Request DNS Cookie for enhanced security and protection against spoofing and amplification attacks          |
| `--padding`   | Request EDNS padding for privacy (helps mitigate traffic analysis attacks by standardizing packet sizes)    |
| `--ede`       | Enable EDNS to receive Extended DNS Errors with detailed error information when queries fail                 |
| `--ecs=SUBNET`| EDNS Client Subnet - sends client subnet information for geo-aware responses (e.g., `192.0.2.0/24` or `2001:db8::/32`) |
| `--bufsize=BYTES` | EDNS UDP buffer size in bytes (512-65535). Setting this enables EDNS even without other EDNS options. Default is 1232 when EDNS is enabled — the [DNS Flagday 2020](https://dnsflagday.net/2020/) recommendation to avoid IP fragmentation. |

`--ede` enables EDNS by adding an OPT record; it does not send an EDE option or payload. A resolver may return EDE information for any query carrying an OPT record, and Doggo displays it regardless of which EDNS-enabling option (`--do`, `--nsid`, `--cookie`, `--padding`, `--ede`, `--ecs`, or `--bufsize`) added that record.

### EDNS Examples

1. Query with NSID to identify the responding nameserver:
   ```
   doggo example.com --nsid @1.1.1.1
   ```

2. Use EDNS Client Subnet for geo-aware CDN responses:
   ```
   doggo example.com --ecs 8.8.8.0/24 @8.8.8.8
   ```

3. Combine multiple EDNS options for privacy and debugging:
   ```
   doggo example.com --nsid --cookie --padding @1.1.1.1
   ```

## Output Options

| Option       | Description                                           |
| ------------ | ----------------------------------------------------- |
| `-J, --json` | Format the output as JSON                             |
| `--short`    | Short output format (shows only the response section) |
| `--color`    | Enable/disable colored output (default: true)         |
| `--debug`    | Enable debug logging                                  |
| `--time`     | Show query response time                              |

## Transport Options

Specify the protocol with a URL-type scheme. UDP is used if no scheme is specified.

| Scheme      | Description                     | Example                                 |
| ----------- | ------------------------------- | --------------------------------------- |
| `@udp://`   | UDP query                       | `@1.1.1.1`                              |
| `@tcp://`   | TCP query                       | `@tcp://1.1.1.1`                        |
| `@https://` | DNS over HTTPS (DoH)            | `@https://cloudflare-dns.com/dns-query` |
| `@tls://`   | DNS over TLS (DoT)              | `@tls://1.1.1.1`                        |
| `@sdns://`  | DNSCrypt or DoH using DNS stamp | `@sdns://...`                           |
| `@quic://`  | DNS over QUIC                   | `@quic://dns.adguard.com`               |

## Globalping API Options

| Option       | Description                        | Example                 |
| ------------ | ---------------------------------- | ----------------------- |
| `--gp-from`  | Specify the location to query from | `--gp-from Europe,Asia` |
| `--gp-limit` | Limit the number of probes to use  | `--gp-limit 5`          |

## General Options

| Option          | Description                                                                                        |
| --------------- | -------------------------------------------------------------------------------------------------- |
| `--config=PATH` | Load defaults from a TOML config file at PATH, bypassing the default search locations               |
| `--version`     | Show version of doggo                                                                              |

## Configuration File and Environment Variables

Defaults for any long flag can be set via a TOML config file or environment variables, so you don't have to repeat flags like `--color=false` or `--strategy=first` on every invocation.

Precedence (lowest to highest):

```
flag defaults < config file < environment variables < command line flags
```

### Config file locations

The config file is resolved in the following order (first match wins):

1. `--config=PATH` flag
2. `DOGGO_CONFIG` environment variable
3. `$XDG_CONFIG_HOME/doggo/config.toml` (or the OS user config dir: `~/Library/Application Support/doggo/config.toml` on macOS, `%AppData%\doggo\config.toml` on Windows)
4. `~/.doggo.toml`

A missing file at a default location is silently ignored. A file requested explicitly via `--config` or `DOGGO_CONFIG` that is missing or malformed is an error.

Keys are the long flag names. Example `config.toml`:

```toml
# Always pick the first nameserver instead of querying all.
strategy = "first"

# Disable colored output.
color = false

# Longer timeout for slow resolvers.
timeout = "10s"
```

Unknown keys are rejected, so a typo like `strategiy` fails immediately instead of being silently ignored.

[`config-cli-sample.toml`](https://github.com/mr-karan/doggo/blob/main/config-cli-sample.toml) in the repository documents every supported key with its default value. Copy it to one of the locations above and uncomment what you need:

```shell
mkdir -p ~/.config/doggo
curl -fsSL https://raw.githubusercontent.com/mr-karan/doggo/main/config-cli-sample.toml -o ~/.config/doggo/config.toml
```

### Environment variables

Environment variables are the `DOGGO_` prefix plus the upper-cased flag name with `-` replaced by `_`:

```shell
export DOGGO_STRATEGY=first
export DOGGO_COLOR=false
export DOGGO_TIMEOUT=10s

# HTTP/3 requires an HTTPS (DoH) nameserver.
export DOGGO_NAMESERVER=https://cloudflare-dns.com/dns-query
export DOGGO_HTTP3=true

# List flags accept comma-separated values.
export DOGGO_TYPE="A,AAAA"
```

### Web lookup API resolver security

`POST /api/lookup/` accepts the resolver URL and encrypted-transport policy in
the JSON payload. The relevant fields are `nameservers`, `http3`,
`tls_hostname`, and `skip_hostname_verification`.

The API client controls both the resolver destination and its certificate
verification policy. Setting `skip_hostname_verification` to `true` disables
TLS certificate verification for that client-selected resolver. Deployments
that expose this API should apply appropriate authentication and outbound
network controls.

## Examples

1. Query a domain using defaults:

   ```
   doggo example.com
   ```

2. Query for a CNAME record:

   ```
   doggo example.com CNAME
   ```

3. Use a custom DNS resolver:

   ```
   doggo example.com MX @9.9.9.9
   ```

4. Using named arguments:

   ```
   doggo -q example.com -t MX -n 1.1.1.1
   ```

5. Query with specific flags:

   ```
   doggo example.com --aa --ad
   ```

6. Query using Globalping API from a specific location:
   ```
   doggo example.com --gp-from Europe,Asia --gp-limit 5
   ```

For more detailed usage examples, refer to the [Examples](/guide/examples) section.
