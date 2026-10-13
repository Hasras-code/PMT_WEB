package student

import "testing"

func TestParseCombination(t *testing.T) {
	for _, test := range []struct {
		input string
		want  Combination
		ok    bool
	}{
		{input: "PMT-ICT", want: CombinationPMTICT, ok: true},
		{input: " pmt-cs ", want: CombinationPMTCS, ok: true},
		{input: "pmt-stat", want: CombinationPMTSTAT, ok: true},
		{input: "PMT-MAN", want: CombinationPMTMAN, ok: true},
		{input: " PMT-CHEM ", want: CombinationPMTCHEM, ok: true},
		{input: "", ok: false},
		{input: "ICT", ok: false},
		{input: "PMT-ICT,PMT-CS", ok: false},
	} {
		got, ok := ParseCombination(test.input)
		if got != test.want || ok != test.ok {
			t.Errorf("ParseCombination(%q) = %q, %t; want %q, %t", test.input, got, ok, test.want, test.ok)
		}
	}
}

func TestCombinationValues(t *testing.T) {
	want := []string{"PMT-ICT", "PMT-CS", "PMT-STAT", "PMT-MAN", "PMT-CHEM"}
	got := CombinationValues()
	if len(got) != len(want) {
		t.Fatalf("CombinationValues() = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("CombinationValues() = %v; want %v", got, want)
		}
	}
}
