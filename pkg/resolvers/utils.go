package resolvers

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/idna"
)

// QueryFlags represents the various DNS query flags
type QueryFlags struct {
	AA bool // Authoritative Answer
	AD bool // Authenticated Data
	CD bool // Checking Disabled
	RD bool // Recursion Desired
	Z  bool // Reserved for future use
	DO bool // DNSSEC OK

	// EDNS0 options
	NSID    bool   // Request Name Server Identifier
	Cookie  bool   // Request DNS Cookie
	Padding bool   // Request EDNS padding for privacy
	EDE     bool   // Request Extended DNS Errors
	ECS     string // EDNS Client Subnet (e.g., "192.0.2.0/24" or "2001:db8::/32")
}

// prepareMessages takes a  DNS Question and returns the
// corresponding DNS messages for the same.
func prepareMessages(q dns.Question, flags QueryFlags, ndots int, searchList []string) []dns.Msg {
	var (
		possibleQNames = constructPossibleQuestions(q.Name, ndots, searchList)
		messages       = make([]dns.Msg, 0, len(possibleQNames))
	)

	for _, qName := range possibleQNames {
		msg := dns.Msg{}
		// generate a random id for the transaction.
		msg.Id = dns.Id()

		// Set query flags
		msg.RecursionDesired = flags.RD
		msg.AuthenticatedData = flags.AD
		msg.CheckingDisabled = flags.CD
		msg.Authoritative = flags.AA
		msg.Zero = flags.Z

		// Set EDNS0 if any EDNS options are requested
		if flags.DO || flags.NSID || flags.Cookie || flags.Padding || flags.EDE || flags.ECS != "" {
			msg.SetEdns0(4096, flags.DO)

			// Add EDNS0 options
			opt := msg.IsEdns0()
			if opt != nil {
				if flags.NSID {
					nsid := &dns.EDNS0_NSID{}
					opt.Option = append(opt.Option, nsid)
				}

				if flags.Cookie {
					cookie := &dns.EDNS0_COOKIE{}
					opt.Option = append(opt.Option, cookie)
				}

				if flags.Padding {
					padding := &dns.EDNS0_PADDING{
						Padding: make([]byte, 128), // Standard padding size
					}
					opt.Option = append(opt.Option, padding)
				}

				if flags.EDE {
					// EDE is typically returned by the server, but we can set up
					// the EDNS0 to signal we understand EDE responses
					ede := &dns.EDNS0_EDE{}
					opt.Option = append(opt.Option, ede)
				}

				if flags.ECS != "" {
					subnet, err := parseECS(flags.ECS)
					if err == nil {
						opt.Option = append(opt.Option, subnet)
					}
				}
			}
		}

		// It's recommended to only send 1 question for 1 DNS message.
		msg.Question = []dns.Question{{
			Name:   qName,
			Qtype:  q.Qtype,
			Qclass: q.Qclass,
		}}
		messages = append(messages, msg)
	}

	return messages
}

// NameList returns all of the names that should be queried based on the
// config. It is based off of go's net/dns name building, but it does not
// check the length of the resulting names.
// NOTE: It is taken from `miekg/dns/clientconfig.go: func (c *ClientConfig) NameList`
// and slightly modified.
func constructPossibleQuestions(name string, ndots int, searchList []string) []string {
	// if this domain is already fully qualified, no append needed.
	if dns.IsFqdn(name) {
		return []string{name}
	}

	// Check to see if the name has more labels than Ndots. Do this before making
	// the domain fully qualified.
	hasNdots := dns.CountLabel(name) > ndots
	// Make the domain fully qualified.
	name = dns.Fqdn(name)

	// Make a list of names based off search.
	names := []string{}

	// If name has enough dots, try that first.
	if hasNdots {
		names = append(names, name)
	}
	for _, s := range searchList {
		names = append(names, dns.Fqdn(name+s))
	}
	// If we didn't have enough dots, try after suffixes.
	if !hasNdots {
		names = append(names, name)
	}
	return names
}

// toUnicodeDomain converts a punycode domain name to Unicode.
// If conversion fails, returns the original domain name.
func toUnicodeDomain(name string) string {
	unicodeName, err := idna.ToUnicode(name)
	if err != nil {
		// If conversion fails, return original name
		return name
	}
	return unicodeName
}

// parseECS parses an EDNS Client Subnet string and returns an EDNS0_SUBNET option.
// Accepts formats like "192.0.2.0/24" or "2001:db8::/32".
func parseECS(subnet string) (*dns.EDNS0_SUBNET, error) {
	// Parse the CIDR notation
	parts := strings.Split(subnet, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ECS format: expected 'ip/prefix', got '%s'", subnet)
	}

	ip := strings.TrimSpace(parts[0])
	prefix := parts[1]

	// Parse the prefix length
	var prefixLen int
	_, err := fmt.Sscanf(prefix, "%d", &prefixLen)
	if err != nil {
		return nil, fmt.Errorf("invalid prefix length: %s", prefix)
	}

	// Parse the IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	// Determine if it's IPv4 or IPv6
	family := uint16(1) // IPv4
	if parsedIP.To4() == nil {
		family = 2 // IPv6
	}

	return &dns.EDNS0_SUBNET{
		Code:          dns.EDNS0SUBNET,
		Family:        family,
		SourceNetmask: uint8(prefixLen),
		SourceScope:   0,
		Address:       parsedIP,
	}, nil
}

// parseMessage takes a `dns.Message` and returns a custom
// Response data struct.
func parseMessage(msg *dns.Msg, rtt time.Duration, server string) Response {
	var resp Response
	timeTaken := fmt.Sprintf("%dms", rtt.Milliseconds())

	// Parse Authorities section.
	for _, ns := range msg.Ns {
		// check for SOA record
		soa, ok := ns.(*dns.SOA)
		if !ok {
			// Currently we only check for SOA in Authority.
			// If it's not SOA, skip this message.
			continue
		}
		mname := soa.Ns + " " + soa.Mbox +
			" " + strconv.FormatInt(int64(soa.Serial), 10) +
			" " + strconv.FormatInt(int64(soa.Refresh), 10) +
			" " + strconv.FormatInt(int64(soa.Retry), 10) +
			" " + strconv.FormatInt(int64(soa.Expire), 10) +
			" " + strconv.FormatInt(int64(soa.Minttl), 10)
		h := ns.Header()
		name := toUnicodeDomain(h.Name)
		qclass := dns.Class(h.Class).String()
		ttl := strconv.FormatInt(int64(h.Ttl), 10) + "s"
		qtype := dns.Type(h.Rrtype).String()
		auth := Authority{
			Name:       name,
			Type:       qtype,
			TTL:        ttl,
			Class:      qclass,
			MName:      mname,
			Nameserver: server,
			RTT:        timeTaken,
			Status:     dns.RcodeToString[msg.Rcode],
		}
		resp.Authorities = append(resp.Authorities, auth)
	}
	// Parse Answers section.
	for _, a := range msg.Answer {
		var (
			h = a.Header()
			// Source https://github.com/jvns/dns-lookup/blob/main/dns.go#L121.
			parts = strings.Split(a.String(), "\t")
			ans   = Answer{
				Name:       toUnicodeDomain(h.Name),
				Type:       dns.Type(h.Rrtype).String(),
				TTL:        strconv.FormatInt(int64(h.Ttl), 10) + "s",
				Class:      dns.Class(h.Class).String(),
				Address:    parts[len(parts)-1],
				RTT:        timeTaken,
				Nameserver: server,
			}
		)

		resp.Answers = append(resp.Answers, ans)
	}
	return resp
}
