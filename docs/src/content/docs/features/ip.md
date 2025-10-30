---
title: IPv4 and IPv6 Support
description: Learn how Doggo handles both IPv4 and IPv6 DNS queries
---

Doggo provides support for both IPv4 and IPv6, allowing you to perform DNS queries over either protocol.

### Default Behavior

By default, when no query type is specified, Doggo will query for both A (IPv4) and AAAA (IPv6) records. This provides a complete view of all available IP addresses for a domain.

```bash
$ doggo mrkaran.dev
NAME            TYPE    CLASS   TTL     ADDRESS                         NAMESERVER
mrkaran.dev.    A       IN      300s    104.21.7.168                    127.0.0.53:53
mrkaran.dev.    A       IN      300s    172.67.187.239                  127.0.0.53:53
mrkaran.dev.    AAAA    IN      300s    2606:4700:3030::ac43:bbef       127.0.0.53:53
mrkaran.dev.    AAAA    IN      300s    2606:4700:3035::6815:7a8        127.0.0.53:53
```

### Querying for Specific IP Versions

You can explicitly request only IPv4 or IPv6 addresses by specifying the record type:

#### IPv4 Only (A Records)

```bash
$ doggo A mrkaran.dev
NAME            TYPE    CLASS   TTL     ADDRESS         NAMESERVER
mrkaran.dev.    A       IN      300s    104.21.7.168    127.0.0.53:53
mrkaran.dev.    A       IN      300s    172.67.187.239  127.0.0.53:53
```

#### IPv6 Only (AAAA Records)

```bash
$ doggo AAAA mrkaran.dev
NAME            TYPE    CLASS   TTL     ADDRESS                         NAMESERVER
mrkaran.dev.    AAAA    IN      300s    2606:4700:3030::ac43:bbef       127.0.0.53:53
mrkaran.dev.    AAAA    IN      300s    2606:4700:3035::6815:7a8        127.0.0.53:53
```

### Forcing IPv4 or IPv6 Transport

You can force Doggo to use only IPv4 or IPv6 **network transport** with the `-4` and `-6` flags. These flags control both the query type and which nameservers are used:

#### IPv4 Only

The `-4` flag forces IPv4 transport and filters system nameservers to use only IPv4 addresses:

```bash
$ doggo -4 mrkaran.dev
# Queries only A records and uses only IPv4 nameservers from /etc/resolv.conf
```

#### IPv6 Only

The `-6` flag forces IPv6 transport and filters system nameservers to use only IPv6 addresses:

```bash
$ doggo -6 mrkaran.dev
# Queries only AAAA records and uses only IPv6 nameservers from /etc/resolv.conf
```

:::note[Nameserver Filtering]
When using `-4` or `-6`, Doggo automatically filters system nameservers from `/etc/resolv.conf` (or Windows DNS settings) to match the requested IP version. This prevents errors like "no suitable address found" when your system has both IPv4 and IPv6 nameservers configured.
:::

### Using IPv6 Nameservers

Doggo supports IPv6 nameservers and accepts them in multiple formats for convenience:

#### With Brackets (Standard Format)

```bash
$ doggo mrkaran.dev @[2606:4700:4700::1111]
```

#### Without Brackets (dig-compatible)

For compatibility with dig, you can omit the brackets:

```bash
$ doggo mrkaran.dev @2606:4700:4700::1111
```

#### Link-Local Addresses with Zone Identifier

You can use link-local IPv6 addresses with zone identifiers:

```bash
$ doggo mrkaran.dev @fe80::1%eth0
```

#### With Protocols

IPv6 addresses work with all supported protocols:

```bash
# DNS over TLS
$ doggo mrkaran.dev @tls://[2606:4700:4700::1111]:853

# DNS over HTTPS
$ doggo mrkaran.dev @https://dns.google/dns-query

# DNS over QUIC
$ doggo mrkaran.dev @quic://dns.adguard-dns.com:853
```
