package cpe

import (
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"

	"github.com/lovewebshell/minicat/minicat/pkg"
)

const jenkinsName = "jenkins"

type filterFn func(cpe pkg.CPE, p pkg.Package) bool

var cpeFilters = []filterFn{
	disallowJiraClientServerMismatch,
	disallowNonParseableCPEs,
}

func filter(cpes []pkg.CPE, p pkg.Package, filters ...filterFn) (result []pkg.CPE) {
cpeLoop:
	for _, cpe := range cpes {
		for _, fn := range filters {
			if fn(cpe, p) {
				continue cpeLoop
			}
		}

		result = append(result, cpe)
	}
	return result
}

func disallowNonParseableCPEs(cpe pkg.CPE, _ pkg.Package) bool {
	v := pkg.CPEString(cpe)
	_, err := pkg.NewCPE(v)

	cannotParse := err != nil

	return cannotParse
}

func disallowJiraClientServerMismatch(cpe pkg.CPE, p pkg.Package) bool {

	if cpe.Product == "jira" && strings.Contains(strings.ToLower(p.Name), "client") {
		if cpe.Vendor == wfn.Any || cpe.Vendor == "jira" || cpe.Vendor == "atlassian" {
			return true
		}
	}
	return false
}
