/*
Package rpm provides a concrete DBCataloger implementation for RPM "Package" DB files
and a FileCataloger for RPM files.
*/
package rpm

import (
	"fmt"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/source"
)

const dbCatalogerName = "rpm-db-cataloger"

type DBCataloger struct{}

func NewRpmdbCataloger() *DBCataloger {
	return &DBCataloger{}
}

func (c *DBCataloger) Name() string {
	return dbCatalogerName
}

func (c *DBCataloger) Catalog(resolver source.FileResolver) ([]pkg.Package, []artifact.Relationship, error) {
	fileMatches, err := resolver.FilesByGlob(pkg.RpmDBGlob)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find rpmdb's by glob: %w", err)
	}

	var pkgs []pkg.Package
	for _, location := range fileMatches {
		dbContentReader, err := resolver.FileContentsByLocation(location)
		if err != nil {
			return nil, nil, err
		}

		discoveredPkgs, err := parseRpmDB(resolver, location, dbContentReader)
		internal.CloseAndLogError(dbContentReader, location.VirtualPath)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to catalog rpmdb package=%+v: %w", location.RealPath, err)
		}

		pkgs = append(pkgs, discoveredPkgs...)
	}

	manifestFileMatches, err := resolver.FilesByGlob(pkg.RpmManifestGlob)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find rpm manifests by glob: %w", err)
	}

	for _, location := range manifestFileMatches {
		reader, err := resolver.FileContentsByLocation(location)
		if err != nil {
			return nil, nil, err
		}

		discoveredPkgs, err := parseRpmManifest(location, reader)
		internal.CloseAndLogError(reader, location.VirtualPath)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to catalog rpm manifest=%+v: %w", location.RealPath, err)
		}

		pkgs = append(pkgs, discoveredPkgs...)
	}

	return pkgs, nil, nil
}
