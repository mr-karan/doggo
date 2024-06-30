---
title: Protocol Tweaks
description: Learn how to fine-tune DNS queries with Doggo's protocol tweaks
---

Doggo provides several options to tweak the DNS protocol parameters, allowing for fine-grained control over your queries.

### Query Flags

Doggo supports setting various DNS query flags:

```
--aa    Set Authoritative Answer flag
--ad    Set Authenticated Data flag
--cd    Set Checking Disabled flag
--rd    Set Recursion Desired flag (default: true)
--z     Set Z flag (reserved for future use)
--do    Set DNSSEC OK flag
```

### Examples

1. Request an authoritative answer:
   ```bash
   doggo example.com --aa
   ```

2. Request DNSSEC data:
   ```bash
   doggo example.com --do
   ```

3. Disable recursive querying:
   ```bash
   doggo example.com --rd=false
   ```
