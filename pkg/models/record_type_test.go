package models

import (
	"testing"

	"github.com/miekg/dns"
)

func TestParseRecordType(t *testing.T) {
	tests := []struct {
		input string
		want  uint16
	}{
		{input: "A", want: dns.TypeA},
		{input: "aaaa", want: dns.TypeAAAA},
		{input: "HTTPS", want: dns.TypeHTTPS},
		{input: "svcb", want: dns.TypeSVCB},
		{input: "65", want: dns.TypeHTTPS},
		{input: "TYPE65", want: dns.TypeHTTPS},
		{input: "type64", want: dns.TypeSVCB},
		{input: "TYPE65400", want: 65400},
		{input: "65535", want: 65535},
		{input: "TYPE65535", want: 65535},
		{input: "AXFR", want: dns.TypeAXFR},
		{input: "IXFR", want: dns.TypeIXFR},
		{input: "252", want: dns.TypeAXFR},
		{input: "TYPE251", want: dns.TypeIXFR},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := ParseRecordType(test.input)
			if err != nil {
				t.Fatalf("ParseRecordType(%q): %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("ParseRecordType(%q) = %d, want %d", test.input, got, test.want)
			}
		})
	}
}

func TestParseRecordTypeRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "NOTATYPE", "None", "NONE", "Reserved", "TYPE", "TYPE-1", "TYPE65536", "65536", "-1", "1.5"} {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseRecordType(input); err == nil {
				t.Fatalf("ParseRecordType(%q) = nil error, want failure", input)
			}
		})
	}
}

func TestParseRecordTypeRejectsReservedAndMetaTypes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "0", want: `DNS record type "0" is reserved and cannot be used in a question`},
		{input: "TYPE0", want: `DNS record type "TYPE0" is reserved and cannot be used in a question`},
		{input: "OPT", want: `DNS record type "OPT" (41) cannot be used in a question`},
		{input: "41", want: `DNS record type "41" (41) cannot be used in a question`},
		{input: "TYPE41", want: `DNS record type "TYPE41" (41) cannot be used in a question`},
		{input: "TKEY", want: `DNS record type "TKEY" (249) cannot be used in a question`},
		{input: "TSIG", want: `DNS record type "TSIG" (250) cannot be used in a question`},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			_, err := ParseRecordType(test.input)
			if err == nil {
				t.Fatalf("ParseRecordType(%q) = nil error, want failure", test.input)
			}
			if err.Error() != test.want {
				t.Fatalf("ParseRecordType(%q) error = %q, want %q", test.input, err, test.want)
			}
		})
	}
}

func TestNormalizeRecordTypes(t *testing.T) {
	got, err := NormalizeRecordTypes([]string{"https", "TYPE64", "65", "65400", "65535"})
	if err != nil {
		t.Fatalf("NormalizeRecordTypes: %v", err)
	}
	want := []string{"HTTPS", "SVCB", "HTTPS", "TYPE65400", "TYPE65535"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("normalized[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeRecordTypesRejectsZeroWithoutPartialOutput(t *testing.T) {
	got, err := NormalizeRecordTypes([]string{"a", "TYPE0"})
	if err == nil {
		t.Fatal("NormalizeRecordTypes = nil error, want failure")
	}
	if got != nil {
		t.Fatalf("NormalizeRecordTypes output = %v, want nil", got)
	}
}

func TestRecordTypeStringReservedBoundary(t *testing.T) {
	if got := RecordTypeString(dns.TypeReserved); got != "TYPE65535" {
		t.Fatalf("RecordTypeString(65535) = %q, want TYPE65535", got)
	}
	if got := RecordTypeString(dns.TypeNone); got != "TYPE0" {
		t.Fatalf("RecordTypeString(0) = %q, want TYPE0", got)
	}
}
