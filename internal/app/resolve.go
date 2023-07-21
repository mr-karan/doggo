package app

import (
	"github.com/hashicorp/go-multierror"
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/pkg/errors"
	"net"
	"time"
)

// Resolve resolves the given DNS queries using the configured resolvers
func (app *App) Resolve() (_ []resolvers.Response, err error) {
	app.LoadFallbacks()
	app.PrepareQuestions()

	var responses []resolvers.Response

	for _, q := range app.Questions {
		for _, rslv := range app.Resolvers {
			if resp, lookupError := rslv.Lookup(q); lookupError != nil {
				err = multierror.Append(err, lookupError)
			} else {
				responses = append(responses, resp)
			}
		}
	}

	return responses, err
}

// Trace traces the resolution path for a dns query.
//
// It resolves the query from the root nameservers downwards and return the results from each query step.
// It will only use the default or explicitly specified nameserver for the initial discovery of the root nameservers.
// Thereafter, it makes its own queries following the delegation referrals it receives.
//
// adapted from: https://superuser.com/a/715656
func (app *App) Trace() (result []resolvers.Response, err error) {
	// pick the first resolver configured
	var rslv = app.Resolvers[0]
	app.Logger.Debugf("using %v resolver", rslv)

	var ques = dns.Question{Name: app.QueryFlags.QNames[0], Qtype: dns.TypeA, Qclass: dns.ClassINET}
	var ans resolvers.Response

	// first, we ask the configured nameserver for NS record for "." (root)
	if ans, err = rslv.Lookup(dns.Question{Name: ".", Qtype: dns.TypeNS, Qclass: dns.ClassINET}); err != nil {
		return nil, errors.Wrapf(err, "failed to lookup NS record for root")
	}
	result = append(result, ans)

	var nameservers []string
	for _, auth := range ans.Answers {
		if auth.Type == dns.TypeToString[dns.TypeNS] {
			nameservers = append(nameservers, auth.Address)
		}
	}

	var classicResolverOpts = resolvers.ClassicResolverOpts{UseTLS: false, UseTCP: false}
	var resolverOpts = resolvers.Options{
		Nameservers:        app.Nameservers,
		UseIPv4:            app.QueryFlags.UseIPv4,
		UseIPv6:            app.QueryFlags.UseIPv6,
		SearchList:         app.ResolverOpts.SearchList,
		Ndots:              app.ResolverOpts.Ndots,
		Timeout:            app.QueryFlags.Timeout * time.Second,
		Logger:             app.Logger,
		Strategy:           app.QueryFlags.Strategy,
		InsecureSkipVerify: app.QueryFlags.InsecureSkipVerify,
		TLSHostname:        app.QueryFlags.TLSHostname,
	}

	// TODO: randomize picking of nameservers (or query all nameservers?)
	var nameserver = net.JoinHostPort(nameservers[0], "53")
	for {
		rslv, _ = resolvers.NewClassicResolver(nameserver, classicResolverOpts, resolverOpts) // safe to suppress here
		if ans, err = rslv.Lookup(ques); err != nil {
			return nil, errors.Wrapf(err, "failed to trace %q", ques.Name)
		}
		result = append(result, ans)

		var prev = nameserver
		// pick the first authority response of type NS
		for _, auth := range ans.Authorities {
			if auth.Type == dns.TypeToString[dns.TypeNS] {
				nameserver = net.JoinHostPort(auth.Address, "53")
				break
			}
		}

		// in case there is no change in the nameserver, does that mean we have reached end of delegation chain?
		// currently, we are assuming that and using it as a terminating condition
		if len(ans.Answers) > 0 || nameserver == prev {
			break
		}
	}

	return result, nil
}
