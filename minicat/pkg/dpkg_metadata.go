package pkg

import (
	"fmt"
	"github.com/anchore/packageurl-go"
	"sort"

	"github.com/scylladb/go-set/strset"

	"github.com/lovewebshell/minicat/minicat/file"
	"github.com/lovewebshell/minicat/minicat/linux"
)

const DpkgDBGlob = "**/var/lib/dpkg/{status,status.d/**}"

var (
	_ FileOwner     = (*DpkgMetadata)(nil)
	_ urlIdentifier = (*DpkgMetadata)(nil)
)

type DpkgMetadata struct {
	Package       string           `mapstructure:"Package" json:"package"`
	Source        string           `mapstructure:"Source" json:"source" cyclonedx:"source"`
	Version       string           `mapstructure:"Version" json:"version"`
	SourceVersion string           `mapstructure:"SourceVersion" json:"sourceVersion" cyclonedx:"sourceVersion"`
	Architecture  string           `mapstructure:"Architecture" json:"architecture"`
	Maintainer    string           `mapstructure:"Maintainer" json:"maintainer"`
	InstalledSize int              `mapstructure:"InstalledSize" json:"installedSize" cyclonedx:"installedSize"`
	Description   string           `mapstructure:"Description" hash:"ignore" json:"-"`
	Files         []DpkgFileRecord `json:"files"`
}

type DpkgFileRecord struct {
	Path         string       `json:"path"`
	Digest       *file.Digest `json:"digest,omitempty"`
	IsConfigFile bool         `json:"isConfigFile"`
}

func (m DpkgMetadata) PackageURL(distro *linux.Release) string {
	var namespace string
	if distro != nil {
		namespace = distro.ID
	}

	qualifiers := map[string]string{
		PURLQualifierArch: m.Architecture,
	}

	if m.Source != "" {
		if m.SourceVersion != "" {
			qualifiers[PURLQualifierUpstream] = fmt.Sprintf("%s@%s", m.Source, m.SourceVersion)
		} else {
			qualifiers[PURLQualifierUpstream] = m.Source
		}
	}

	return packageurl.NewPackageURL(

		"deb",
		namespace,
		m.Package,
		m.Version,
		purlQualifiers(
			qualifiers,
			distro,
		),
		"",
	).ToString()
}

func (m DpkgMetadata) OwnedFiles() (result []string) {
	s := strset.New()
	for _, f := range m.Files {
		if f.Path != "" {
			s.Add(f.Path)
		}
	}
	result = s.List()
	sort.Strings(result)
	return
}
