---
title: Output Formats
description: Learn about Doggo's various output formats including colored, JSON, and short outputs
---

Doggo provides flexible output formats to suit different use cases, from human-readable colored output to machine-parsable JSON.

### Colored Output

By default, Doggo uses a colored, tabular format for easy readability.

```bash
doggo mrkaran.dev
NAME            TYPE    CLASS   TTL     ADDRESS         NAMESERVER
mrkaran.dev.    A       IN      300s    104.21.7.168    127.0.0.53:53
mrkaran.dev.    A       IN      300s    172.67.187.239  127.0.0.53:53
```

```bash
doggo mrkaran.dev --from Europe,Asia --limit 2
LOCATION                      	NAME        	TYPE	CLASS	TTL 	ADDRESS       	NAMESERVER
Vienna, AT, EU, EDIS GmbH
(AS57169)
                              	mrkaran.dev.	A   	IN   	300s	104.21.7.168  	private
                              	mrkaran.dev.	A   	IN   	300s	172.67.187.239	private
Tokyo, JP, AS, Tencent
Building, Kejizhongyi Avenue
(AS132203)
                              	mrkaran.dev.	A   	IN   	300s	104.21.7.168  	private
                              	mrkaran.dev.	A   	IN   	300s	172.67.187.239	private
```

To disable colored output, use the `--color=false` flag:

```bash
doggo mrkaran.dev --color=false
```

### JSON Output

For scripting and programmatic use, Doggo supports JSON output using the `--json` or `-J` flag:

```bash
doggo internetfreedom.in --json | jq
```

```json
{
  "responses": {
    "answers": [
      {
        "name": "internetfreedom.in.",
        "type": "A",
        "class": "IN",
        "ttl": "22s",
        "address": "104.27.158.96",
        "rtt": "37ms",
        "nameserver": "127.0.0.1:53"
      }
      // ... more entries ...
    ],
    "queries": [
      {
        "name": "internetfreedom.in.",
        "type": "A",
        "class": "IN"
      }
    ]
  }
}
```

```bash
doggo mrkaran.dev --from Europe,Asia --limit 2 --json | jq
```

```json
{
  "responses": [
    {
      "location": "Groningen, NL, EU, Google LLC (AS396982)",
      "answers": [
        {
          "name": "mrkaran.dev.",
          "type": "A",
          "class": "IN",
          "ttl": "300s",
          "address": "172.67.187.239",
          "status": "",
          "rtt": "",
          "nameserver": "private"
        }
        // ... more entries ...
      ]
    },
    {
      "location": "Jakarta, ID, AS, Zenlayer Inc (AS21859)",
      "answers": [
        {
          "name": "mrkaran.dev.",
          "type": "A",
          "class": "IN",
          "ttl": "300s",
          "address": "172.67.187.239",
          "status": "",
          "rtt": "",
          "nameserver": "private"
        }
        // ... more entries ...
      ]
    }
  ]
}
```

### Short Output

For a more concise view, use the `--short` flag to show only the response section:

```bash
doggo mrkaran.dev --short
104.21.7.168
172.67.187.239
```

```bash
doggo mrkaran.dev --from Europe,Asia --limit 2 --short
Frankfurt, DE, EU, WIBO Baltic UAB (AS59939)
104.21.7.168
172.67.187.239
Saratov, RU, AS, LLC "SMART CENTER" (AS48763)
172.67.187.239
104.21.7.168
```
