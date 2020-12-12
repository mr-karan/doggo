package resolvers

import "github.com/miekg/dns"

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
