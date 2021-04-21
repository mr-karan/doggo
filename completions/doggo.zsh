#compdef doggo

__doggo() {
    _arguments \
        "(- 1 *)"{-v,--version}"[Show version of doggo]" \
        "(- 1 *)"{-\?,--help}"[Show list of command-line options]" \
        {-q,--query}"[Hostname to query the DNS records for]::_hosts" \
        {-t,--type}"[Type of the DNS Record]:(record type):(A AAAA CAA CNAME HINFO MX NS PTR SOA SRV TXT)" \
        {-n,--nameserver}"[Address of a specific nameserver to send queries to]::_hosts;" \
        {-c,--class}"[Network class of the DNS record being queried]:(network class):(IN CH HS)" \
        {-J,--json}"[Format the output as JSON]" \
        {--color}"[Defaults to true. Set --color=false to disable colored output]:(setting):(true false)" \
        {--debug}"[Enable debug logging]:(setting):(true false)" \
        --time"[Shows how long the response took from the server"] \
        '*:filename:_hosts'
}

__doggo
