---
title: DNS over QUIC (DoQ)
description: Leverage the speed and security of QUIC protocol for DNS queries with Doggo
---

Doggo supports DNS over QUIC (DoQ), a relatively new protocol that enhances security through data encryption and improves internet performance by utilizing QUIC. QUIC, or Quick UDP Internet Connections, is a network protocol developed by Google. It reduces latency and speeds up data transmission compared to the traditional TCP protocol.

### Using DoQ with Doggo

To use DoQ, specify a DoQ server URL prefixed with `@quic://`:

```bash
doggo mrkaran.dev @quic://dns.adguard.com
```

### Available DoQ Providers

As DoQ is a relatively new protocol, fewer providers currently support it compared to DoH or DoT. Here are some known DoQ providers:

1. AdGuard: `@quic://dns.adguard.com`
2. Cloudflare: `@quic://cloudflare-dns.com`

### Benefits of Using DoQ

- Reduces connection establishment time compared to TCP-based protocols
- Improves performance on unreliable networks
