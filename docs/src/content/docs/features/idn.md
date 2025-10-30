---
title: Internationalized Domain Names (IDN)
description: Learn how Doggo handles Unicode domain names and punycode conversion
---

Doggo fully supports Internationalized Domain Names (IDN), allowing you to query domains with Unicode characters in their native script.

### What are IDNs?

Internationalized Domain Names allow domain names to contain characters from non-ASCII scripts like Arabic, Chinese, Cyrillic, Hebrew, and many others. Since DNS itself only supports ASCII characters, these Unicode domain names are converted to ASCII using Punycode encoding.

### Automatic Punycode Conversion

Doggo automatically handles punycode conversion for you:

- **Input**: You can enter domain names in Unicode
- **Processing**: Doggo converts them to punycode for DNS queries
- **Output**: Results are displayed in Unicode for readability

```bash
$ doggo münchen.de
NAME            TYPE    CLASS   TTL     ADDRESS         NAMESERVER
münchen.de.     A       IN      300s    185.52.1.77     127.0.0.53:53
münchen.de.     A       IN      300s    185.52.3.77     127.0.0.53:53
```

### Examples with International Domains

#### German Domains (Umlauts)

```bash
$ doggo die-förderer.net
NAME                TYPE    CLASS   TTL     ADDRESS         NAMESERVER
die-förderer.net.   A       IN      300s    134.119.225.93  127.0.0.53:53
```

#### Arabic Domains

```bash
$ doggo مصر.eg
NAME        TYPE    CLASS   TTL     ADDRESS         NAMESERVER
مصر.eg.     A       IN      300s    156.160.2.8     127.0.0.53:53
```

#### Chinese Domains

```bash
$ doggo 中国.cn
NAME        TYPE    CLASS   TTL     ADDRESS         NAMESERVER
中国.cn.    A       IN      300s    203.119.25.1    127.0.0.53:53
```

#### Japanese Domains

```bash
$ doggo 日本.jp
NAME        TYPE    CLASS   TTL     ADDRESS         NAMESERVER
日本.jp.    A       IN      300s    210.155.141.200 127.0.0.53:53
```

### Punycode in Output

While Doggo displays Unicode domain names in the output, the underlying DNS protocol uses punycode. For reference, here's how some Unicode domains map to punycode:

| Unicode Domain | Punycode Equivalent |
|---------------|-------------------|
| münchen.de | xn--mnchen-3ya.de |
| die-förderer.net | xn--die-frderer-feb.net |
| مصر.eg | xn--wgbh1c.eg |
| 中国.cn | xn--fiqs8s.cn |
| 日本.jp | xn--wgv71a.jp |

### Technical Details

Doggo uses the IDNA2008 standard for domain name internationalization, which provides:

- Support for the latest Unicode characters
- Proper handling of right-to-left scripts (Arabic, Hebrew)
- Validation of domain name characters
- Bidirectional text support

:::tip[Emoji Domains]
Some domain registries support emoji in domain names! While these are fun, they may have limited practical use and not all registries support them.
:::

### Benefits

- **Native Script**: Query domains in their native language without needing to know punycode
- **Better Readability**: See results in Unicode instead of confusing punycode strings
- **Global Support**: Works with all Unicode-enabled top-level domains
- **Transparent**: Conversion happens automatically - you don't need to think about it
