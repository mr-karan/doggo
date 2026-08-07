package app

import (
	"testing"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/models"
)

func TestPrepareQuestionsParsesRecordTypeForms(t *testing.T) {
	app := App{QueryFlags: models.QueryFlags{
		QNames:   []string{"example.test"},
		QTypes:   []string{"HTTPS", "SVCB", "TYPE65", "65"},
		QClasses: []string{"IN"},
	}}
	if err := app.PrepareQuestions(); err != nil {
		t.Fatalf("PrepareQuestions: %v", err)
	}

	want := []uint16{dns.TypeHTTPS, dns.TypeSVCB, dns.TypeHTTPS, dns.TypeHTTPS}
	if len(app.Questions) != len(want) {
		t.Fatalf("len(Questions) = %d, want %d", len(app.Questions), len(want))
	}
	for i := range want {
		if app.Questions[i].Qtype != want[i] {
			t.Errorf("Questions[%d].Qtype = %d, want %d", i, app.Questions[i].Qtype, want[i])
		}
	}
}

func TestPrepareQuestionsRejectsInvalidRecordTypes(t *testing.T) {
	for _, recordType := range []string{"NOTATYPE", "0", "TYPE0", "OPT", "TKEY", "TSIG", "TYPE65536", "65536"} {
		t.Run(recordType, func(t *testing.T) {
			app := App{QueryFlags: models.QueryFlags{
				QNames:   []string{"example.test"},
				QTypes:   []string{"a", recordType},
				QClasses: []string{"IN"},
			}}
			if err := app.PrepareQuestions(); err == nil {
				t.Fatal("PrepareQuestions = nil error, want failure")
			}
			if len(app.Questions) != 0 {
				t.Fatalf("PrepareQuestions appended %d questions before failing", len(app.Questions))
			}
			if app.QueryFlags.QTypes[0] != "a" {
				t.Fatalf("QTypes was partially normalized on failure: %v", app.QueryFlags.QTypes)
			}
		})
	}
}

func TestPrepareQuestionsPreservesTransferQTypes(t *testing.T) {
	app := App{QueryFlags: models.QueryFlags{
		QNames:   []string{"example.test"},
		QTypes:   []string{"AXFR", "IXFR"},
		QClasses: []string{"IN"},
	}}
	if err := app.PrepareQuestions(); err != nil {
		t.Fatalf("PrepareQuestions: %v", err)
	}
	want := []uint16{dns.TypeAXFR, dns.TypeIXFR}
	for i := range want {
		if app.Questions[i].Qtype != want[i] {
			t.Errorf("Questions[%d].Qtype = %d, want %d", i, app.Questions[i].Qtype, want[i])
		}
	}
}
