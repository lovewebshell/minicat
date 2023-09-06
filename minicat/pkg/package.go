/*
Package pkg provides the data structures for a package, a package catalog, package types, and domain-specific metadata.
*/
package pkg

import (
	"fmt"

	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/source"
)

type Package struct {
	id           artifact.ID `hash:"ignore"`
	Name         string
	Version      string
	FoundBy      string `cyclonedx:"foundBy"`
	Locations    source.LocationSet
	Licenses     []string
	Language     Language     `cyclonedx:"language"`
	Type         Type         `cyclonedx:"type"`
	CPEs         []CPE        `hash:"ignore"`
	PURL         string       `hash:"ignore"`
	MetadataType MetadataType `cyclonedx:"metadataType"`
	Metadata     interface{}
	GroupName    string
}

func (p *Package) OverrideID(id artifact.ID) {
	p.id = id
}

func (p *Package) SetID() {
	id, err := artifact.IDByHash(p)
	if err != nil {

		log.Warnf("unable to get fingerprint of package=%s@%s: %+v", p.Name, p.Version, err)
		return
	}
	p.id = id
}

func (p Package) ID() artifact.ID {
	return p.id
}

func (p Package) String() string {
	return fmt.Sprintf("Pkg(name=%q version=%q type=%q id=%q)", p.Name, p.Version, p.Type, p.id)
}

func (p *Package) merge(other Package) error {
	if p.id != other.id {
		return fmt.Errorf("cannot merge packages with different IDs: %q vs %q", p.id, other.id)
	}
	if p.PURL != other.PURL {
		log.Warnf("merging packages have with different pURLs: %q=%q vs %q=%q", p.id, p.PURL, other.id, other.PURL)
	}

	p.Locations.Add(other.Locations.ToSlice()...)

	p.CPEs = mergeCPEs(p.CPEs, other.CPEs)

	if p.PURL == "" {
		p.PURL = other.PURL
	}
	return nil
}

func IsValid(p *Package) bool {
	return p != nil && p.Name != ""
}
