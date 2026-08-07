package resolvers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestParseMessageUsesGenericNamesForReservedTypesInAllSections(t *testing.T) {
	for _, test := range []struct {
		name       string
		recordType uint16
		want       string
	}{
		{name: "zero", recordType: dns.TypeNone, want: "TYPE0"},
		{name: "upper boundary", recordType: dns.TypeReserved, want: "TYPE65535"},
	} {
		t.Run(test.name, func(t *testing.T) {
			header := dns.RR_Header{
				Name:   "example.test.",
				Rrtype: test.recordType,
				Class:  dns.ClassINET,
				Ttl:    60,
			}
			msg := &dns.Msg{
				Answer: []dns.RR{&dns.RFC3597{Hdr: header, Rdata: "beef"}},
				Ns: []dns.RR{&dns.SOA{
					Hdr:     header,
					Ns:      "ns.example.test.",
					Mbox:    "hostmaster.example.test.",
					Serial:  1,
					Refresh: 2,
					Retry:   3,
					Expire:  4,
					Minttl:  5,
				}},
				Extra: []dns.RR{&dns.RFC3597{Hdr: header, Rdata: "cafe"}},
			}

			response := parseMessage(msg, time.Millisecond, "127.0.0.1:53")
			if len(response.Answers) != 1 || response.Answers[0].Type != test.want {
				t.Fatalf("answer types = %+v, want %q", response.Answers, test.want)
			}
			if len(response.Authorities) != 1 || response.Authorities[0].Type != test.want {
				t.Fatalf("authority types = %+v, want %q", response.Authorities, test.want)
			}
			if len(response.Additional) != 1 || response.Additional[0].Type != test.want {
				t.Fatalf("additional types = %+v, want %q", response.Additional, test.want)
			}
		})
	}
}

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

func TestPrepareMessagesEDEUsesOPTWithoutEDEOption(t *testing.T) {
	q := dns.Question{Name: "example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}

	tests := []struct {
		name        string
		flags       QueryFlags
		wantOptions []uint16
	}{
		{
			name:        "EDE alone adds only an OPT record",
			flags:       QueryFlags{EDE: true},
			wantOptions: []uint16{},
		},
		{
			name: "EDE preserves other EDNS options",
			flags: QueryFlags{
				NSID:    true,
				Cookie:  true,
				Padding: true,
				EDE:     true,
				ECS:     "192.0.2.0/24",
			},
			wantOptions: []uint16{dns.EDNS0NSID, dns.EDNS0COOKIE, dns.EDNS0PADDING, dns.EDNS0SUBNET},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := prepareMessages(q, tt.flags, 1, nil)
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}

			opt := msgs[0].IsEdns0()
			if opt == nil {
				t.Fatal("expected OPT record, got nil")
			}
			if got := opt.UDPSize(); got != 1232 {
				t.Errorf("UDPSize = %d, want 1232", got)
			}

			gotOptions := make([]uint16, 0, len(opt.Option))
			for _, option := range opt.Option {
				gotOptions = append(gotOptions, option.Option())
			}
			if !reflect.DeepEqual(gotOptions, tt.wantOptions) {
				t.Fatalf("EDNS options = %#v, want %#v", gotOptions, tt.wantOptions)
			}
		})
	}
}

