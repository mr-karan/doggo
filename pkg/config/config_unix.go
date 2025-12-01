// +build !windows,!darwin

package config

import (
	"net"

	"github.com/miekg/dns"
)

// DefaultResolvConfPath specifies path to default resolv config file on UNIX.
var DefaultResolvConfPath = "/etc/resolv.conf"

// GetDefaultServers get system default nameserver
func GetDefaultServers() ([]string, int, []string, error) {
	// if no nameserver is provided, take it from `resolv.conf`
	cfg, err := dns.ClientConfigFromFile(DefaultResolvConfPath)
	if err != nil {
		return nil, 0, nil, err
	}
	servers := make([]string, 0)
	for _, server := range cfg.Servers {
		ip := net.ParseIP(server)
		if isUnicastLinkLocal(ip) {
			continue
		}
		servers = append(servers, server)
	}
	return servers, cfg.Ndots, cfg.Search, nil
}
