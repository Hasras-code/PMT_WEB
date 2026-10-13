package student

import "strings"

type Combination string

const (
	CombinationPMTICT  Combination = "PMT-ICT"
	CombinationPMTCS   Combination = "PMT-CS"
	CombinationPMTSTAT Combination = "PMT-STAT"
	CombinationPMTMAN  Combination = "PMT-MAN"
	CombinationPMTCHEM Combination = "PMT-CHEM"
)

func ParseCombination(value string) (Combination, bool) {
	combination := Combination(strings.ToUpper(strings.TrimSpace(value)))
	switch combination {
	case CombinationPMTICT, CombinationPMTCS, CombinationPMTSTAT, CombinationPMTMAN, CombinationPMTCHEM:
		return combination, true
	default:
		return "", false
	}
}

func CombinationValues() []string {
	return []string{
		string(CombinationPMTICT),
		string(CombinationPMTCS),
		string(CombinationPMTSTAT),
		string(CombinationPMTMAN),
		string(CombinationPMTCHEM),
	}
}
