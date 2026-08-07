package models

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

// ParseRecordType accepts a DNS record type by canonical name, decimal
// number, or RFC 3597 generic TYPE<number> notation.
func ParseRecordType(value string) (uint16, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, invalidRecordTypeError(value)
	}

	upper := strings.ToUpper(value)
	if recordType, ok := dns.StringToType[upper]; ok {
		return validateQuestionType(value, recordType)
	}

	number := value
	if strings.HasPrefix(upper, "TYPE") {
		number = value[len("TYPE"):]
		if !isDecimal(number) {
			return 0, fmt.Errorf("invalid DNS record type %q: TYPE must be followed by a decimal number between 0 and 65535", value)
		}
	} else if !isDecimal(number) {
		return 0, invalidRecordTypeError(value)
	}

	recordType, err := strconv.ParseUint(number, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("DNS record type %q is out of range: must be between 0 and 65535", value)
	}
	return validateQuestionType(value, uint16(recordType))
}

// NormalizeRecordTypes validates record types and returns the canonical name
// for each numeric value. Unrecognized values and type 65535 are rendered
// using RFC 3597 TYPE<number> notation.
func NormalizeRecordTypes(values []string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		recordType, err := ParseRecordType(value)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, RecordTypeString(recordType))
	}
	return normalized, nil
}

// RecordTypeString returns the canonical presentation form for a question
// type without exposing miekg/dns's "None" and "Reserved" pseudo-names.
func RecordTypeString(recordType uint16) string {
	if recordType == dns.TypeNone || recordType == dns.TypeReserved {
		return "TYPE" + strconv.FormatUint(uint64(recordType), 10)
	}
	return dns.Type(recordType).String()
}

func validateQuestionType(value string, recordType uint16) (uint16, error) {
	switch recordType {
	case dns.TypeNone:
		return 0, fmt.Errorf("DNS record type %q is reserved and cannot be used in a question", value)
	case dns.TypeOPT, dns.TypeTKEY, dns.TypeTSIG:
		return 0, fmt.Errorf("DNS record type %q (%d) cannot be used in a question", value, recordType)
	default:
		return recordType, nil
	}
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func invalidRecordTypeError(value string) error {
	return fmt.Errorf("invalid DNS record type %q: use a canonical name (such as A or HTTPS), a decimal number, or TYPE<number>", value)
}
