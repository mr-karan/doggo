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

By default, Doggo uses its normal HTTPS transport. To require HTTP/3 for the
DoH request, add `--http3` while keeping the standard `https://` URL:

```bash
doggo mrkaran.dev @https://cloudflare-dns.com/dns-query --http3
```

There is no separate HTTP/3 URL scheme. `--http3` is also available as
`DOGGO_HTTP3=true` or `http3 = true` in the TOML config. Enabling it without
at least one HTTPS DoH nameserver (including a DoH DNS stamp) is an error.
Doggo does not automatically fall back to HTTP/2 when an HTTP/3 request fails.
HTTP/3 connects directly over UDP and does not use `HTTP_PROXY` or
`HTTPS_PROXY` environment variables.

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
