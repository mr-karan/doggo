# doggo - Initial Release Milestone

## Resolver
- [x] Create a DNS Resolver struct
- [x] Add methods to initialise the config, set defaults
- [x] Add a resolve method
- [x] Make it separate from Hub
- [x] Parse output into separate fields
- [x] Test IPv6/IPv4 only options
- [x] Add DOH support
- [x] Add DOT support
- [x] Add DNS protocol on TCP mode support.
  - [x] Change lookup method.
- [x] Major records supported
- [x] Support multiple resolvers
  - [x] Take multiple transport options and initialise resolvers accordingly. 
- [x] Add timeout support
- [x] Support SOA/NXDOMAIN

## CLI Features
- [x] `ndots` support
- [x] `search list` support
- [x] JSON output
- [x] Colorized output
- [x] Table output
- [x] Parsing options free-form

## CLI Grunt
- [x] Query args
- [x] Neatly package them to load args in different functions
- [x] Upper case is not mandatory for query type/classes
- [x] Output
- [x] Custom Help Text
  - [x] Add examples
  - [x] Colorize
  - [x] Add different commands
- [x] Add client transport options
- [x] Fix an issue while loading free form args, where the same records are being added twice
- [x] Remove urfave/cli in favour of `pflag + koanf`
- [x] Flags - Remove unneeded ones

## Documentation
- [x] README
  - [x] Usage
  - [x] Installation
  - [x] Features


## Release Checklist
- [x] Goreleaser
  - [x] Snap
  - [x] Docker
---
# Future Release

- [ ] Support obscure protocol tweaks in `dig`
- [ ] Read from file with `-f`
- [ ] Support more DNS Record Types
- [ ] Shell completions
  - [ ] bash
  - [ ] zsh
  - [ ] fish
- [ ] Support non RFC Compliant DOH Google response (_ugh_)
- [ ] Add tests for Resolvers.
- [ ] Add tests for CLI Output. 
- [ ] Mkdocs init project
  - [ ] Custom Index (Landing Page)
- [ ] Homebrew - Goreleaser
- [x] Separate Authority/Answer in JSON output.
- [x] Error on NXDomain (Related upstream [bug](https://github.com/miekg/dns/issues/1198))
