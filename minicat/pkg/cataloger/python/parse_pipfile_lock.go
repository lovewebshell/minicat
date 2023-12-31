package python

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

type PipfileLock struct {
	Meta struct {
		Hash struct {
			Sha256 string `json:"sha256"`
		} `json:"hash"`
		PipfileSpec int `json:"pipfile-spec"`
		Requires    struct {
			PythonVersion string `json:"python_version"`
		} `json:"requires"`
		Sources []struct {
			Name      string `json:"name"`
			URL       string `json:"url"`
			VerifySsl bool   `json:"verify_ssl"`
		} `json:"sources"`
	} `json:"_meta"`
	Default map[string]Dependency `json:"default"`
	Develop map[string]Dependency `json:"develop"`
}

type Dependency struct {
	Version string `json:"version"`
}

var _ common.ParserFn = parsePipfileLock

func parsePipfileLock(_ string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {
	packages := make([]*pkg.Package, 0)
	dec := json.NewDecoder(reader)

	for {
		var lock PipfileLock
		if err := dec.Decode(&lock); err == io.EOF {
			break
		} else if err != nil {
			return nil, nil, fmt.Errorf("failed to parse Pipfile.lock file: %w", err)
		}
		for name, pkgMeta := range lock.Default {
			version := strings.TrimPrefix(pkgMeta.Version, "==")
			packages = append(packages, &pkg.Package{
				Name:     name,
				Version:  version,
				Language: pkg.Python,
				Type:     pkg.PythonPkg,
			})
		}
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].String() < packages[j].String()
	})

	return packages, nil, nil
}
