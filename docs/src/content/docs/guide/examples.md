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

## Troubleshooting and Debugging

21. Enable debug logging for verbose output:
    ```bash
    doggo example.com --debug
    ```

22. Compare responses with and without EDNS Client Subnet:
    ```bash
    doggo example.com @8.8.8.8
    doggo example.com @8.8.8.8 --z
    ```

23. Test DNSSEC validation:
    ```bash
    doggo rsasecured.net --do @8.8.8.8
    ```
    This example uses a domain known to be DNSSEC-signed. The `--do` flag sets the DNSSEC OK bit.

    Note: DNSSEC validation can be complex and depends on various factors:
    - The domain must be properly DNSSEC-signed
    - The resolver must support DNSSEC
    - The resolver must be configured to perform DNSSEC validation

    If you don't see DNSSEC-related information in the output, try using a resolver known to support DNSSEC, like 8.8.8.8 (Google) or 9.9.9.9 (Quad9).

24. Compare responses with and without EDNS Client Subnet:
    ```bash
    doggo example.com @8.8.8.8
    doggo example.com @8.8.8.8 --z
    ```

25. Check for DNSSEC records (DNSKEY, DS, RRSIG):
    ```bash
    doggo DNSKEY example.com @8.8.8.8
    doggo DS example.com @8.8.8.8
    doggo RRSIG example.com @8.8.8.8
    ```

26. Verify DNSSEC chain of trust:
    ```bash
    doggo example.com --type=A --do --cd=false @8.8.8.8
    ```
