package main

import "testing"

func TestLoadUnparsedArgsRecognizesRecordTypeForms(t *testing.T) {
	_, queryTypes, _, queryNames, err := loadUnparsedArgs([]string{
		"HTTPS", "svcb", "TYPE65", "64", "example.com",
	})
	if err != nil {
		t.Fatalf("loadUnparsedArgs: %v", err)
	}
	wantTypes := []string{"HTTPS", "SVCB", "HTTPS", "SVCB"}
	for i := range wantTypes {
		if queryTypes[i] != wantTypes[i] {
			t.Errorf("queryTypes[%d] = %q, want %q", i, queryTypes[i], wantTypes[i])
		}
	}
	if len(queryNames) != 1 || queryNames[0] != "example.com" {
		t.Fatalf("queryNames = %v, want [example.com]", queryNames)
	}
}

func TestLoadUnparsedArgsRejectsMalformedNumericRecordTypes(t *testing.T) {
	for _, input := range []string{"0", "TYPE0", "OPT", "41", "TYPE41", "TKEY", "TSIG", "TYPE-1", "TYPE65536", "65536"} {
		t.Run(input, func(t *testing.T) {
			if _, _, _, _, err := loadUnparsedArgs([]string{input, "example.com"}); err == nil {
				t.Fatalf("loadUnparsedArgs(%q) = nil error, want failure", input)
			}
		})
	}
}

func TestLoadUnparsedArgsRecognizesPositionalClasses(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "NONE", want: "NONE"},
		{input: "None", want: "NONE"},
		{input: "CH", want: "CH"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			_, queryTypes, queryClasses, queryNames, err := loadUnparsedArgs([]string{test.input, "example.com"})
			if err != nil {
				t.Fatalf("loadUnparsedArgs: %v", err)
			}
			if len(queryTypes) != 0 {
				t.Fatalf("queryTypes = %v, want none", queryTypes)
			}
			if len(queryClasses) != 1 || queryClasses[0] != test.want {
				t.Fatalf("queryClasses = %v, want [%s]", queryClasses, test.want)
			}
			if len(queryNames) != 1 || queryNames[0] != "example.com" {
				t.Fatalf("queryNames = %v, want [example.com]", queryNames)
			}
		})
	}
}

func TestLoadUnparsedArgsKeepsUnknownNamesAsQueries(t *testing.T) {
	_, queryTypes, _, queryNames, err := loadUnparsedArgs([]string{"types.test", "NOTATYPE", "example.com"})
	if err != nil {
		t.Fatalf("loadUnparsedArgs: %v", err)
	}
	if len(queryTypes) != 0 {
		t.Fatalf("queryTypes = %v, want none", queryTypes)
	}
	wantNames := []string{"types.test", "NOTATYPE", "example.com"}
	for i := range wantNames {
		if queryNames[i] != wantNames[i] {
			t.Errorf("queryNames[%d] = %q, want %q", i, queryNames[i], wantNames[i])
		}
	}
}
