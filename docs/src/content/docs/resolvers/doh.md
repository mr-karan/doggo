---
title: DNS over HTTPS (DoH)
description: Secure your DNS queries using Doggo's DNS over HTTPS feature
---

Doggo supports DNS over HTTPS (DoH), which encrypts DNS queries and responses, enhancing privacy and security by preventing eavesdropping and manipulation of DNS traffic.

### Using DoH with Doggo

To use DoH, specify a DoH server URL prefixed with `@https://`:

```bash
doggo mrkaran.dev @https://cloudflare-dns.com/dns-query
```

### Popular DoH Providers

Doggo works with various DoH providers. Here are some popular options:

1. Cloudflare: `@https://cloudflare-dns.com/dns-query`
2. Google: `@https://dns.google/dns-query`
3. Quad9: `@https://dns.quad9.net/dns-query`

### Benefits of Using DoH

- Encrypts DNS traffic, improving privacy
- Helps bypass DNS-based content filters
- Can improve DNS security by preventing DNS spoofing attacks

### Considerations When Using DoH

- May introduce slight latency compared to classic DNS
- Some network administrators may not approve of DoH use, as it bypasses local DNS controls
