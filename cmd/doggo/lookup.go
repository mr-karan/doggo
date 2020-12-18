package main

import (
	"runtime"
	"strings"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/sirupsen/logrus"
)

// Lookup sends the DNS queries to the server.
// It prepares a list of `dns.Questions` and sends
// to all resolvers. It returns a list of []resolver.Response from
// each resolver
func (hub *Hub) Lookup() ([][]resolvers.Response, error) {
	questions, err := hub.prepareQuestions()
	if err != nil {
		return nil, err
	}
	hub.Questions = questions
	// for each type of resolver do a DNS lookup
	responses := make([][]resolvers.Response, 0, len(hub.Questions))
	for _, r := range hub.Resolver {
		resp, err := r.Lookup(hub.Questions)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

// prepareQuestions takes a list of hostnames and some
// additional options and returns a list of all possible
// `dns.Questions`.
func (hub *Hub) prepareQuestions() ([]dns.Question, error) {
	var (
		questions []dns.Question
	)
	for _, name := range hub.QueryFlags.QNames {
		var (
			domains []string
		)
		// If `search` flag is specified then fetch the search list
		// from `resolv.conf` and set the
		if hub.QueryFlags.UseSearchList {
			list, err := fetchDomainList(name, hub.QueryFlags.Ndots)
			if err != nil {
				return nil, err
			}
			domains = list
		} else {
			domains = []string{dns.Fqdn(name)}
		}
		for _, d := range domains {
			hub.Logger.WithFields(logrus.Fields{
				"domain": d,
				"ndots":  hub.QueryFlags.Ndots,
			}).Debug("Attempting to resolve")
			question := dns.Question{
				Name: d,
			}
			// iterate on a list of query types.
			for _, q := range hub.QueryFlags.QTypes {
				question.Qtype = dns.StringToType[strings.ToUpper(q)]
				// iterate on a list of query classes.
				for _, c := range hub.QueryFlags.QClasses {
					question.Qclass = dns.StringToClass[strings.ToUpper(c)]
					// append a new question for each possible pair.
					questions = append(questions, question)
				}
			}
		}
	}
	return questions, nil
}

func fetchDomainList(d string, ndots int) ([]string, error) {
	if runtime.GOOS == "windows" {
		// TODO: Add a method for reading system default nameserver in windows.
		return []string{d}, nil
	}
	cfg, err := dns.ClientConfigFromFile(DefaultResolvConfPath)
	if err != nil {
		return nil, err
	}
	cfg.Ndots = ndots
	return cfg.NameList(d), nil
}
