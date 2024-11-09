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

    opts="-v --version -h --help -q --query -t --type -n --nameserver -c --class -r --reverse --strategy --ndots --search --timeout -4 --ipv4 -6 --ipv6 --tls-hostname --skip-hostname-verification -J --json --short -1 --single --color --debug --time --gp-from --gp-limit"

    case "${prev}" in
        -t|--type)
            COMPREPLY=( $(compgen -W "A AAAA CAA CNAME HINFO MX NS PTR SOA SRV TXT" -- ${cur}) )
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
            COMPREPLY=( $(compgen -W "all random first" -- ${cur}) )
            return 0
            ;;
        --search|--color)
            COMPREPLY=( $(compgen -W "true false" -- ${cur}) )
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
    '(-v --version)'{-v,--version}'[Show version of doggo]' \
    '(-h --help)'{-h,--help}'[Show list of command-line options]' \
    '(-q --query)'{-q,--query}'[Hostname to query the DNS records for]:hostname:_hosts' \
    '(-t --type)'{-t,--type}'[Type of the DNS Record]:record type:(A AAAA CAA CNAME HINFO MX NS PTR SOA SRV TXT)' \
    '(-n --nameserver)'{-n,--nameserver}'[Address of a specific nameserver to send queries to]:nameserver:_hosts' \
    '(-c --class)'{-c,--class}'[Network class of the DNS record being queried]:network class:(IN CH HS)' \
    '(-r --reverse)'{-r,--reverse}'[Performs a DNS Lookup for an IPv4 or IPv6 address]' \
    '--strategy[Strategy to query nameserver listed in etc/resolv.conf]:strategy:(all random first)' \
    '--ndots[Number of required dots in hostname to assume FQDN]:number of dots' \
    '--search[Use the search list defined in resolv.conf]:setting:(true false)' \
    '--timeout[Timeout (in seconds) for the resolver to return a response]:seconds' \
    '(-4 --ipv4)'{-4,--ipv4}'[Use IPv4 only]' \
    '(-6 --ipv6)'{-6,--ipv6}'[Use IPv6 only]' \
    '--tls-hostname[Hostname used for verification of certificate incase the provided DoT nameserver is an IP]:hostname:_hosts' \
    '--skip-hostname-verification[Skip TLS hostname verification in case of DoT lookups]' \
    '(-J --json)'{-J,--json}'[Format the output as JSON]' \
    '--short[Shows only the response section in the output]' \
    '(-1 --single)'{-1,--single}'[Shows only the single address]' \
    '--color[Colored output]:setting:(true false)' \
    '--debug[Enable debug logging]' \
    '--time[Shows how long the response took from the server]' \
    '--gp-from[Query using Globalping API from a specific location]' \
    '--gp-limit[Limit the number of probes to use from Globalping]' \
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

# Meta options
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'version' -d "Show version of doggo"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'help'    -d "Show list of command-line options"

# Query options
complete -c doggo -n '__fish_doggo_no_subcommand' -s 'q' -l 'query'      -d "Hostname to query the DNS records for" -x -a "(__fish_print_hostnames)"
complete -c doggo -n '__fish_doggo_no_subcommand' -s 't' -l 'type'       -d "Type of the DNS Record" -x -a "A AAAA CAA CNAME HINFO MX NS PTR SOA SRV TXT"
complete -c doggo -n '__fish_doggo_no_subcommand' -s 'n' -l 'nameserver' -d "Address of a specific nameserver to send queries to" -x -a "(__fish_print_hostnames)"
complete -c doggo -n '__fish_doggo_no_subcommand' -s 'c' -l 'class'      -d "Network class of the DNS record being queried" -x -a "IN CH HS"
complete -c doggo -n '__fish_doggo_no_subcommand' -s 'r' -l 'reverse'    -d "Performs a DNS Lookup for an IPv4 or IPv6 address"

# Resolver options
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'strategy'  -d "Strategy to query nameserver listed in etc/resolv.conf" -x -a "all random first"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'ndots'     -d "Specify ndots parameter"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'search'    -d "Use the search list defined in resolv.conf" -x -a "true false"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'timeout'   -d "Specify timeout (in seconds) for the resolver to return a response"
complete -c doggo -n '__fish_doggo_no_subcommand' -s '4' -l 'ipv4' -d "Use IPv4 only"
complete -c doggo -n '__fish_doggo_no_subcommand' -s '6' -l 'ipv6' -d "Use IPv6 only"

# Output options
complete -c doggo -n '__fish_doggo_no_subcommand' -s 'J' -l 'json'  -d "Format the output as JSON"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'short'        -d "Shows only the response section in the output"
complete -c doggo -n '__fish_doggo_no_subcommand' -s '1' -l 'single' -d "Shows only the single address"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'color'        -d "Colored output" -x -a "true false"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'debug'        -d "Enable debug logging"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'time'         -d "Shows how long the response took from the server"

# TLS options
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'tls-hostname'               -d "Hostname for certificate verification" -x -a "(__fish_print_hostnames)"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'skip-hostname-verification' -d "Skip TLS hostname verification in case of DoT lookups"

# Globalping options
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'gp-from'  -d "Query using Globalping API from a specific location"
complete -c doggo -n '__fish_doggo_no_subcommand' -l 'gp-limit' -d "Limit the number of probes to use from Globalping"

# Completions command
complete -c doggo -n '__fish_doggo_no_subcommand' -a completions -d "Generate shell completion scripts"
complete -c doggo -n '__fish_seen_subcommand_from completions' -a "bash zsh fish" -d "Shell type"
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
