---
title: IPv4 and IPv6 Support
description: Learn how Doggo handles both IPv4 and IPv6 DNS queries
---

Doggo provides support for both IPv4 and IPv6, allowing you to perform DNS queries over either protocol.

### Default Behavior

By default, Doggo will query for A (IPv4) records only. This means it will return only IPv4 addresses when querying a domain without specifying a record type.

```bash
$ doggo mrkaran.dev
NAME            TYPE    CLASS   TTL     ADDRESS         NAMESERVER
mrkaran.dev.    A       IN      300s    104.21.7.168    127.0.0.53:53
mrkaran.dev.    A       IN      300s    172.67.187.239  127.0.0.53:53
```

### Querying for IPv6 (AAAA) Records

To query for IPv6 addresses, you need to explicitly request AAAA records:

```bash
$ doggo AAAA mrkaran.dev
NAME            TYPE    CLASS   TTL     ADDRESS                         NAMESERVER
mrkaran.dev.    AAAA    IN      300s    2606:4700:3030::ac43:bbef       127.0.0.53:53
mrkaran.dev.    AAAA    IN      300s    2606:4700:3035::6815:7a8        127.0.0.53:53
```

### Querying for Both IPv4 and IPv6

To get both IPv4 and IPv6 addresses, you can specify both A and AAAA record types:

```bash
$ doggo A AAAA mrkaran.dev
NAME            TYPE    CLASS   TTL     ADDRESS                         NAMESERVER
mrkaran.dev.    A       IN      204s    104.21.7.168                    127.0.0.53:53
mrkaran.dev.    A       IN      204s    172.67.187.239                  127.0.0.53:53
mrkaran.dev.    AAAA    IN      284s    2606:4700:3035::6815:7a8        127.0.0.53:53
mrkaran.dev.    AAAA    IN      284s    2606:4700:3030::ac43:bbef       127.0.0.53:53
```

### Forcing IPv4 or IPv6

You can force Doggo to use only IPv4 or IPv6 with the `-4` and `-6` flags respectively:

#### IPv4 Only (Default behavior)

```bash
$ doggo -4 mrkaran.dev
```

#### IPv6 Only

```bash
$ doggo -6 mrkaran.dev
```
