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
