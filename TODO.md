# doggo - v1.0 Milestone

## Resolver
- [x] Create a DNS Resolver struct
- [x]] Add methods to initialise the config, set defaults
- [x] Add a resolve method
- [x] Make it separate from Hub
- [x] Parse output into separate fields
- [ ] Test IPv6
- [x] Add DOH support
- [x] Add DOT support
- [x] Add DNS protocol on TCP mode support.
- [ ] Error on NXDomain (Realted upstream [bug](https://github.com/miekg/dns/issues/1198))

## CLI Features
- [ ] `digfile`
- [x] `ndots` support
- [x] `search list` support
- [x] JSON output
- [x] Colorized output
- [x] Table output
- [ ] Parsing options free-form
- [x] Remove urfave/cli in favour of `flag`

## CLI Grunt
- [x] Query args
- [x] Neatly package them to load args in different functions
- [x] Upper case is not mandatory for query type/classes
- [x] Output
- [x] Add client transport options

## Tests

## Documentation

## Release Checklist
- [ ] Goreleaser
  - [ ] Snap
  - [ ] Homebrew
  - [ ] ARM

