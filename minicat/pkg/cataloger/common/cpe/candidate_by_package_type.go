package cpe

import (
	"github.com/lovewebshell/minicat/minicat/pkg"
)

type candidateComposite struct {
	pkg.Type
	candidateKey
	candidateAddition
}

var defaultCandidateAdditions = buildCandidateLookup(
	[]candidateComposite{

		{
			pkg.JavaPkg,
			candidateKey{PkgName: "springframework"},
			candidateAddition{AdditionalProducts: []string{"spring_framework", "springsource_spring_framework"}, AdditionalVendors: []string{"pivotal_software", "springsource", "vmware"}},
		},
		{
			pkg.JavaPkg,
			candidateKey{PkgName: "spring-core"},
			candidateAddition{AdditionalProducts: []string{"spring_framework", "springsource_spring_framework"}, AdditionalVendors: []string{"pivotal_software", "springsource", "vmware"}},
		},
		{

			pkg.JavaPkg,
			candidateKey{PkgName: "elasticsearch"},
			candidateAddition{AdditionalVendors: []string{"elastic"}},
		},
		{

			pkg.JavaPkg,
			candidateKey{PkgName: "log4j"},
			candidateAddition{AdditionalVendors: []string{"apache"}},
		},

		{

			pkg.JavaPkg,
			candidateKey{PkgName: "apache-cassandra"},
			candidateAddition{AdditionalProducts: []string{"cassandra"}},
		},
		{

			pkg.JavaPkg,
			candidateKey{PkgName: "handlebars"},
			candidateAddition{AdditionalVendors: []string{"handlebarsjs"}},
		},

		{
			pkg.NpmPkg,
			candidateKey{PkgName: "hapi"},
			candidateAddition{AdditionalProducts: []string{"hapi_server_framework"}},
		},
		{
			pkg.NpmPkg,
			candidateKey{PkgName: "handlebars.js"},
			candidateAddition{AdditionalProducts: []string{"handlebars"}},
		},
		{
			pkg.NpmPkg,
			candidateKey{PkgName: "is-my-json-valid"},
			candidateAddition{AdditionalProducts: []string{"is_my_json_valid"}},
		},
		{
			pkg.NpmPkg,
			candidateKey{PkgName: "mustache"},
			candidateAddition{AdditionalProducts: []string{"mustache.js"}},
		},

		{
			pkg.GemPkg,
			candidateKey{PkgName: "Arabic-Prawn"},
			candidateAddition{AdditionalProducts: []string{"arabic_prawn"}},
		},
		{
			pkg.GemPkg,
			candidateKey{PkgName: "bio-basespace-sdk"},
			candidateAddition{AdditionalProducts: []string{"basespace_ruby_sdk"}},
		},
		{
			pkg.GemPkg,
			candidateKey{PkgName: "cremefraiche"},
			candidateAddition{AdditionalProducts: []string{"creme_fraiche"}},
		},
		{
			pkg.GemPkg,
			candidateKey{PkgName: "html-sanitizer"},
			candidateAddition{AdditionalProducts: []string{"html_sanitizer"}},
		},
		{
			pkg.GemPkg,
			candidateKey{PkgName: "sentry-raven"},
			candidateAddition{AdditionalProducts: []string{"raven-ruby"}},
		},
		{
			pkg.GemPkg,
			candidateKey{PkgName: "RedCloth"},
			candidateAddition{AdditionalProducts: []string{"redcloth_library"}},
		},
		{
			pkg.GemPkg,
			candidateKey{PkgName: "VladTheEnterprising"},
			candidateAddition{AdditionalProducts: []string{"vladtheenterprising"}},
		},
		{
			pkg.GemPkg,
			candidateKey{PkgName: "yajl-ruby"},
			candidateAddition{AdditionalProducts: []string{"yajl-ruby_gem"}},
		},

		{
			pkg.PythonPkg,
			candidateKey{PkgName: "python-rrdtool"},
			candidateAddition{AdditionalProducts: []string{"rrdtool"}},
		},
	})

func buildCandidateLookup(cc []candidateComposite) (ca map[pkg.Type]map[candidateKey]candidateAddition) {
	ca = make(map[pkg.Type]map[candidateKey]candidateAddition)
	for _, c := range cc {
		if _, ok := ca[c.Type]; !ok {
			ca[c.Type] = make(map[candidateKey]candidateAddition)
		}
		ca[c.Type][c.candidateKey] = c.candidateAddition
	}

	return ca
}

type candidateKey struct {
	Vendor  string
	PkgName string
}

type candidateAddition struct {
	AdditionalProducts []string
	AdditionalVendors  []string
}

func findAdditionalVendors(allAdditions map[pkg.Type]map[candidateKey]candidateAddition, ty pkg.Type, pkgName, vendor string) (vendors []string) {
	additions, ok := allAdditions[ty]
	if !ok {
		return nil
	}

	if addition, ok := additions[candidateKey{
		Vendor:  vendor,
		PkgName: pkgName,
	}]; ok {
		vendors = append(vendors, addition.AdditionalVendors...)
	}

	if addition, ok := additions[candidateKey{
		PkgName: pkgName,
	}]; ok {
		vendors = append(vendors, addition.AdditionalVendors...)
	}

	if addition, ok := additions[candidateKey{
		Vendor: vendor,
	}]; ok {
		vendors = append(vendors, addition.AdditionalVendors...)
	}

	return vendors
}

func findAdditionalProducts(allAdditions map[pkg.Type]map[candidateKey]candidateAddition, ty pkg.Type, pkgName string) (products []string) {
	additions, ok := allAdditions[ty]
	if !ok {
		return nil
	}

	if addition, ok := additions[candidateKey{
		PkgName: pkgName,
	}]; ok {
		products = append(products, addition.AdditionalProducts...)
	}

	return products
}
