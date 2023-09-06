package cpe

import (
	"strconv"

	"github.com/scylladb/go-set/strset"
)

type fieldCandidate struct {
	value                       string
	disallowSubSelections       bool
	disallowDelimiterVariations bool
}

type fieldCandidateSet map[fieldCandidate]struct{}

func newFieldCandidateSetFromSets(sets ...fieldCandidateSet) fieldCandidateSet {
	s := newFieldCandidateSet()
	for _, set := range sets {
		s.add(set.list()...)
	}
	return s
}

func newFieldCandidateSet(values ...string) fieldCandidateSet {
	s := make(fieldCandidateSet)
	s.addValue(values...)
	return s
}

func (s fieldCandidateSet) addValue(values ...string) {
	for _, value := range values {

		candidate := fieldCandidate{
			value: cleanCandidateField(value),
		}
		s[candidate] = struct{}{}
	}
}

func (s fieldCandidateSet) add(candidates ...fieldCandidate) {
	for _, candidate := range candidates {
		candidate.value = cleanCandidateField(candidate.value)
		s[candidate] = struct{}{}
	}
}

func (s fieldCandidateSet) removeByValue(values ...string) {
	for _, value := range values {
		s.removeWhere(valueEquals(value))
	}
}

func (s fieldCandidateSet) removeWhere(condition fieldCandidateCondition) {
	for candidate := range s {
		if condition(candidate) {
			delete(s, candidate)
		}
	}
}

func (s fieldCandidateSet) clear() {
	for k := range s {
		delete(s, k)
	}
}

func (s fieldCandidateSet) union(others ...fieldCandidateSet) {
	for _, other := range others {
		s.add(other.list()...)
	}
}

func (s fieldCandidateSet) list() (results []fieldCandidate) {
	for c := range s {
		results = append(results, c)
	}

	return results
}

func (s fieldCandidateSet) values() (results []string) {
	for _, c := range s.list() {
		results = append(results, c.value)
	}

	return results
}

func (s fieldCandidateSet) uniqueValues() []string {
	return strset.New(s.values()...).List()
}

func (s fieldCandidateSet) copy() fieldCandidateSet {
	newSet := newFieldCandidateSet()
	newSet.add(s.list()...)

	return newSet
}

func cleanCandidateField(field string) string {
	cleanedValue, err := strconv.Unquote(field)
	if err != nil {
		return field
	}
	return cleanedValue
}
