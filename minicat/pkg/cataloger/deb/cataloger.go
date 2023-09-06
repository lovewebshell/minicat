/*
Package dpkg provides a concrete Cataloger implementation for Debian package DB status files.
*/
package deb

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/source"
)

const (
	md5sumsExt   = ".md5sums"
	conffilesExt = ".conffiles"
	docsPath     = "/usr/share/doc"
)

type Cataloger struct{}

func NewDpkgdbCataloger() *Cataloger {
	return &Cataloger{}
}

func (c *Cataloger) Name() string {
	return "dpkgdb-cataloger"
}

func (c *Cataloger) Catalog(resolver source.FileResolver) ([]pkg.Package, []artifact.Relationship, error) {
	dbFileMatches, err := resolver.FilesByGlob(pkg.DpkgDBGlob)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find dpkg status files's by glob: %w", err)
	}

	var allPackages []pkg.Package
	for _, dbLocation := range dbFileMatches {
		dbContents, err := resolver.FileContentsByLocation(dbLocation)
		if err != nil {
			return nil, nil, err
		}

		pkgs, err := parseDpkgStatus(dbContents)
		internal.CloseAndLogError(dbContents, dbLocation.VirtualPath)
		if err != nil {
			log.Warnf("dpkg cataloger: unable to catalog package=%+v: %w", dbLocation.RealPath, err)
			continue
		}

		for i := range pkgs {
			p := &pkgs[i]
			p.FoundBy = c.Name()
			p.Locations.Add(dbLocation)

			mergeFileListing(resolver, dbLocation, p)

			addLicenses(resolver, dbLocation, p)

			p.SetID()
		}

		allPackages = append(allPackages, pkgs...)
	}
	return allPackages, nil, nil
}

func addLicenses(resolver source.FileResolver, dbLocation source.Location, p *pkg.Package) {

	copyrightReader, copyrightLocation := fetchCopyrightContents(resolver, dbLocation, p)

	if copyrightReader != nil && copyrightLocation != nil {
		defer internal.CloseAndLogError(copyrightReader, copyrightLocation.VirtualPath)

		p.Licenses = parseLicensesFromCopyright(copyrightReader)

		p.Locations.Add(*copyrightLocation)
	}
}

func mergeFileListing(resolver source.FileResolver, dbLocation source.Location, p *pkg.Package) {
	metadata := p.Metadata.(pkg.DpkgMetadata)

	files, infoLocations := getAdditionalFileListing(resolver, dbLocation, p)
loopNewFiles:
	for _, newFile := range files {
		for _, existingFile := range metadata.Files {
			if existingFile.Path == newFile.Path {

				continue loopNewFiles
			}
		}
		metadata.Files = append(metadata.Files, newFile)
	}

	sort.SliceStable(metadata.Files, func(i, j int) bool {
		return metadata.Files[i].Path < metadata.Files[j].Path
	})

	p.Metadata = metadata

	p.Locations.Add(infoLocations...)
}

func getAdditionalFileListing(resolver source.FileResolver, dbLocation source.Location, p *pkg.Package) ([]pkg.DpkgFileRecord, []source.Location) {

	var files = make([]pkg.DpkgFileRecord, 0)
	var locations []source.Location

	md5Reader, md5Location := fetchMd5Contents(resolver, dbLocation, p)

	if md5Reader != nil && md5Location != nil {
		defer internal.CloseAndLogError(md5Reader, md5Location.VirtualPath)

		files = append(files, parseDpkgMD5Info(md5Reader)...)

		locations = append(locations, *md5Location)
	}

	conffilesReader, conffilesLocation := fetchConffileContents(resolver, dbLocation, p)

	if conffilesReader != nil && conffilesLocation != nil {
		defer internal.CloseAndLogError(conffilesReader, conffilesLocation.VirtualPath)

		files = append(files, parseDpkgConffileInfo(conffilesReader)...)

		locations = append(locations, *conffilesLocation)
	}

	return files, locations
}

func fetchMd5Contents(resolver source.FileResolver, dbLocation source.Location, p *pkg.Package) (io.ReadCloser, *source.Location) {
	var md5Reader io.ReadCloser
	var err error

	parentPath := filepath.Dir(dbLocation.RealPath)

	name := md5Key(p)
	location := resolver.RelativeFileByPath(dbLocation, path.Join(parentPath, "info", name+md5sumsExt))

	if location == nil {

		location = resolver.RelativeFileByPath(dbLocation, path.Join(parentPath, "info", p.Name+md5sumsExt))
	}

	if location != nil {
		md5Reader, err = resolver.FileContentsByLocation(*location)
		if err != nil {
			log.Warnf("failed to fetch deb md5 contents (package=%s): %+v", p.Name, err)
		}
	}

	return md5Reader, location
}

func fetchConffileContents(resolver source.FileResolver, dbLocation source.Location, p *pkg.Package) (io.ReadCloser, *source.Location) {
	var reader io.ReadCloser
	var err error

	parentPath := filepath.Dir(dbLocation.RealPath)

	name := md5Key(p)
	location := resolver.RelativeFileByPath(dbLocation, path.Join(parentPath, "info", name+conffilesExt))

	if location == nil {

		location = resolver.RelativeFileByPath(dbLocation, path.Join(parentPath, "info", p.Name+conffilesExt))
	}

	if location != nil {
		reader, err = resolver.FileContentsByLocation(*location)
		if err != nil {
			log.Warnf("failed to fetch deb conffiles contents (package=%s): %+v", p.Name, err)
		}
	}

	return reader, location
}

func fetchCopyrightContents(resolver source.FileResolver, dbLocation source.Location, p *pkg.Package) (io.ReadCloser, *source.Location) {

	name := p.Name
	copyrightPath := path.Join(docsPath, name, "copyright")
	location := resolver.RelativeFileByPath(dbLocation, copyrightPath)

	if location == nil {
		return nil, nil
	}

	reader, err := resolver.FileContentsByLocation(*location)
	if err != nil {
		log.Warnf("failed to fetch deb copyright contents (package=%s): %w", p.Name, err)
	}

	return reader, location
}

func md5Key(p *pkg.Package) string {
	metadata := p.Metadata.(pkg.DpkgMetadata)

	contentKey := p.Name
	if metadata.Architecture != "" && metadata.Architecture != "all" {
		contentKey = contentKey + ":" + metadata.Architecture
	}
	return contentKey
}
