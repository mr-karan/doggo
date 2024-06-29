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

    opts="-v --version -h --help -q --query -t --type -n --nameserver -c --class -r --reverse --strategy --ndots --search --timeout -4 --ipv4 -6 --ipv6 --tls-hostname --skip-hostname-verification -J --json --short --color --debug --time"

    case "${prev}" in
        -t|--type)
            COMPREPLY=( $(compgen -W "A AAAA CAA CNAME HINFO MX NS PTR SOA SRV TXT" -- ${cur}) )
            return 0
            ;;
        -c|--class)
            COMPREPLY=( $(compgen -W "IN CH HS" -- ${cur}) )
            return 0
            ;;
        --strategy)
            COMPREPLY=( $(compgen -W "all random first" -- ${cur}) )
            return 0
            ;;
        --search|--color|--debug)
            COMPREPLY=( $(compgen -W "true false" -- ${cur}) )
            return 0
            ;;
    esac

    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi
}

complete -F _doggo doggo
`

	zshCompletion = `
#compdef _doggo doggo

_doggo() {
    _arguments \
        "(- 1 *)"{-v,--version}"[Show version of doggo]" \
        "(- 1 *)"{-h,--help}"[Show list of command-line options]" \
        {-q,--query=}"[Hostname to query the DNS records for]::_hosts" \
        {-t,--type=}"[Type of the DNS Record]:(record type):(A AAAA CAA CNAME HINFO MX NS PTR SOA SRV TXT)" \
        {-n,--nameserver=}"[Address of a specific nameserver to send queries to]::_hosts;" \
        {-c,--class=}"[Network class of the DNS record being queried]:(network class):(IN CH HS)" \
        {-r,--reverse}"[Performs a DNS Lookup for an IPv4 or IPv6 address]" \
        --strategy="[Strategy to query nameserver listed in etc/resolv.conf]:(strategy):(all random first)" \
        --ndots="[Number of requred dots in hostname to assume FQDN]:(number of dots):()" \
        --search"[Defaults to true. Set --search=false to not use the search list defined in resolve.conf]:(setting):(true false)" \
        --timeout"[Timeout (in seconds) for the resolver to return a response]:(seconds):()" \
        {-4,--ipv4}"[Use IPv4 only]" \
        {-6,--ipv6}"[Use IPv6 only]" \
        --tls-hostname="[Hostname used for verification of certificate incase the provided DoT nameserver is an IP]::_hosts" \
        --skip-hostname-verification"[Skip TLS hostname verification in case of DoT lookups]" \
        {-J,--json}"[Format the output as JSON]" \
        --short"[Shows only the response section in the output]" \
        --color="[Defaults to true. Set --color=false to disable colored output]:(setting):(true false)" \
        --debug"[Enable debug logging]:(setting):(true false)" \
        --time"[Shows how long the response took from the server]" \
        '*:hostname:_hosts'
}
`

	fishCompletion = `
# Meta options
complete -c doggo -l 'version' -d "Show version of doggo"
complete -c doggo -l 'help'    -d "Show list of command-line options"

# Single line all options
complete -c doggo -x -a "(__fish_print_hostnames) A AAAA CAA CNAME HINFO MX NS PTR SOA SRV TXT IN CH HS"

# Query options
complete -c doggo -s 'q' -l 'query'      -d "Hostname to query the DNS records for" -x -a "(__fish_print_hostnames)"
complete -c doggo -s 't' -l 'type'       -d "Type of the DNS Record" -x -a "A AAAA CAA CNAME HINFO MX NS PTR SOA SRV TXT"
complete -c doggo -s 'n' -l 'nameserver' -d "Address of a specific nameserver to send queries to" -x -a "1.1.1.1 8.8.8.8 9.9.9.9 (__fish_print_hostnames)"
complete -c doggo -s 'c' -l 'class'      -d "Network class of the DNS record being queried" -x -a "IN CH HS"

# Transport options
complete -c doggo -x -a "@udp:// @tcp:// @https:// @tls:// @sdns://"          -d "Select the protocol for resolving queries"

# Resolver options
complete -c doggo -l 'ndots'             -d "Specify ndots parameter. Takes value from /etc/resolv.conf if using the system namesever or 1 otherwise"
complete -c doggo -l 'search'            -d "Use the search list defined in resolv.conf. Defaults to true. Set --search=false to disable search list"
complete -c doggo -l 'timeout'           -d "Specify timeout (in seconds) for the resolver to return a response"
complete -c doggo -s '-4' -l 'ipv4'      -d "Use IPv4 only"
complete -c doggo -s '-6' -l 'ipv6'      -d "Use IPv6 only"

# Output options
complete -c doggo -s 'J' -l 'json'       -d "Format the output as JSON"
complete -c doggo        -l 'color'      -d "Defaults to true. Set --color=false to disable colored output"
complete -c doggo        -l 'debug'      -d "Enable debug logging"
complete -c doggo        -l 'time'       -d "Shows how long the response took from the server"
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
