---
title: DNS over TLS (DoT)
description: Secure your DNS queries using Doggo's DNS over TLS feature
---

Doggo supports DNS over TLS (DoT), which provides encryption for DNS queries, enhancing privacy and security by protecting DNS traffic from interception and tampering.

### Using DoT with Doggo

To use DoT, specify a DoT server address prefixed with `@tls://`:

```bash
doggo example.com @tls://1.1.1.1
```

### Popular DoT Providers

Doggo works with various DoT providers. Here are some popular options:

1. Cloudflare: `@tls://1.1.1.1`
2. Google: `@tls://8.8.8.8`
3. Quad9: `@tls://9.9.9.9`

### Benefits of Using DoT

- Encrypts DNS traffic, improving privacy
- Helps prevent DNS spoofing and man-in-the-middle attacks
- Compatible with most network configurations that allow outbound connections on port 853

### Considerations When Using DoT

- May introduce slight latency compared to classic DNS
- Requires trust in the DoT provider, as they can see your DNS queries

### Advanced DoT Usage

For DNS over TLS (DoT), Doggo provides additional options:

```
--tls-hostname=HOSTNAME       Provide a hostname for certificate verification if the DoT nameserver is an IP.
--skip-hostname-verification  Skip TLS Hostname Verification for DoT Lookups.
```

#### Specify a custom TLS hostname:
   ```bash
   doggo example.com @tls://1.1.1.1 --tls-hostname=cloudflare-dns.com
   ```

#### Skip hostname verification (use with caution):
   ```bash
   doggo example.com @tls://1.1.1.1 --skip-hostname-verification
   ```
