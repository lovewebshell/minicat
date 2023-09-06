package pkg

import (
	"fmt"
	"github.com/anchore/packageurl-go"
	"sort"
	"strconv"

	"github.com/scylladb/go-set/strset"

	"github.com/lovewebshell/minicat/minicat/file"
	"github.com/lovewebshell/minicat/minicat/linux"
)

const RpmDBGlob = "**/var/lib/rpm/{Packages,Packages.db,rpmdb.sqlite}"

const RpmManifestGlob = "**/var/lib/rpmmanifest/container-manifest-2"

var (
	_ FileOwner     = (*RpmMetadata)(nil)
	_ urlIdentifier = (*RpmMetadata)(nil)
)

type RpmMetadata struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Epoch           *int              `json:"epoch"  cyclonedx:"epoch" jsonschema:"nullable"`
	Arch            string            `json:"architecture"`
	Release         string            `json:"release" cyclonedx:"release"`
	SourceRpm       string            `json:"sourceRpm" cyclonedx:"sourceRpm"`
	Size            int               `json:"size" cyclonedx:"size"`
	License         string            `json:"license"`
	Vendor          string            `json:"vendor"`
	ModularityLabel string            `json:"modularityLabel"`
	Files           []RpmdbFileRecord `json:"files"`
}

type RpmdbFileRecord struct {
	Path      string        `json:"path"`
	Mode      RpmdbFileMode `json:"mode"`
	Size      int           `json:"size"`
	Digest    file.Digest   `json:"digest"`
	UserName  string        `json:"userName"`
	GroupName string        `json:"groupName"`
	Flags     string        `json:"flags"`
}

type RpmdbFileMode uint16

func (m RpmMetadata) PackageURL(distro *linux.Release) string {
	var namespace string
	if distro != nil {
		namespace = distro.ID
	}

	qualifiers := map[string]string{
		PURLQualifierArch: m.Arch,
	}

	if m.Epoch != nil {
		qualifiers[PURLQualifierEpoch] = strconv.Itoa(*m.Epoch)
	}

	if m.SourceRpm != "" {
		qualifiers[PURLQualifierUpstream] = m.SourceRpm
	}

	return packageurl.NewPackageURL(
		packageurl.TypeRPM,
		namespace,
		m.Name,

		fmt.Sprintf("%s-%s", m.Version, m.Release),
		purlQualifiers(
			qualifiers,
			distro,
		),
		"",
	).ToString()
}

func (m RpmMetadata) OwnedFiles() (result []string) {
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
