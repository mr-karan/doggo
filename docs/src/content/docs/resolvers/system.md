---
title: System Resolver
description: Learn how Doggo interacts with system resolver settings and how to configure resolver behavior
---

Doggo interacts with your system's DNS resolver configuration and provides options to customize this behavior. This page explains how Doggo handles `ndots`, `search` domains, and resolver strategies.

## Reading system resolver configuration

By default, Doggo reads nameservers, `ndots`, and search domains from your OS:

- **Linux / BSD:** `/etc/resolv.conf`
- **macOS:** `scutil --dns` (falls back to `/etc/resolv.conf` if scutil fails)
- **Windows:** the system DNS server list

### macOS split-DNS / Supplemental resolvers

macOS can advertise several resolver classes via `scutil --dns`:

| Source | Used by Doggo? |
| --- | --- |
| General-purpose resolvers (no `domain`, not Supplemental) | Yes — default system nameserver list |
| Supplemental / domain-specific resolvers (`domain : example.corp`) | Yes — **only when the query name matches that domain** |
| `DNS configuration (for scoped queries)` (Scoped / per-interface) | No — never selected for outbound queries |
| mDNS (`.local`, reverse-DNS) | No |

When a query name matches a Supplemental/domain-specific resolver, Doggo uses **that** resolver's nameservers and the `NAMESERVER` column reports them (for example `10.100.0.2:53`). With `--debug`, Doggo also logs `Using macOS domain-specific resolver from scutil` including the matched domain(s).

`--strategy=internal` still consults Supplemental resolvers with private IPs even when the query name does not match a domain entry (useful for VPN/Tailscale discovery). Explicit `@host` / `--nameserver` overrides always win over system selection.

## ndots Configuration

The `ndots` option sets the threshold for the number of dots that must appear in a name before an initial absolute query will be made.

- When using the system nameserver, Doggo reads the `ndots` value from `/etc/resolv.conf`.
- If not using the system nameserver, it defaults to 1.
- You can override this with the `--ndots` flag:

```bash
$ doggo example --ndots=2
```

This affects how Doggo handles non-fully qualified domain names.

## Search Configuration

The search configuration allows Doggo to append domain names to queries that are not fully qualified.

- By default, Doggo uses the search list defined in `resolv.conf`.
- You can disable this behavior with `--search=false`:

```bash
$ doggo example --search=false
```

- When search is enabled and a query is not fully qualified, Doggo will try appending domains from the search list.

## Resolver Strategy

The resolver strategy determines how Doggo uses nameservers, whether they come from the system resolver configuration or are specified directly with `@host` / `--nameserver`. You can specify a strategy using the `--strategy` flag:

```bash
$ doggo example.com --strategy=first
```

Available strategies:

- `all` (default): Use all nameservers.
- `first`: Use only the first nameserver in the list.
- `random`: Randomly choose one nameserver from the list for each query. This can help distribute the load across multiple nameservers.
- `internal`: Use private IP nameservers only (RFC 1918 IPv4, RFC 6598 CGNAT, or RFC 4193 IPv6 ULA). On macOS this also considers Supplemental/domain-specific resolvers so VPN/Tailscale private resolvers can be discovered.

## Command-line Options

```bash
--ndots=INT             Specify ndots parameter. Takes value from /etc/resolv.conf if using the system nameserver or 1 otherwise.
--search                Use the search list defined in resolv.conf. Defaults to true. Set --search=false to disable search list.
--strategy=STRATEGY     Specify strategy to query nameservers. Options: all, first, random, internal. Defaults to all.
--timeout=DURATION    Set the timeout for resolver responses (e.g., 5s, 400ms, 1m).
```

## Examples

1. Use system resolver with default settings:
   ```bash
   doggo example.com
   ```

2. Use system resolver but change ndots and disable search:
   ```bash
   doggo example --ndots=2 --search=false
   ```

3. Use system resolver with 'first' strategy and custom timeout:
   ```bash
   doggo example.com --strategy=first --timeout=2s
   ```

4. Override system resolver and use specific nameservers:
   ```bash
   doggo example.com @1.1.1.1 @8.8.8.8
   ```

5. Use only the first explicitly specified nameserver:
   ```bash
   doggo example.com --strategy=first @1.1.1.1 @8.8.8.8
   ```

6. On macOS with split-DNS, query an internal name and see the Supplemental resolver reported:
   ```bash
   doggo intranet.corp.example --debug
   # debug: Using macOS domain-specific resolver from scutil matched_domains=[corp.example] nameservers=[10.x.x.x]
   ```

You can find more examples at [Examples](/guide/examples) section.
