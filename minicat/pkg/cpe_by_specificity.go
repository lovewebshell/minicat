package pkg

import (
	"sort"

	"github.com/facebookincubator/nvdtools/wfn"
)

var _ sort.Interface = (*CPEBySpecificity)(nil)

type CPEBySpecificity []wfn.Attributes

func (c CPEBySpecificity) Len() int { return len(c) }

func (c CPEBySpecificity) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

func (c CPEBySpecificity) Less(i, j int) bool {
	iScore := weightedCountForSpecifiedFields(c[i])
	jScore := weightedCountForSpecifiedFields(c[j])

	if iScore != jScore {
		return iScore > jScore
	}

	if countFieldLength(c[i]) != countFieldLength(c[j]) {
		return countFieldLength(c[i]) > countFieldLength(c[j])
	}

	return c[i].BindToFmtString() < c[j].BindToFmtString()
}

func countFieldLength(cpe wfn.Attributes) int {
	return len(cpe.Part + cpe.Vendor + cpe.Product + cpe.Version + cpe.TargetSW)
}

func weightedCountForSpecifiedFields(cpe wfn.Attributes) int {
	checksForSpecifiedField := []func(cpe wfn.Attributes) (bool, int){
		func(cpe wfn.Attributes) (bool, int) { return cpe.Part != "", 2 },
		func(cpe wfn.Attributes) (bool, int) { return cpe.Vendor != "", 3 },
		func(cpe wfn.Attributes) (bool, int) { return cpe.Product != "", 4 },
		func(cpe wfn.Attributes) (bool, int) { return cpe.Version != "", 1 },
		func(cpe wfn.Attributes) (bool, int) { return cpe.TargetSW != "", 1 },
	}

	weightedCount := 0
	for _, fieldIsSpecified := range checksForSpecifiedField {
		isSpecified, weight := fieldIsSpecified(cpe)
		if isSpecified {
			weightedCount += weight
		}
	}

	return weightedCount
}
