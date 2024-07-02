---
title: System Resolver
description: Learn how Doggo interacts with system resolver settings and how to configure resolver behavior
---

Doggo interacts with your system's DNS resolver configuration and provides options to customize this behavior. This page explains how Doggo handles `ndots`, `search` domains, and resolver strategies.

## Reading from /etc/resolv.conf

By default, Doggo reads configuration from your system's `/etc/resolv.conf` file. This includes:

- List of nameservers
- The `ndots` value
- Search domains

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

The resolver strategy determines how Doggo uses the nameservers listed in `/etc/resolv.conf`. You can specify a strategy using the `--strategy` flag:

```bash
$ doggo example.com --strategy=first
```

Available strategies:

- `all` (default): Use all nameservers listed in `/etc/resolv.conf`.
- `first`: Use only the first nameserver in the list.
- `random`: Randomly choose one nameserver from the list for each query. This can help distribute the load across multiple nameservers.

## Command-line Options

```bash
--ndots=INT             Specify ndots parameter. Takes value from /etc/resolv.conf if using the system nameserver or 1 otherwise.
--search                Use the search list defined in resolv.conf. Defaults to true. Set --search=false to disable search list.
--strategy=STRATEGY     Specify strategy to query nameservers listed in /etc/resolv.conf. Options: all, first, random. Defaults to all.
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
   Note: When specifying nameservers directly, the system resolver configuration (including strategy) is not used.

You can find more examples at [Examples](/guide/examples) section.