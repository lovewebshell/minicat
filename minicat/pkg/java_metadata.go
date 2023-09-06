package pkg

import (
	"github.com/lovewebshell/minicat/minicat/file"
	"github.com/lovewebshell/minicat/minicat/linux"
)

var _ urlIdentifier = (*JavaMetadata)(nil)

var jenkinsPluginPomPropertiesGroupIDs = []string{
	"io.jenkins.plugins",
	"org.jenkins.plugins",
	"org.jenkins-ci.plugins",
	"io.jenkins-ci.plugins",
	"com.cloudbees.jenkins.plugins",
}

type JavaMetadata struct {
	VirtualPath    string         `json:"virtualPath" cyclonedx:"virtualPath"`
	Manifest       *JavaManifest  `mapstructure:"Manifest" json:"manifest,omitempty"`
	PomProperties  *PomProperties `mapstructure:"PomProperties" json:"pomProperties,omitempty" cyclonedx:"-"`
	PomProject     *PomProject    `mapstructure:"PomProject" json:"pomProject,omitempty"`
	ArchiveDigests []file.Digest  `hash:"ignore" json:"digest,omitempty"`
	PURL           string         `hash:"ignore" json:"-"`
	Parent         *Package       `hash:"ignore" json:"-"`
}

type PomProperties struct {
	Path       string            `mapstructure:"path" json:"path"`
	Name       string            `mapstructure:"name" json:"name"`
	GroupID    string            `mapstructure:"groupId" json:"groupId" cyclonedx:"groupID"`
	ArtifactID string            `mapstructure:"artifactId" json:"artifactId" cyclonedx:"artifactID"`
	Version    string            `mapstructure:"version" json:"version"`
	Extra      map[string]string `mapstructure:",remain" json:"extraFields"`
}

type PomProject struct {
	Path        string     `json:"path"`
	Parent      *PomParent `json:"parent,omitempty"`
	GroupID     string     `json:"groupId"`
	ArtifactID  string     `json:"artifactId"`
	Version     string     `json:"version"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	URL         string     `json:"url,omitempty"`
}

type PomParent struct {
	GroupID    string `json:"groupId"`
	ArtifactID string `json:"artifactId"`
	Version    string `json:"version"`
}

func (p PomProperties) PkgTypeIndicated() Type {

	return JavaPkg
}

type JavaManifest struct {
	Main          map[string]string            `json:"main,omitempty"`
	NamedSections map[string]map[string]string `json:"namedSections,omitempty"`
}

func (m JavaMetadata) PackageURL(_ *linux.Release) string {
	return m.PURL
}
