// +build !windows

package config

import (
	"github.com/miekg/dns"
)

//DefaultResolvConfPath specifies path to default resolv config file on UNIX.
const DefaultResolvConfPath = "/etc/resolv.conf"

func GetDefaultServers() ([]string, int, []string, error) {
	// if no nameserver is provided, take it from `resolv.conf`
	cfg, err := dns.ClientConfigFromFile(DefaultResolvConfPath)
	if err != nil {
		return nil, 0, nil, err
	}
	return cfg.Servers, cfg.Ndots, cfg.Search, nil
}
