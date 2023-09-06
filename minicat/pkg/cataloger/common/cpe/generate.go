package cpe

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/minicat/pkg"
)

func newCPE(product, vendor, version, targetSW string) *wfn.Attributes {
	cpe := *(wfn.NewAttributesWithAny())
	cpe.Part = "a"
	cpe.Product = product
	cpe.Vendor = vendor
	cpe.Version = version
	cpe.TargetSW = targetSW
	if pkg.ValidateCPEString(pkg.CPEString(cpe)) != nil {
		return nil
	}
	return &cpe
}

func Generate(p pkg.Package) []pkg.CPE {
	vendors := candidateVendors(p)
	products := candidateProducts(p)
	if len(products) == 0 {
		return nil
	}

	keys := internal.NewStringSet()
	cpes := make([]pkg.CPE, 0)
	for _, product := range products {
		for _, vendor := range vendors {

			key := fmt.Sprintf("%s|%s|%s", product, vendor, p.Version)
			if keys.Contains(key) {
				continue
			}
			keys.Add(key)

			if cpe := newCPE(product, vendor, p.Version, wfn.Any); cpe != nil {
				cpes = append(cpes, *cpe)
			}
		}
	}

	cpes = filter(cpes, p, cpeFilters...)

	sort.Sort(pkg.CPEBySpecificity(cpes))

	return cpes
}

func candidateVendors(p pkg.Package) []string {

	vendors := newFieldCandidateSet(candidateProducts(p)...)

	switch p.Language {

	case pkg.Go:

		vendors.clear()

		vendor := candidateVendorForGo(p.Name)
		if vendor != "" {
			vendors.addValue(vendor)
		}
	}

	switch p.Language {
	case pkg.JavaScript:
		vendors.addValue(wfn.Any)
	}

	switch p.MetadataType {
	case pkg.RpmMetadataType:
		vendors.union(candidateVendorsForRPM(p))
	case pkg.GemMetadataType:
		vendors.union(candidateVendorsForRuby(p))
	case pkg.PythonPackageMetadataType:
		vendors.union(candidateVendorsForPython(p))
	case pkg.JavaMetadataType:
		vendors.union(candidateVendorsForJava(p))
	}

	addDelimiterVariations(vendors)

	addAllSubSelections(vendors)

	for _, vendor := range vendors.uniqueValues() {
		vendors.addValue(findAdditionalVendors(defaultCandidateAdditions, p.Type, p.Name, vendor)...)
	}

	return vendors.uniqueValues()
}

func candidateProducts(p pkg.Package) []string {
	products := newFieldCandidateSet(p.Name)

	switch {
	case p.Language == pkg.Python:
		if !strings.HasPrefix(p.Name, "python") {
			products.addValue("python-" + p.Name)
		}
	case p.Language == pkg.Java || p.MetadataType == pkg.JavaMetadataType:
		products.addValue(candidateProductsForJava(p)...)
	case p.Language == pkg.Go:

		products.clear()

		prod := candidateProductForGo(p.Name)
		if prod != "" {
			products.addValue(prod)
		}
	}

	products.removeByValue("")
	products.removeByValue("*")

	addDelimiterVariations(products)

	products.addValue(findAdditionalProducts(defaultCandidateAdditions, p.Type, p.Name)...)

	return products.uniqueValues()
}

func addAllSubSelections(fields fieldCandidateSet) {
	candidatesForVariations := fields.copy()
	candidatesForVariations.removeWhere(subSelectionsDisallowed)

	for _, candidate := range candidatesForVariations.values() {
		fields.addValue(generateSubSelections(candidate)...)
	}
}

func generateSubSelections(field string) (results []string) {
	scanner := bufio.NewScanner(strings.NewReader(field))
	scanner.Split(scanByHyphenOrUnderscore)
	var lastToken uint8
	for scanner.Scan() {
		rawCandidate := scanner.Text()
		if len(rawCandidate) == 0 {
			break
		}

		candidate := strings.TrimFunc(rawCandidate, trimHyphenOrUnderscore)

		if len(candidate) > 0 {
			if len(results) > 0 {
				results = append(results, results[len(results)-1]+string(lastToken)+candidate)
			} else {
				results = append(results, candidate)
			}
		}

		lastToken = rawCandidate[len(rawCandidate)-1]
	}
	return results
}

func trimHyphenOrUnderscore(r rune) bool {
	switch r {
	case '-', '_':
		return true
	}
	return false
}

func scanByHyphenOrUnderscore(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "-_"); i >= 0 {
		return i + 1, data[0 : i+1], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

func addDelimiterVariations(fields fieldCandidateSet) {
	candidatesForVariations := fields.copy()
	candidatesForVariations.removeWhere(delimiterVariationsDisallowed)

	for _, candidate := range candidatesForVariations.list() {
		field := candidate.value
		hasHyphen := strings.Contains(field, "-")
		hasUnderscore := strings.Contains(field, "_")

		if hasHyphen {

			newValue := strings.ReplaceAll(field, "-", "_")
			underscoreCandidate := candidate
			underscoreCandidate.value = newValue
			fields.add(underscoreCandidate)
		}

		if hasUnderscore {

			newValue := strings.ReplaceAll(field, "_", "-")
			hyphenCandidate := candidate
			hyphenCandidate.value = newValue
			fields.add(hyphenCandidate)
		}
	}
}
