package cpe

import "github.com/lovewebshell/minicat/minicat/pkg"

func candidateVendorsForRuby(p pkg.Package) fieldCandidateSet {
	metadata, ok := p.Metadata.(pkg.GemMetadata)
	if !ok {
		return nil
	}

	vendors := newFieldCandidateSet()

	for _, author := range metadata.Authors {

		vendors.add(fieldCandidate{
			value:                 normalizePersonName(stripEmailSuffix(author)),
			disallowSubSelections: true,
		})
	}
	return vendors
}
