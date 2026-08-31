package main

import (
	"fmt"
	"os"
)

var (
	bashCompletion = `
_doggo() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    opts="--version -h --help -q --query -t --type -n --nameserver -c --class -x --reverse --any -A --authoritative --strategy --ndots --search -T --timeout -4 --ipv4 -6 --ipv6 --http3 --tls-hostname --skip-hostname-verification -b --source --aa --ad --cd --rd --z --do --nsid --cookie --padding --ede --ecs --bufsize -J --json --short --color --debug --time --gp-from --gp-limit --config"

    case "${prev}" in
        --config)
            COMPREPLY=( $(compgen -f -- ${cur}) )
            return 0
            ;;
        -t|--type)
            COMPREPLY=( $(compgen -W "A AAAA CAA CNAME HINFO HTTPS MX NS PTR SOA SRV SVCB TXT" -- ${cur}) )
            return 0
            ;;
        -c|--class)
            COMPREPLY=( $(compgen -W "IN CH HS" -- ${cur}) )
            return 0
            ;;
        -n|--nameserver)
            COMPREPLY=( $(compgen -A hostname -- ${cur}) )
            return 0
            ;;
        --strategy)
            COMPREPLY=( $(compgen -W "all random first internal" -- ${cur}) )
            return 0
            ;;
        -T|--timeout|--ndots|--ecs|--bufsize|-b|--source|--gp-from|--gp-limit)
            # Value-taking flags with no sensible static completion; suppress
            # the fallback hostname completion below.
            COMPREPLY=()
            return 0
            ;;
    esac

    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    else
        COMPREPLY=( $(compgen -A hostname -- ${cur}) )
    fi
}

complete -F _doggo doggo
`

	zshCompletion = `#compdef doggo

_doggo() {
  local -a commands
  commands=(
    'completions:Generate shell completion scripts'
  )

  _arguments -C \
    '--version[Show version of doggo]' \
    '(-h --help)'{-h,--help}'[Show list of command-line options]' \
    '(-q --query)'{-q,--query}'[Hostname to query the DNS records for]:hostname:_hosts' \
    '(-t --type)'{-t,--type}'[DNS record type by name, number, or TYPE<number>]:record type:(A AAAA CAA CNAME HINFO HTTPS MX NS PTR SOA SRV SVCB TXT)' \
    '(-n --nameserver)'{-n,--nameserver}'[Address of a specific nameserver to send queries to]:nameserver:_hosts' \
    '(-c --class)'{-c,--class}'[Network class of the DNS record being queried]:network class:(IN CH HS)' \
    '(-x --reverse)'{-x,--reverse}'[Performs a DNS Lookup for an IPv4 or IPv6 address]' \
    '--any[Query all supported DNS record types]' \
    '(-A --authoritative)'{-A,--authoritative}'[Automatically query the authoritative nameserver for the domain]' \
    '--strategy[Strategy to query nameservers]:strategy:(all random first internal)' \
    '--ndots[Number of required dots in hostname to assume FQDN]:number of dots' \
    '--search[Use the search list defined in resolv.conf]' \
    '(-T --timeout)'{-T,--timeout}'[Timeout for the resolver to return a response (e.g., 5s, 400ms, 1m)]:duration' \
    '(-4 --ipv4)'{-4,--ipv4}'[Use IPv4 only]' \
    '(-6 --ipv6)'{-6,--ipv6}'[Use IPv6 only]' \
    '--http3[Use HTTP/3 for DNS-over-HTTPS nameservers]' \
    '--tls-hostname[Hostname used for verification of certificate incase the provided DoT nameserver is an IP]:hostname:_hosts' \
    '--skip-hostname-verification[Skip TLS hostname verification in case of DoT lookups]' \
    '(-b --source)'{-b,--source}'[Bind queries to a local source IP address]:IP address' \
    '--aa[Set Authoritative Answer flag]' \
    '--ad[Set Authenticated Data flag]' \
    '--cd[Set Checking Disabled flag]' \
    '--rd[Set Recursion Desired flag]' \
    '--z[Set Z flag (reserved for future use)]' \
    '--do[Set DNSSEC OK flag]' \
    '--nsid[Request Name Server Identifier (NSID)]' \
    '--cookie[Request DNS Cookie]' \
    '--padding[Request EDNS padding for privacy]' \
    '--ede[Enable EDNS to receive Extended DNS Errors]' \
    '--ecs[EDNS Client Subnet]:subnet' \
    '--bufsize[EDNS UDP buffer size in bytes]:buffer size' \
    '(-J --json)'{-J,--json}'[Format the output as JSON]' \
    '--short[Shows only the response section in the output]' \
    '--color[Colored output]' \
    '--debug[Enable debug logging]' \
    '--time[Shows how long the response took from the server]' \
    '--gp-from[Query using Globalping API from a specific location]:location' \
    '--gp-limit[Limit the number of probes to use from Globalping]:limit' \
    '--config[Load defaults from a TOML config file]:config file:_files' \
    '*:hostname:_hosts' \
    && ret=0

  case $state in
    (commands)
      _describe -t commands 'doggo commands' commands && ret=0
      ;;
  esac

  return ret
}

_doggo
`

	fishCompletion = `
function __fish_doggo_no_subcommand
    set cmd (commandline -opc)
    if [ (count $cmd) -eq 1 ]
        return 0
    end
    return 1
end

function __fish_doggo_arg1_is_completions
    # The CLI only recognizes 'completions' as the first argument (os.Args[1]),
    # so a later word named 'completions' is just a query argument.
    set -l cmd (commandline -opc)
    test (count $cmd) -ge 2; and test $cmd[2] = completions
end

# Meta options
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'version' -d "Show version of doggo"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 'h' -l 'help'    -d "Show list of command-line options"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'config'  -d "Load defaults from a TOML config file" -r

# Query options
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 'q' -l 'query'      -d "Hostname to query the DNS records for" -x -a "(__fish_print_hostnames)"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 't' -l 'type'       -d "DNS record type by name, number, or TYPE<number>" -x -a "A AAAA CAA CNAME HINFO HTTPS MX NS PTR SOA SRV SVCB TXT"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 'n' -l 'nameserver' -d "Address of a specific nameserver to send queries to" -x -a "(__fish_print_hostnames)"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 'c' -l 'class'      -d "Network class of the DNS record being queried" -x -a "IN CH HS"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 'x' -l 'reverse'    -d "Performs a DNS Lookup for an IPv4 or IPv6 address"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'any'               -d "Query all supported DNS record types"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 'A' -l 'authoritative' -d "Automatically query the authoritative nameserver for the domain"

# Resolver options
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'strategy'  -d "Strategy to query nameservers" -x -a "all random first internal"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'ndots'     -d "Specify ndots parameter" -x
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'search'    -d "Use the search list defined in resolv.conf"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 'T' -l 'timeout'   -d "Timeout for the resolver to return a response (e.g., 5s, 400ms, 1m)" -x
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s '4' -l 'ipv4' -d "Use IPv4 only"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s '6' -l 'ipv6' -d "Use IPv6 only"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'http3' -d "Use HTTP/3 for DNS-over-HTTPS nameservers"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 'b' -l 'source' -d "Bind queries to a local source IP address" -x

# Query flags
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'aa' -d "Set Authoritative Answer flag"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'ad' -d "Set Authenticated Data flag"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'cd' -d "Set Checking Disabled flag"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'rd' -d "Set Recursion Desired flag"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'z'  -d "Set Z flag (reserved for future use)"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'do' -d "Set DNSSEC OK flag"

# EDNS0 options
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'nsid'    -d "Request Name Server Identifier (NSID)"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'cookie'  -d "Request DNS Cookie"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'padding' -d "Request EDNS padding for privacy"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'ede'     -d "Enable EDNS to receive Extended DNS Errors"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'ecs'     -d "EDNS Client Subnet" -x
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'bufsize' -d "EDNS UDP buffer size in bytes" -x

# Output options
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -s 'J' -l 'json'  -d "Format the output as JSON"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'short'        -d "Shows only the response section in the output"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'color'        -d "Colored output"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'debug'        -d "Enable debug logging"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'time'         -d "Shows how long the response took from the server"

# TLS options
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'tls-hostname'               -d "Hostname for certificate verification" -x -a "(__fish_print_hostnames)"
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'skip-hostname-verification' -d "Skip TLS hostname verification in case of DoT lookups"

# Globalping options
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'gp-from'  -d "Query using Globalping API from a specific location" -x
complete -c doggo -n 'not __fish_doggo_arg1_is_completions' -l 'gp-limit' -d "Limit the number of probes to use from Globalping" -x

# Completions command
complete -c doggo -n '__fish_doggo_no_subcommand' -a completions -d "Generate shell completion scripts"
complete -c doggo -n '__fish_doggo_arg1_is_completions' -x -f -a "bash zsh fish" -d "Shell type"
`
)

func completionsCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: doggo completions [bash|zsh|fish]")
		os.Exit(1)
	}

	shell := os.Args[2]
	switch shell {
	case "bash":
		fmt.Println(bashCompletion)
	case "zsh":
		fmt.Println(zshCompletion)
	case "fish":
		fmt.Println(fishCompletion)
	default:
		fmt.Printf("Unsupported shell: %s\n", shell)
		os.Exit(1)
	}
}
