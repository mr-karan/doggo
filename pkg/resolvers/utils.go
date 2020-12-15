package resolvers

import (
	"errors"
	"fmt"
	"net"
	"runtime"

	"github.com/miekg/dns"
)

// prepareMessages takes a slice fo `dns.Question`
// and initialises `dns.Messages` for each question
func prepareMessages(questions []dns.Question) []dns.Msg {
	var messages = make([]dns.Msg, 0, len(questions))
	for _, q := range questions {
		msg := dns.Msg{}
		msg.Id = dns.Id()
		msg.RecursionDesired = true
		// It's recommended to only send 1 question for 1 DNS message.
		msg.Question = []dns.Question{q}
		messages = append(messages, msg)
	}
	return messages
}

func getDefaultServers() ([]string, error) {
	if runtime.GOOS == "windows" {
		// TODO: Add a method for reading system default nameserver in windows.
		return nil, errors.New(`unable to read default nameservers in this machine`)
	}
	// if no nameserver is provided, take it from `resolv.conf`
	cfg, err := dns.ClientConfigFromFile(DefaultResolvConfPath)
	if err != nil {
		return nil, err
	}
	servers := make([]string, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		ip := net.ParseIP(s)
		// handle IPv6
		if ip != nil && ip.To4() != nil {
			servers = append(servers, fmt.Sprintf("%s:%s", s, cfg.Port))
		} else {
			servers = append(servers, fmt.Sprintf("[%s]:%s", s, cfg.Port))
		}
	}
	return servers, nil
}
