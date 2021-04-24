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