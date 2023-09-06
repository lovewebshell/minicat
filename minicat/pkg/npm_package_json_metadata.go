package pkg

import (
	"github.com/anchore/packageurl-go"
	"strings"

	"github.com/lovewebshell/minicat/minicat/linux"
)

var _ urlIdentifier = (*NpmPackageJSONMetadata)(nil)

type NpmPackageJSONMetadata struct {
	Name        string   `mapstructure:"name" json:"name"`
	Version     string   `mapstructure:"version" json:"version"`
	Files       []string `mapstructure:"files" json:"files,omitempty"`
	Author      string   `mapstructure:"author" json:"author"`
	Licenses    []string `mapstructure:"licenses" json:"licenses"`
	Homepage    string   `mapstructure:"homepage" json:"homepage"`
	Description string   `mapstructure:"description" json:"description"`
	URL         string   `mapstructure:"url" json:"url"`
	Private     bool     `mapstructure:"private" json:"private"`
}

func (p NpmPackageJSONMetadata) PackageURL(_ *linux.Release) string {
	var namespace string
	name := p.Name

	fields := strings.SplitN(p.Name, "/", 2)
	if len(fields) > 1 {
		namespace = fields[0]
		name = fields[1]
	}

	return packageurl.NewPackageURL(
		packageurl.TypeNPM,
		namespace,
		name,
		p.Version,
		nil,
		"",
	).ToString()
}
