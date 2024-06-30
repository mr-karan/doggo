---
title: Reverse IP Lookups
description: Learn how to perform reverse IP lookups with Doggo
---

Doggo supports reverse IP lookups, allowing you to find the domain name associated with a given IP address. This feature is particularly useful for network diagnostics, security analysis, and understanding the ownership of IP addresses.

### Performing a Reverse IP Lookup

To perform a reverse IP lookup, use the `--reverse` flag followed by the IP address:

```bash
doggo --reverse 8.8.4.4
```

### Short Output Format

You can combine the reverse lookup with the `--short` flag to get a concise output:

```bash
doggo --reverse 8.8.4.4 --short
dns.google.
```

This command returns only the domain name associated with the IP address, without any additional information.

### Full Output Format

Without the `--short` flag, Doggo will provide more detailed information:

```bash
$ doggo --reverse 8.8.4.4
NAME                    TYPE    CLASS   TTL     ADDRESS     NAMESERVER
4.4.8.8.in-addr.arpa.   PTR     IN      21599s  dns.google. 127.0.0.53:53
```

### IPv6 Support

Reverse IP lookups also work with IPv6 addresses:

```bash
$ doggo --reverse 2001:4860:4860::8888 --short
dns.google.
```
