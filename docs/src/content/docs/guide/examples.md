---
title: Usage Examples
description: Practical examples showcasing the versatility and power of Doggo DNS client
---

These examples showcase how to combine different features for powerful DNS querying.

## Basic Queries

1. Simple A record lookup:

   ```bash
   doggo example.com
   ```

2. Query for a specific record type:

   ```bash
   doggo AAAA example.com
   ```

3. Query multiple record types simultaneously:

   ```bash
   doggo A AAAA MX example.com
   ```

4. Query using Globalping API from a specific location:
   ```bash
   doggo example.com --gp-from Germany
   ```

### Using Different Resolvers

4. Query using a specific DNS resolver:

   ```bash
   doggo example.com @1.1.1.1
   ```

5. Use DNS-over-HTTPS (DoH):

   ```bash
   doggo example.com @https://cloudflare-dns.com/dns-query
   ```

6. Use DNS-over-TLS (DoT):

   ```bash
   doggo example.com @tls://1.1.1.1
   ```

7. Query multiple resolvers and compare results:

   ```bash
   doggo example.com @1.1.1.1 @8.8.8.8 @9.9.9.9
   ```

8. Using Globalping API
   ```bash
   doggo example.com @1.1.1.1  --gp-from Germany
   ```

### Advanced Queries

8. Perform a reverse DNS lookup:

   ```bash
   doggo --reverse 8.8.8.8
   ```

9. Set query flags for DNSSEC validation:

   ```bash
   doggo example.com --do --cd
   ```

10. Use the short output format for concise results:

    ```bash
    doggo example.com --short
    ```

11. Show query timing information:
    ```bash
    doggo example.com --time
    ```

### Combining Flags

12. Perform a reverse lookup with short output and custom resolver:

    ```bash
    doggo --reverse 8.8.8.8 --short @1.1.1.1
    ```

13. Query for MX records using DoH with JSON output:

    ```bash
    doggo MX example.com @https://dns.google/dns-query --json
    ```

14. Use IPv6 only with a specific timeout and DNSSEC checking:
    ```bash
    doggo AAAA example.com -6 --timeout 3s --do
    ```

## Scripting and Automation

16. Use JSON output for easy parsing in scripts:

    ```bash
    doggo example.com --json | jq '.responses[0].answers[].address'
    ```

17. Batch query multiple domains from a file:

    ```bash
    cat domains.txt | xargs -I {} doggo {} --short
    ```

18. Find all nameservers for a domain and its parent domains:

    ```bash
    doggo NS example.com example.com. com. . --short
    ```

19. Extract all MX records and their priorities:

    ```bash
    doggo MX gmail.com --json | jq -r '.responses[0].answers[] | "\(.address) \(.preference)"'
    ```

20. Count the number of IPv6 addresses for a domain:
    ```bash
    doggo AAAA example.com --json | jq '.responses[0].answers | length'
    ```

## EDNS Options

21. Request Name Server Identifier (NSID) to see which server responded:

    ```bash
    doggo example.com --nsid @1.1.1.1
    ```

22. Use EDNS Client Subnet (ECS) for geo-aware CDN responses:

    ```bash
    doggo example.com --ecs 8.8.8.0/24 @8.8.8.8
    ```

    This is particularly useful for testing how CDNs route traffic based on client location.

23. Compare responses from different geographic locations using ECS:

    ```bash
    # North America subnet
    doggo example.com --ecs 8.8.8.0/24 @8.8.8.8

    # Europe subnet
    doggo example.com --ecs 1.1.1.0/24 @8.8.8.8
    ```

