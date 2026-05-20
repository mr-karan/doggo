package resolvers

import (
	"reflect"
	"testing"

	"github.com/miekg/dns"
)

func TestPrepareMessagesEDNS(t *testing.T) {
	q := dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}

	tests := []struct {
		name        string
		flags       QueryFlags
		wantEDNS    bool
		wantBufsize uint16
	}{
		{
			name:     "no EDNS options omits OPT record",
			flags:    QueryFlags{RD: true},
			wantEDNS: false,
		},
		{
			name:        "DO flag advertises default 1232",
			flags:       QueryFlags{RD: true, DO: true},
			wantEDNS:    true,
			wantBufsize: 1232,
		},
		{
			name:        "explicit bufsize is used",
			flags:       QueryFlags{RD: true, Bufsize: 2048},
			wantEDNS:    true,
			wantBufsize: 2048,
		},
		{
			name:        "bufsize alone enables EDNS",
			flags:       QueryFlags{RD: true, Bufsize: 4096},
			wantEDNS:    true,
			wantBufsize: 4096,
		},
		{
			name:        "explicit bufsize overrides default when combined with DO",
			flags:       QueryFlags{RD: true, DO: true, Bufsize: 1500},
			wantEDNS:    true,
			wantBufsize: 1500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := prepareMessages(q, tt.flags, 1, nil)
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}
			opt := msgs[0].IsEdns0()
			if !tt.wantEDNS {
				if opt != nil {
					t.Errorf("expected no OPT record, got %+v", opt)
				}
				return
			}
			if opt == nil {
				t.Fatal("expected OPT record, got nil")
			}
			if got := opt.UDPSize(); got != tt.wantBufsize {
				t.Errorf("UDPSize = %d, want %d", got, tt.wantBufsize)
			}
		})
	}
}

func TestConstructPossibleQuestionsWithRootSearchDomain(t *testing.T) {
	tests := []struct {
		name       string
		qName      string
		ndots      int
		searchList []string
		want       []string
	}{
		{
			name:       "root search does not append an extra trailing dot",
			qName:      "non-existent.test",
			ndots:      0,
			searchList: []string{"."},
			want:       []string{"non-existent.test."},
		},
		{
			name:       "root search is de-duplicated when original name is tried first",
			qName:      "non-existent.test",
			ndots:      1,
			searchList: []string{"."},
			want:       []string{"non-existent.test."},
		},
		{
			name:       "root search can follow regular search domains",
			qName:      "printer",
			ndots:      1,
			searchList: []string{"lan", "."},
			want:       []string{"printer.lan.", "printer."},
		},
		{
			name:       "fully qualified names ignore search domains",
			qName:      "example.com.",
			ndots:      1,
			searchList: []string{"."},
			want:       []string{"example.com."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constructPossibleQuestions(tt.qName, tt.ndots, tt.searchList)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("constructPossibleQuestions() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
