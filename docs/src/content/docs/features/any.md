---
title: Common record types
description: Learn how to query multiple DNS record types simultaneously
---

The `--any` flag in Doggo allows you to query all supported DNS record types for a given domain in a single command. This can be particularly useful when you want to get a comprehensive view of a domain's DNS configuration without running multiple queries.

## Syntax

```bash
doggo [domain] --any
```

## Supported Record Types

When you use the `--any` flag, Doggo will query for the following common DNS record types:

- A (IPv4 address)
- AAAA (IPv6 address)
- CNAME (Canonical name)
- MX (Mail exchange)
- NS (Name server)
- PTR (Pointer)
- SOA (Start of authority)
- SRV (Service)
- TXT (Text)
- CAA (Certification Authority Authorization)

## Example Usage

```bash
doggo mrkaran.dev --any
```

## Example Output

```bash
$ doggo example.com --any
NAME            TYPE    CLASS   TTL     ADDRESS                         NAMESERVER
example.com.    A       IN      3401s   93.184.215.14                           127.0.0.53:53
example.com.    AAAA    IN      3600s   2606:2800:21f:cb07:6820:80da:af6b:8b2c  127.0.0.53:53
example.com.    MX      IN      77316s  0 .                                     127.0.0.53:53
example.com.    NS      IN      86400s  a.iana-servers.net.                     127.0.0.53:53
example.com.    NS      IN      86400s  b.iana-servers.net.                     127.0.0.53:53
example.com.    SOA     IN      3600s   ns.icann.org.                           127.0.0.53:53
                                        noc.dns.icann.org. 2024041841
                                        7200 3600 1209600 3600
example.com.    TXT     IN      86400s  "v=spf1 -all"                           127.0.0.53:53
example.com.    TXT     IN      86400s  "wgyf8z8cgvm2qmxpnbnldrcltvk4xqfn"      127.0.0.53:53
```

## Considerations

- The `--any` query may take longer to complete compared to querying a single record type, as it's fetching multiple record types.
