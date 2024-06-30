---
title: DNSCrypt Resolver
description: Enhance your DNS privacy and security with Doggo's DNSCrypt support
---

Doggo supports DNSCrypt, a protocol that authenticates communications between a DNS client and a DNS resolver. It prevents DNS spoofing and provides confidentiality for DNS queries.

### Using DNSCrypt

To use DNSCrypt, you need to provide a DNS stamp prefixed with `@sdns://`:

```bash
doggo mrkaran.dev @sdns://AgcAAAAAAAAABzEuMC4wLjEAEmRucy5jbG91ZGZsYXJlLmNvbQovZG5zLXF1ZXJ5
```

This command initiates a DNSCrypt (or DoH) resolver using its DNS stamp.

### DNS Stamps

DNS stamps are compact, encoded strings that contain all the necessary information to connect to a DNSCrypt or DoH server. They include:

- The protocol used (DNSCrypt or DoH)
- The server's address and port
- The provider's public key
- The provider name

The stamp in the example above is for Cloudflare's DNS service.

### Benefits of Using DNSCrypt

- Authenticates the DNS resolver, preventing DNS spoofing attacks
- Encrypts DNS queries and responses, enhancing privacy
- Supports features like DNS-based blocklists and custom DNS rules

### Public DNSCrypt Resolvers

You can find a list of public DNSCrypt resolvers at [dnscrypt.info](https://dnscrypt.info/public-servers). Each resolver will have its own DNS stamp that you can use with Doggo.

### Considerations When Using DNSCrypt

- Requires trust in the DNSCrypt provider, as they can see your DNS queries.
- May introduce slight latency compared to classic DNS resolver.
- Not all DNS providers support DNSCrypt.
