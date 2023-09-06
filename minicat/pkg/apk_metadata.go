package pkg

import (
	"github.com/anchore/packageurl-go"
	"sort"

	"github.com/scylladb/go-set/strset"

	"github.com/lovewebshell/minicat/minicat/file"
	"github.com/lovewebshell/minicat/minicat/linux"
)

const ApkDBGlob = "**/lib/apk/db/installed"

var (
	_ FileOwner     = (*ApkMetadata)(nil)
	_ urlIdentifier = (*ApkMetadata)(nil)
)

type ApkMetadata struct {
	Package          string          `mapstructure:"P" json:"package"`
	OriginPackage    string          `mapstructure:"o" json:"originPackage" cyclonedx:"originPackage"`
	Maintainer       string          `mapstructure:"m" json:"maintainer"`
	Version          string          `mapstructure:"V" json:"version"`
	License          string          `mapstructure:"L" json:"license"`
	Architecture     string          `mapstructure:"A" json:"architecture"`
	URL              string          `mapstructure:"U" json:"url"`
	Description      string          `mapstructure:"T" json:"description"`
	Size             int             `mapstructure:"S" json:"size" cyclonedx:"size"`
	InstalledSize    int             `mapstructure:"I" json:"installedSize" cyclonedx:"installedSize"`
	PullDependencies string          `mapstructure:"D" json:"pullDependencies" cyclonedx:"pullDependencies"`
	PullChecksum     string          `mapstructure:"C" json:"pullChecksum" cyclonedx:"pullChecksum"`
	GitCommitOfAport string          `mapstructure:"c" json:"gitCommitOfApkPort" cyclonedx:"gitCommitOfApkPort"`
	Files            []ApkFileRecord `json:"files"`
}

type ApkFileRecord struct {
	Path        string       `json:"path"`
	OwnerUID    string       `json:"ownerUid,omitempty"`
	OwnerGID    string       `json:"ownerGid,omitempty"`
	Permissions string       `json:"permissions,omitempty"`
	Digest      *file.Digest `json:"digest,omitempty"`
}

func (m ApkMetadata) PackageURL(distro *linux.Release) string {
	qualifiers := map[string]string{
		PURLQualifierArch: m.Architecture,
	}

	if m.OriginPackage != "" {
		qualifiers[PURLQualifierUpstream] = m.OriginPackage
	}

	return packageurl.NewPackageURL(

		"alpine",
		"",
		m.Package,
		m.Version,
		purlQualifiers(
			qualifiers,
			distro,
		),
		"",
	).ToString()
}

func (m ApkMetadata) OwnedFiles() (result []string) {
	s := strset.New()
	for _, f := range m.Files {
		if f.Path != "" {
			s.Add(f.Path)
		}
	}
	result = s.List()
	sort.Strings(result)
	return result
}
