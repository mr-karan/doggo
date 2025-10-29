---
title: Protocol Tweaks
description: Learn how to fine-tune DNS queries with Doggo's protocol tweaks
---

Doggo provides several options to tweak the DNS protocol parameters, allowing for fine-grained control over your queries.

## Query Flags

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

## EDNS Options

EDNS (Extension Mechanisms for DNS) provides additional capabilities beyond basic DNS queries. Doggo supports several EDNS0 options:

```
--nsid      Request Name Server Identifier
--cookie    Request DNS Cookie for enhanced security
--padding   Request EDNS padding for privacy
--ede       Request Extended DNS Errors
--ecs       EDNS Client Subnet (e.g., '192.0.2.0/24')
```

### EDNS Examples

1. **Name Server Identifier (NSID)** - Identify which server responded:
   ```bash
   doggo google.com --nsid @8.8.8.8
   ```

   Output includes:
   ```
   EDNS Information:
     NSID: gpdns-maa
   ```

2. **EDNS Client Subnet (ECS)** - Get geo-aware responses:
   ```bash
   # Query as if from USA
   doggo netflix.com --ecs 8.8.8.0/24 @8.8.8.8

   # Query as if from India
   doggo netflix.com --ecs 49.207.0.0/24 @8.8.8.8
   ```

   This reveals how CDNs route traffic based on client location. Netflix will return different IP addresses for different regions!

3. **DNS Cookie** - Enhanced security against spoofing:
   ```bash
   doggo example.com --cookie @1.1.1.1
   ```

4. **EDNS Padding** - Privacy protection against traffic analysis:
   ```bash
   doggo example.com --padding @1.1.1.1
   ```

5. **Extended DNS Errors (EDE)** - Detailed error information:
   ```bash
   doggo nonexistent.example --ede @1.1.1.1
   ```

6. **Combine multiple EDNS options**:
   ```bash
   doggo example.com --nsid --cookie --padding --do @8.8.8.8
   ```

### Understanding ECS (EDNS Client Subnet)

ECS allows DNS resolvers to include client subnet information in queries, enabling authoritative servers to provide location-aware responses. This is commonly used by:

- **CDNs** (Content Delivery Networks) to direct users to nearby servers
- **Streaming services** like Netflix to serve region-specific content
- **Cloud providers** to optimize latency

**How it works:**
1. You specify a subnet (e.g., `--ecs 8.8.8.0/24`)
2. DNS resolver includes this in the query to the authoritative server
3. Server responds with IPs optimized for that geographic region
4. Response includes the actual scope used (e.g., `Scope: 24`)

**Real-world example:**
```bash
# From USA - returns AWS US-East servers
doggo netflix.com --ecs 8.8.8.0/24 @8.8.8.8

# From India - returns AWS EU-Ireland servers
doggo netflix.com --ecs 49.207.0.0/24 @8.8.8.8

# Different IPs returned based on location!
```

This lets you test geo-routing without traveling to different countries!
