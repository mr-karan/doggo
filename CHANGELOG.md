# Changelog

All notable changes to doggo are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.4.1] - 2026-10-12

### Fixed

- Honored `-4` and `-6` when connecting to DNS over HTTPS, HTTP/3, and DNS over QUIC servers.
- Canceled pending upstream DNS lookups during HTTP/3 cleanup and family-bound DNS over QUIC queries.

### Security

- Updated the Go toolchain requirement to Go 1.27.2 and `golang.org/x/net` to v0.61.0 for security fixes.

### Changed

- Updated Go dependencies, GitHub Actions, release tools, container bases, and the Astro documentation stack to current releases.

## [1.4.0] - 2026-09-01

### Added

- Added iterative DNS delegation tracing from the root servers with `--trace`, including referral, authoritative answer, CNAME, DNAME, NXDOMAIN, and NODATA handling.
- Added table, short, and versioned JSON trace output with partial-result reporting and documented exit codes.
- Added `-b`/`--source` support for binding DNS queries to a local IPv4 or IPv6 address across supported transports.

### Changed

- Updated Globalping integration for `globalping-go` v0.3.0.
- Updated the Go toolchain requirement to Go 1.26.6.
- Added missing flags to shell completions.

### Fixed

- Rejected unsupported explicit `ANY` trace questions instead of misclassifying authoritative responses as lame delegations.
- Classified invalid trace options as configuration errors with exit code 1, including malformed source addresses and address-family mismatches.
- Made `--trace` without a query fail validation instead of exiting successfully after printing usage.
- Correctly selected and reported domain-specific macOS `scutil` resolvers.
- Preserved EDNS footer output through the configured color output stream.
- Sized UDP receive buffers for non-EDNS queries so responses larger than 512 bytes are not truncated by the client.
- Hardened source-address binding across UDP, TCP, TLS, HTTPS, and truncated-response retries.
- Removed a data race from the oversized UDP response regression test.

## [1.3.0] - 2026-08-07

### Added

- Added support for numeric DNS record types.
- Added explicit HTTP/3 transport for DNS over HTTPS.
- Added reporting for Extended DNS Errors.
- Added 32-bit ARM release artifacts.

## [1.2.1] - 2026-08-07

### Added

- Added configuration file and `DOGGO_*` environment variable support.

### Fixed

- Allowed the internal nameserver strategy to discover VPN and Tailscale resolvers on macOS.

Older release notes are available on the [GitHub releases page](https://github.com/mr-karan/doggo/releases).

[Unreleased]: https://github.com/mr-karan/doggo/compare/v1.4.1...HEAD
[1.4.1]: https://github.com/mr-karan/doggo/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/mr-karan/doggo/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/mr-karan/doggo/compare/v1.2.1...v1.3.0
[1.2.1]: https://github.com/mr-karan/doggo/releases/tag/v1.2.1
