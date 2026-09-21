package student

import "strings"

type Combination string

const (
	CombinationPMTICT Combination = "PMT-ICT"
	CombinationPMTCS  Combination = "PMT-CS"
)

func ParseCombination(value string) (Combination, bool) {
	combination := Combination(strings.ToUpper(strings.TrimSpace(value)))
	switch combination {
	case CombinationPMTICT, CombinationPMTCS:
		return combination, true
	default:
		return "", false
	}
}

func CombinationValues() []string {
	return []string{string(CombinationPMTICT), string(CombinationPMTCS)}
}
