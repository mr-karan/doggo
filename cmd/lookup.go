package main

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/urfave/cli/v2"
)

// Lookup sends the DNS queries to the server.
func (hub *Hub) Lookup(c *cli.Context) error {
	hub.prepareQuestions()
	responses, err := hub.Resolver.Lookup(hub.Questions)
	if err != nil {
		hub.Logger.Error(err)
	}
	for _, r := range responses {
		for _, a := range r.Message.Answer {
			if t, ok := a.(*dns.A); ok {
				fmt.Println(t.String())
			}
		}
	}

	return nil
}

// prepareQuestions iterates on list of domain names
// and prepare a list of questions
// sent to the server with all possible combinations.
func (hub *Hub) prepareQuestions() {
	var question dns.Question
	for _, name := range hub.QueryFlags.QNames.Value() {
		question.Name = dns.Fqdn(name)
		// iterate on a list of query types.
		for _, q := range hub.QueryFlags.QTypes.Value() {
			question.Qtype = dns.StringToType[strings.ToUpper(q)]
			// iterate on a list of query classes.
			for _, c := range hub.QueryFlags.QClasses.Value() {
				question.Qclass = dns.StringToClass[strings.ToUpper(c)]
				// append a new question for each possible pair.
				hub.Questions = append(hub.Questions, question)
			}
		}
	}
}
