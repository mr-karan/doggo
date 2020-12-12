package main

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/urfave/cli/v2"
)

func (hub *Hub) Lookup(c *cli.Context) error {
	hub.prepareQuestions()
	err := hub.Resolver.Lookup(hub.Questions)
	if err != nil {
		fmt.Println(err)
		hub.Logger.Error(err)
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
