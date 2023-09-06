package cpe

type fieldCandidateCondition func(fieldCandidate) bool

func subSelectionsDisallowed(c fieldCandidate) bool {
	return c.disallowSubSelections
}

func delimiterVariationsDisallowed(c fieldCandidate) bool {
	return c.disallowDelimiterVariations
}

func valueEquals(v string) fieldCandidateCondition {
	return func(candidate fieldCandidate) bool {
		return candidate.value == v
	}
}
