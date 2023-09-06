package alpm

import (
	"fmt"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/source"
)

const catalogerName = "alpmdb-cataloger"

type Cataloger struct{}

func NewAlpmdbCataloger() *Cataloger {
	return &Cataloger{}
}

func (c *Cataloger) Name() string {
	return catalogerName
}

func (c *Cataloger) Catalog(resolver source.FileResolver) ([]pkg.Package, []artifact.Relationship, error) {
	fileMatches, err := resolver.FilesByGlob(pkg.AlpmDBGlob)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find rpmdb's by glob: %w", err)
	}

	var pkgs []pkg.Package
	for _, location := range fileMatches {
		dbContentReader, err := resolver.FileContentsByLocation(location)
		if err != nil {
			return nil, nil, err
		}

		discoveredPkgs, err := parseAlpmDB(resolver, location.RealPath, dbContentReader)
		internal.CloseAndLogError(dbContentReader, location.VirtualPath)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to catalog package=%+v: %w", location.RealPath, err)
		}
		pkgs = append(pkgs, discoveredPkgs...)
	}
	return pkgs, nil, nil
}
