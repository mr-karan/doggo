---
title: Installation
description: Learn how to install Doggo, a modern command-line DNS client for humans
---

Doggo can be installed using various methods. Choose the one that best suits your needs and system configuration.

### Easy Install (Recommended)

The easiest way to install Doggo is by using the installation script:

```shell
curl -sS https://raw.githubusercontent.com/mr-karan/doggo/main/install.sh | sh
```

This script will automatically download and install the latest version of Doggo for your system.

### Binary Installation

You can download pre-compiled binaries for various operating systems and architectures from the [Releases](https://github.com/mr-karan/doggo/releases) section of the GitHub repository.

### Docker

Doggo is available as a Docker image hosted on GitHub Container Registry (ghcr.io). It supports both x86 and ARM architectures.

To pull the latest image:

```shell
docker pull ghcr.io/mr-karan/doggo:latest
```

To run Doggo using Docker:

```shell
docker run ghcr.io/mr-karan/doggo:latest mrkaran.dev @1.1.1.1 MX
```

### Package Managers

#### Homebrew (macOS and Linux)

Install via [Homebrew](https://brew.sh/):

```bash
brew install doggo
```

#### Arch Linux

Install using an AUR helper like `yay`:

```bash
yay -S doggo-bin
```

#### Scoop (Windows)

Install via [Scoop](https://scoop.sh/):

```bash
scoop install doggo
```

### From Source

To install Doggo from source, you need to have Go installed on your system.

```bash
go install github.com/mr-karan/doggo/cmd@latest
```

The binary will be available at `$GOPATH/bin/doggo`.

After installation, you can verify the installation by running `doggo` in your terminal. For usage examples and command-line arguments, refer to the [Usage](/usage) section of the documentation.
