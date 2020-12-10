package models

// Question represents a given query to the client.
// A question can have multiple domains, multiple nameservers
// but it's the responsibility of the client to send each question
// to the nameserver and collect responses.
type Question struct {
	Domain      []string
	Nameservers []string
	QClass      []uint16
	QType       []uint16
}
