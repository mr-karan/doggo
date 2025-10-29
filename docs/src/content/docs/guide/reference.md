---
title: CLI Reference Guide
description: Comprehensive guide to all command-line options and flags for Doggo DNS client
---

This guide provides a comprehensive list of all command-line options and flags available in Doggo.

## Basic Syntax

```
doggo [--] [query options] [arguments...]
```

## Query Options

| Option                  | Description                                                                  |
| ----------------------- | ---------------------------------------------------------------------------- |
| `-q, --query=HOSTNAME`  | Hostname to query the DNS records for (e.g., example.com)                    |
| `-t, --type=TYPE`       | Type of the DNS Record (A, MX, NS, etc.)                                     |
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
| `--ede`       | Request Extended DNS Errors for detailed error information when queries fail                                 |
| `--ecs=SUBNET`| EDNS Client Subnet - sends client subnet information for geo-aware responses (e.g., `192.0.2.0/24` or `2001:db8::/32`) |

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
