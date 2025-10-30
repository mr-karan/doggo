---
title: Additional Section (Glue Records)
description: Understanding DNS Additional section and glue records in Doggo
---

Doggo displays the DNS Additional section, which contains supplementary information that can help resolve queries more efficiently. The most common use case is **glue records** for nameserver queries.

### What are Glue Records?

Glue records are A or AAAA records included in the Additional section of a DNS response. They provide the IP addresses of nameservers, preventing circular dependencies when resolving nameserver hostnames.

### Why Glue Records Matter

Consider querying for nameservers of a domain:

```bash
$ doggo NS example.com @a.iana-servers.net
```

The response tells you the nameservers are `a.iana-servers.net` and `b.iana-servers.net`, but how do you find their IP addresses? This is where glue records come in - they provide the IP addresses directly in the Additional section.

### Viewing Glue Records

To see glue records, query NS records from an authoritative server with recursion disabled:

```bash
$ doggo NS com. @a.gtld-servers.net --rd=false
```

This returns:
- **Answer section**: NS records listing the nameservers
- **Additional section**: A and AAAA records (glue records) for those nameservers

### Example: Querying TLD Nameservers

```bash
$ doggo NS com. @a.gtld-servers.net --rd=false

# Answer Section
NAME    TYPE  CLASS  TTL       ADDRESS
com.    NS    IN     172800s   a.gtld-servers.net.
com.    NS    IN     172800s   b.gtld-servers.net.
...

# Additional Section (Glue Records)
NAME                    TYPE   CLASS  TTL       ADDRESS
a.gtld-servers.net.     A      IN     172800s   192.5.6.30
b.gtld-servers.net.     A      IN     172800s   192.33.14.30
a.gtld-servers.net.     AAAA   IN     172800s   2001:503:a83e::2:30
b.gtld-servers.net.     AAAA   IN     172800s   2001:503:231d::2:30
```

###  Understanding the Flags

The `--rd=false` flag disables recursion, which is important when querying authoritative servers:

- **With recursion (`--rd` default)**: The resolver follows the chain of nameservers
- **Without recursion (`--rd=false`)**: Get the direct authoritative answer with glue records

### Common Use Cases

#### 1. Investigating DNS Delegation

Check how a domain is delegated at the TLD level:

```bash
$ doggo NS example.org @a.gtld-servers.net --rd=false
```

#### 2. Debugging Nameserver Configuration

Verify that nameserver IPs are correctly configured:

```bash
$ doggo NS yourdomain.com @ns1.yourdomain.com --rd=false
```

#### 3. Finding Root Server IPs

Query root servers for TLD nameserver information:

```bash
$ doggo NS org. @a.root-servers.net --rd=false
```

### Additional Section Content

The Additional section may contain:

1. **Glue Records**: A/AAAA records for NS entries
2. **EDNS Information**: OPT pseudo-records with EDNS data
3. **SRV Target Records**: A/AAAA records for SRV targets
4. **MX Target Records**: A/AAAA records for mail servers

:::note[EDNS vs Glue Records]
The Additional section serves dual purposes. While glue records help with nameserver resolution, EDNS (OPT records) provides metadata about DNS capabilities. Doggo separates these into distinct "Additional" and "EDNS" tabs in the output for clarity.
:::

### Practical Tips

**When glue records are needed:**
- Nameserver hostname is within the zone it serves
- Example: `ns1.example.com` serving `example.com`

**When glue records aren't needed:**
- Nameserver is outside the zone
- Example: `ns1.cloudflare.com` serving `example.com`

**Best practice for zone operators:**
Always provide glue records for in-zone nameservers to prevent resolution delays and circular dependencies.

### Viewing in Web Interface

The Doggo web interface displays the Additional section in a dedicated tab, making it easy to review glue records alongside your DNS query results.

### Example with MX Records

Additional section can also include A/AAAA records for MX targets:

```bash
$ doggo MX gmail.com

# Answer Section
gmail.com.    MX    IN    3600s    5 gmail-smtp-in.l.google.com.
...

# Additional Section (IP addresses for mail servers)
gmail-smtp-in.l.google.com.    A    IN    300s    172.253.115.27
```

This optimization helps mail clients immediately connect to mail servers without additional DNS lookups.
