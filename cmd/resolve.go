package main

import (
	"fmt"

	"github.com/miekg/dns"
)

// Resolve resolves the domain name
func (hub *Hub) Resolve() {
	var messages = make([]dns.Msg, 0, len(hub.Domains.Value()))
	for _, d := range hub.Domains.Value() {
		msg := dns.Msg{}
		msg.Id = dns.Id()
		msg.RecursionDesired = true
		msg.Question = []dns.Question{(dns.Question{dns.Fqdn(d), dns.TypeA, dns.ClassINET})}
		messages = append(messages, msg)
	}
	c := new(dns.Client)
	for _, msg := range messages {
		in, rtt, err := c.Exchange(&msg, "127.0.0.1:53")
		if err != nil {
			panic(err)
		}
		for _, ans := range in.Answer {
			if t, ok := ans.(*dns.A); ok {
				fmt.Println(t.String())
			}
		}
		fmt.Println("rtt is", rtt, msg.Question[0].Name)
	}
}
