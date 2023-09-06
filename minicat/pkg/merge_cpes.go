package pkg

import (
	"sort"
)

func mergeCPEs(a, b []CPE) (result []CPE) {
	aCPEs := make(map[string]CPE)

	for _, aCPE := range a {
		aCPEs[aCPE.BindToFmtString()] = aCPE
		result = append(result, aCPE)
	}

	for _, bCPE := range b {
		if _, exists := aCPEs[bCPE.BindToFmtString()]; !exists {
			result = append(result, bCPE)
		}
	}

	sort.Sort(CPEBySpecificity(result))
	return result
}