func TestParseEdnsExtendedErrors(t *testing.T) {
	tests := []struct {
		name             string
		options          []dns.EDNS0
		wantExtendedErr  string
		wantExtendedErrs []ExtendedError
	}{
		{
			name: "zero EDE options",
		},
		{
			name: "one EDE option",
			options: []dns.EDNS0{
				&dns.EDNS0_EDE{InfoCode: dns.ExtendedErrorCodeBlocked, ExtraText: "blocked by policy"},
			},
			wantExtendedErr: "Code: 15, Info: blocked by policy",
			wantExtendedErrs: []ExtendedError{
				{Code: 15, Description: "Blocked", ExtraText: "blocked by policy"},
			},
		},
		{
			name: "unknown EDE code",
			options: []dns.EDNS0{
				&dns.EDNS0_EDE{InfoCode: 65000, ExtraText: "private error"},
			},
			wantExtendedErr: "Code: 65000, Info: private error",
			wantExtendedErrs: []ExtendedError{
				{Code: 65000, ExtraText: "private error"},
			},
		},
		{
			name: "multiple EDE options preserve response order",
			options: []dns.EDNS0{
				&dns.EDNS0_EDE{InfoCode: dns.ExtendedErrorCodeStaleAnswer},
				&dns.EDNS0_EDE{InfoCode: dns.ExtendedErrorCodeNoReachableAuthority, ExtraText: "time limit exceeded"},
			},
			wantExtendedErr: "Code: 3",
			wantExtendedErrs: []ExtendedError{
				{Code: 3, Description: "Stale Answer"},
				{Code: 22, Description: "No Reachable Authority", ExtraText: "time limit exceeded"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := new(dns.Msg)
			msg.SetEdns0(1232, false)
			msg.IsEdns0().Option = tt.options

			got := parseEdns(msg)
			if got == nil {
				t.Fatal("parseEdns() = nil, want EDNS metadata")
			}
			if got.ExtendedErr != tt.wantExtendedErr {
				t.Errorf("ExtendedErr = %q, want %q", got.ExtendedErr, tt.wantExtendedErr)
			}
			if !reflect.DeepEqual(got.ExtendedErrors, tt.wantExtendedErrs) {
				t.Errorf("ExtendedErrors = %#v, want %#v", got.ExtendedErrors, tt.wantExtendedErrs)
			}
		})
	}
}

func TestMergeEdnsInfoAccumulatesExtendedErrors(t *testing.T) {
	accumulated := &EdnsInfo{
		Nameserver:  "first-resolver:53",
		NSID:        "first-nsid",
		ExtendedErr: "Code: 3, Info: first attempt",
		ExtendedErrors: []ExtendedError{
			{Code: 3, Description: "Stale Answer", ExtraText: "first attempt"},
		},
		UDPSize: 1232,
	}
	latest := &EdnsInfo{
		Nameserver:  "final-resolver:53",
		Cookie:      "final-cookie",
		ExtendedErr: "Code: 22, Info: final attempt",
		ExtendedErrors: []ExtendedError{
			{Code: 22, Description: "No Reachable Authority", ExtraText: "final attempt"},
		},
		UDPSize:  4096,
		DNSSECOk: true,
	}

	got := mergeEdnsInfo(accumulated, latest)
	want := &EdnsInfo{
		Nameserver:  "final-resolver:53",
		Cookie:      "final-cookie",
		ExtendedErr: "Code: 3, Info: first attempt",
		ExtendedErrors: []ExtendedError{
			{Code: 3, Description: "Stale Answer", ExtraText: "first attempt"},
			{Code: 22, Description: "No Reachable Authority", ExtraText: "final attempt"},
		},
		UDPSize:  4096,
		DNSSECOk: true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeEdnsInfo() = %#v, want %#v", got, want)
	}

	got.ExtendedErrors[1].ExtraText = "changed"
	if latest.ExtendedErrors[0].ExtraText != "final attempt" {
		t.Fatal("mergeEdnsInfo() aliased the latest EDE slice")
	}
	if got := mergeEdnsInfo(accumulated, nil); got != accumulated {
		t.Fatal("mergeEdnsInfo(accumulated, nil) should retain accumulated metadata")
	}
}

func TestDOHResolverQueryAccumulatesExtendedErrorsAcrossSearchAttempts(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading DNS request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		query := new(dns.Msg)
		if err := query.Unpack(body); err != nil {
			t.Errorf("unpacking DNS request: %v", err)
			http.Error(w, "bad DNS request", http.StatusBadRequest)
			return
		}

		attempt := attempts.Add(1)
		response := new(dns.Msg)
		response.SetReply(query)
		if attempt == 1 {
			response.Rcode = dns.RcodeServerFailure
			response.SetEdns0(1232, false)
			response.IsEdns0().Option = []dns.EDNS0{
				&dns.EDNS0_EDE{InfoCode: dns.ExtendedErrorCodeStaleAnswer, ExtraText: "search suffix failed"},
			}
		} else {
			response.Rcode = dns.RcodeSuccess
			response.SetEdns0(4096, true)
			response.IsEdns0().Option = []dns.EDNS0{
				&dns.EDNS0_EDE{InfoCode: dns.ExtendedErrorCodeNoReachableAuthority, ExtraText: "bare name failed"},
			}
		}

		wire, err := response.Pack()
		if err != nil {
			t.Errorf("packing DNS response: %v", err)
			http.Error(w, "bad response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(wire)
	}))
	t.Cleanup(server.Close)

	resolver := &DOHResolver{
		client: server.Client(),
		server: server.URL,
		resolverOptions: Options{
			SearchList: []string{"example.test"},
			Ndots:      1,
			Logger:     discardLogger(),
		},
	}
	question := dns.Question{Name: "host", Qtype: dns.TypeA, Qclass: dns.ClassINET}

	got, err := resolver.query(context.Background(), question, QueryFlags{EDE: true})
	if err != nil {
		t.Fatalf("query() error = %v", err)
	}
	if attempts.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", attempts.Load())
	}
	if got.Edns == nil {
		t.Fatal("query() EDNS metadata = nil")
	}
	if got.Edns.Nameserver != server.URL {
		t.Errorf("Nameserver = %q, want %q", got.Edns.Nameserver, server.URL)
	}
	if got.Edns.UDPSize != 4096 || !got.Edns.DNSSECOk {
		t.Errorf("final EDNS metadata = %#v, want UDPSize 4096 and DNSSECOk", got.Edns)
	}
	if got.Edns.ExtendedErr != "Code: 3, Info: search suffix failed" {
		t.Errorf("legacy ExtendedErr = %q, want first attempt", got.Edns.ExtendedErr)
	}
	wantErrors := []ExtendedError{
		{Code: 3, Description: "Stale Answer", ExtraText: "search suffix failed"},
		{Code: 22, Description: "No Reachable Authority", ExtraText: "bare name failed"},
	}
	if !reflect.DeepEqual(got.Edns.ExtendedErrors, wantErrors) {
		t.Errorf("ExtendedErrors = %#v, want %#v", got.Edns.ExtendedErrors, wantErrors)
	}
}

func TestEdnsInfoJSONPreservesLegacyExtendedError(t *testing.T) {
	rawExtraText := "raw\n\x1b]0;terminal-title\x07"
	edns := EdnsInfo{
		ExtendedErr: "Code: 15, Info: " + rawExtraText,
		ExtendedErrors: []ExtendedError{
			{Code: 15, Description: "Blocked", ExtraText: rawExtraText},
			{Code: 22, Description: "No Reachable Authority"},
		},
	}

	encoded, err := json.Marshal(edns)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["extended_error"] != edns.ExtendedErr {
		t.Errorf("extended_error = %#v, want %q", got["extended_error"], edns.ExtendedErr)
	}
	extendedErrors, ok := got["extended_errors"].([]any)
	if !ok || len(extendedErrors) != 2 {
		t.Errorf("extended_errors = %#v, want array with 2 entries", got["extended_errors"])
		return
	}
	firstError, ok := extendedErrors[0].(map[string]any)
	if !ok {
		t.Fatalf("extended_errors[0] = %#v, want object", extendedErrors[0])
	}
	if firstError["extra_text"] != rawExtraText {
		t.Errorf("JSON extra_text = %#v, want raw diagnostic text %q", firstError["extra_text"], rawExtraText)
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