24. **Real-world example: Test geo-aware DNS with Netflix**

    Netflix uses geo-aware DNS to route users to regional servers. Query from different locations to see how they return different IP addresses:

    ```bash
    # From USA (using subnet 8.8.8.0/24)
    doggo netflix.com --ecs 8.8.8.0/24 @8.8.8.8
    # Returns: 3.225.92.8 (AWS US-East servers)

    # From India (using subnet 49.207.0.0/24)
    doggo netflix.com --ecs 49.207.0.0/24 @8.8.8.8
    # Returns: 54.246.79.9 (AWS EU-Ireland servers)

    # From Germany (using subnet 5.9.0.0/24)
    doggo netflix.com --ecs 5.9.0.0/24 @8.8.8.8
    # Returns: 54.74.73.31 (AWS EU-Ireland servers)
    ```

    Notice how Netflix returns **completely different IP addresses** based on your location. This ensures you connect to the closest data center for faster streaming.

    **Understanding CDN behavior:**
    - Some services like Netflix use **geo-aware DNS** (different IPs per region)
    - Others like Cloudflare use **Anycast** (same IPs globally, routing happens at network level)
    - ECS lets you test this without actually traveling!

25. Use DNS Cookie for enhanced security:

    ```bash
    doggo example.com --cookie @1.1.1.1
    ```

26. Combine EDNS options for privacy and debugging:

    ```bash
    doggo example.com --nsid --cookie --padding @1.1.1.1
    ```

    The `--padding` flag helps protect against traffic analysis attacks.

## Troubleshooting and Debugging

27. Enable debug logging for verbose output:

    ```bash
    doggo example.com --debug
    ```

28. Request Extended DNS Errors (EDE) for detailed failure information:

    ```bash
    doggo nonexistent.example --ede @1.1.1.1
    ```

29. Test DNSSEC validation:

    ```bash
    doggo rsasecured.net --do @8.8.8.8
    ```

    This example uses a domain known to be DNSSEC-signed. The `--do` flag sets the DNSSEC OK bit.

    Note: DNSSEC validation can be complex and depends on various factors:

    - The domain must be properly DNSSEC-signed
    - The resolver must support DNSSEC
    - The resolver must be configured to perform DNSSEC validation

    If you don't see DNSSEC-related information in the output, try using a resolver known to support DNSSEC, like 8.8.8.8 (Google) or 9.9.9.9 (Quad9).

30. Check for DNSSEC records (DNSKEY, DS, RRSIG):

    ```bash
    doggo DNSKEY example.com @8.8.8.8
    doggo DS example.com @8.8.8.8
    doggo RRSIG example.com @8.8.8.8
    ```

31. Verify DNSSEC chain of trust:
    ```bash
    doggo example.com --type=A --do --cd=false @8.8.8.8
    ```

## Internationalized Domain Names (IDN)

32. Query Unicode domain names directly:

    ```bash
    doggo münchen.de
    ```

    Doggo automatically converts Unicode to punycode for DNS queries and displays results in Unicode for readability.

33. Query international domains in different scripts:

    ```bash
    # German (Umlauts)
    doggo die-förderer.net

    # Arabic
    doggo مصر.eg

    # Chinese
    doggo 中国.cn

    # Japanese
    doggo 日本.jp
    ```

34. Use IDN domains with different resolvers:

    ```bash
    doggo münchen.de @https://dns.google/dns-query
    ```

## DNS Additional Section (Glue Records)

35. Query TLD nameservers to see glue records:

    ```bash
    doggo NS com. @a.gtld-servers.net --rd=false
    ```

    The Additional section shows IPv4 and IPv6 addresses for nameservers, preventing circular dependencies.

36. Investigate DNS delegation for a domain:

    ```bash
    doggo NS example.org @a.gtld-servers.net --rd=false
    ```

    This reveals how the domain is delegated at the TLD level, including glue records for in-zone nameservers.

37. Query root servers for TLD nameserver information:

    ```bash
    doggo NS org. @a.root-servers.net --rd=false
    ```

38. Debug nameserver configuration:

    ```bash
    doggo NS yourdomain.com @ns1.yourdomain.com --rd=false
    ```

    Verify that nameserver IPs are correctly configured in glue records.

39. See MX target addresses in Additional section:

    ```bash
    doggo MX gmail.com
    ```

    The Additional section may include A/AAAA records for mail servers, optimizing resolution.
