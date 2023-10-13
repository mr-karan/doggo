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
