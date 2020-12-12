package resolvers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

// DOHResolver represents the config options for setting up a DOH based resolver.
type DOHResolver struct {
	client  *http.Client
	servers []string
}

// NewDOHResolver accepts a list of nameservers and configures a DOH based resolver.
func NewDOHResolver(servers []string) (Resolver, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	return &DOHResolver{
		client:  httpClient,
		servers: servers,
	}, nil
}

func (r *DOHResolver) Lookup(questions []dns.Question) error {
	messages := prepareMessages(questions)

	for _, m := range messages {
		b, err := m.Pack()
		if err != nil {
			return err
		}
		for _, srv := range r.servers {
			resp, err := r.client.Post(srv, "application/dns-message", bytes.NewBuffer(b))
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return err
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			r := &dns.Msg{}
			err = r.Unpack(body)
			if err != nil {
				return err
			}
			for _, ans := range r.Answer {
				if t, ok := ans.(*dns.A); ok {
					fmt.Println(t.String())
				}
			}
		}
	}
	return nil
}
