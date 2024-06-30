---
title: Installation
description: Learn how to install Doggo, a modern command-line DNS client for humans
---

Doggo can be installed using various methods. Choose the one that best suits your needs and system configuration.


### Binary Installation

You can download pre-compiled binaries for Linux, macOS, and Windows from the [Releases](https://github.com/mr-karan/doggo/releases) section of the GitHub repository.

To install the latest `linux-amd64` binary:

```shell
$ cd "$(mktemp -d)"
$ curl -sL "https://github.com/mr-karan/doggo/releases/download/v0.3.7/doggo_0.3.7_linux_amd64.tar.gz" | tar xz
$ mv doggo /usr/local/bin
# doggo should be available now in your $PATH
$ doggo
```

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
go install github.com/mr-karan/doggo/cmd/doggo@latest
```

The binary will be available at `$GOPATH/bin/doggo`.

After installation, you can verify the installation by running `doggo` in your terminal. For usage examples and command-line arguments, refer to the [Usage](/usage) section of the documentation.
