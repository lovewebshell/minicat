package javascript

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/mitchellh/mapstructure"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

var _ common.ParserFn = parsePackageJSON

type packageJSON struct {
	Version      string            `json:"version"`
	Latest       []string          `json:"latest"`
	Author       author            `json:"author"`
	License      json.RawMessage   `json:"license"`
	Licenses     json.RawMessage   `json:"licenses"`
	Name         string            `json:"name"`
	Homepage     string            `json:"homepage"`
	Description  string            `json:"description"`
	Dependencies map[string]string `json:"dependencies"`
	Repository   repository        `json:"repository"`
	Private      bool              `json:"private"`
}

type author struct {
	Name  string `json:"name" mapstruct:"name"`
	Email string `json:"email" mapstruct:"email"`
	URL   string `json:"url" mapstruct:"url"`
}

type repository struct {
	Type string `json:"type" mapstructure:"type"`
	URL  string `json:"url" mapstructure:"url"`
}

var authorPattern = regexp.MustCompile(`^\s*(?P<name>[^<(]*)(\s+<(?P<email>.*)>)?(\s\((?P<url>.*)\))?\s*$`)

func (a *author) UnmarshalJSON(b []byte) error {
	var authorStr string
	var fields map[string]string
	var auth author

	if err := json.Unmarshal(b, &authorStr); err != nil {

		if err := json.Unmarshal(b, &fields); err != nil {
			return fmt.Errorf("unable to parse package.json author: %w", err)
		}
	} else {

		fields = internal.MatchNamedCaptureGroups(authorPattern, authorStr)
	}

	if err := mapstructure.Decode(fields, &auth); err != nil {
		return fmt.Errorf("unable to decode package.json author: %w", err)
	}

	*a = auth

	return nil
}

func (a *author) AuthorString() string {
	result := a.Name
	if a.Email != "" {
		result += fmt.Sprintf(" <%s>", a.Email)
	}
	if a.URL != "" {
		result += fmt.Sprintf(" (%s)", a.URL)
	}
	return result
}

func (r *repository) UnmarshalJSON(b []byte) error {
	var repositoryStr string
	var fields map[string]string
	var repo repository

	if err := json.Unmarshal(b, &repositoryStr); err != nil {

		if err := json.Unmarshal(b, &fields); err != nil {
			return fmt.Errorf("unable to parse package.json author: %w", err)
		}

		if err := mapstructure.Decode(fields, &repo); err != nil {
			return fmt.Errorf("unable to decode package.json author: %w", err)
		}

		*r = repo
	} else {
		r.URL = repositoryStr
	}

	return nil
}

type license struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func licenseFromJSON(b []byte) (string, error) {

	var licenseString string
	err := json.Unmarshal(b, &licenseString)
	if err == nil {
		return licenseString, nil
	}

	var licenseObject license
	err = json.Unmarshal(b, &licenseObject)
	if err == nil {
		return licenseObject.Type, nil
	}

	return "", errors.New("unable to unmarshal license field as either string or object")
}

func (p packageJSON) licensesFromJSON() ([]string, error) {
	if p.License == nil && p.Licenses == nil {

		return []string{}, nil
	}

	singleLicense, err := licenseFromJSON(p.License)
	if err == nil {
		return []string{singleLicense}, nil
	}

	multiLicense, err := licensesFromJSON(p.Licenses)

	if multiLicense != nil && err == nil {
		mapLicenses := func(licenses []license) []string {
			mappedLicenses := make([]string, len(licenses))
			for i, l := range licenses {
				mappedLicenses[i] = l.Type
			}
			return mappedLicenses
		}

		return mapLicenses(multiLicense), nil
	}

	return nil, err
}

func licensesFromJSON(b []byte) ([]license, error) {
	var licenseObject []license
	err := json.Unmarshal(b, &licenseObject)
	if err == nil {
		return licenseObject, nil
	}

	return nil, errors.New("unmarshal failed")
}

func parsePackageJSON(path string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {
	var packages []*pkg.Package
	dec := json.NewDecoder(reader)

	for {
		var p packageJSON
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, nil, fmt.Errorf("failed to parse package.json file: %w", err)
		}

		if !p.hasNameAndVersionValues() {
			log.Debugf("encountered package.json file without a name and/or version field, ignoring (path=%q)", path)
			return nil, nil, nil
		}

		packages = append(packages, newPackageJSONPackage(p))
	}

	return packages, nil, nil
}

func newPackageJSONPackage(p packageJSON) *pkg.Package {
	licenses, err := p.licensesFromJSON()
	if err != nil {
		log.Warnf("unable to extract licenses from javascript package.json: %+v", err)
	}

	return &pkg.Package{
		Name:         p.Name,
		Version:      p.Version,
		Licenses:     licenses,
		Language:     pkg.JavaScript,
		Type:         pkg.NpmPkg,
		MetadataType: pkg.NpmPackageJSONMetadataType,
		Metadata: pkg.NpmPackageJSONMetadata{
			Name:     p.Name,
			Version:  p.Version,
			Author:   p.Author.AuthorString(),
			Homepage: p.Homepage,
			URL:      p.Repository.URL,
			Licenses: licenses,
			Private:  p.Private,
		},
	}
}

func (p packageJSON) hasNameAndVersionValues() bool {
	return p.Name != "" && p.Version != ""
}

var filepathSeparator = regexp.MustCompile(`[\\/]`)

func pathContainsNodeModulesDirectory(p string) bool {
	for _, subPath := range filepathSeparator.Split(p, -1) {
		if subPath == "node_modules" {
			return true
		}
	}
	return false
}
