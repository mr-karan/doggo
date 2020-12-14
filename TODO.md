# doggo - Initial Release Milestone

## Resolver
- [x] Create a DNS Resolver struct
- [x] Add methods to initialise the config, set defaults
- [x] Add a resolve method
- [x] Make it separate from Hub
- [x] Parse output into separate fields
- [ ] Test IPv6/IPv4 only options
- [x] Add DOH support
- [x] Add DOT support
- [x] Add DNS protocol on TCP mode support.
- [x] Major records supported

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

## Refactors

- [ ] Don't abuse Hub as global. Refactor methods to be independent of hub.
- [ ] Add meaningful comments where required.

## Tests
- [ ] Add tests for Command Line Usage. 

## Documentation
- [ ] README
  - [ ] Usage
  - [ ] Installation
  - [ ] Features
- [ ] Mkdocs init project
  - [ ] Custom Index (Landing Page)

## Release Checklist
- [ ] Goreleaser
  - [ ] Snap
  - [ ] Homebrew
  - [ ] ARM
  - [ ] Docker
## Future Release

- [ ] Support obscure protocal tweaks in `dig`
- [ ] `digfile`
- [ ] Support more DNS Record Types
- [ ] Error on NXDomain (Realted upstream [bug](https://github.com/miekg/dns/issues/1198))
