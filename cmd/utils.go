package main

import "unicode"

var (
	QTYPES = []string{"A", "AAAA", "MX"}
	QCLASS = []string{"CN", "AAAA", "MX"}
)

func isUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func parseQueryType(s string) bool {
	for _, b := range QTYPES {
		if b == s {
			return true
		}
	}
	return false
}

func parseQueryClass(s string) bool {
	for _, b := range QCLASS {
		if b == s {
			return true
		}
	}
	return false
}
