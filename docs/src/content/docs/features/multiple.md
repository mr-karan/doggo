---
title: Multiple Resolvers
description: Learn how to use multiple DNS resolvers simultaneously with Doggo
---

Doggo supports querying multiple DNS resolvers simultaneously, allowing you to compare responses or use different resolvers for different purposes.

### Using Multiple Resolvers

To use multiple resolvers, simply specify them in your command:

```bash
$ doggo mrkaran.dev @1.1.1.1 @8.8.8.8 @9.9.9.9
```

This will query the domain `mrkaran.dev` using Cloudflare (1.1.1.1), Google (8.8.8.8), and Quad9 (9.9.9.9) DNS servers.

### Mixing Resolver Types

You can mix different types of resolvers in a single query:

```bash
doggo mrkaran.dev @1.1.1.1 @https://dns.google/dns-query @tls://9.9.9.9
```

This command uses a standard DNS resolver (1.1.1.1), a DoH resolver (Google), and a DoT resolver (Quad9).
