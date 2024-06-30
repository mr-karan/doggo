---
title: Classic Resolver
description: Understanding and using Doggo's classic DNS resolver functionality
---

Doggo's classic resolver supports traditional DNS queries over UDP and TCP protocols. This is the default mode of operation and is compatible with standard DNS servers.

### Using the Classic Resolver

By default, `doggo` uses the classic resolver when no specific protocol is specified.

```bash
doggo mrkaran.dev
```

You can explicitly use UDP or TCP by prefixing the nameserver with `@udp://` or `@tcp://` respectively.

#### UDP

```bash
doggo mrkaran.dev @udp://1.1.1.1
```

#### TCP

```bash
doggo mrkaran.dev @tcp://8.8.8.8
```
