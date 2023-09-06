package python

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/source"
)

const (
	eggMetadataGlob     = "**/*egg-info/PKG-INFO"
	eggFileMetadataGlob = "**/*.egg-info"
	wheelMetadataGlob   = "**/*dist-info/METADATA"
)

type PackageCataloger struct{}

func NewPythonPackageCataloger() *PackageCataloger {
	return &PackageCataloger{}
}

func (c *PackageCataloger) Name() string {
	return "python-package-cataloger"
}

func (c *PackageCataloger) Catalog(resolver source.FileResolver) ([]pkg.Package, []artifact.Relationship, error) {
	var fileMatches []source.Location

	for _, glob := range []string{eggMetadataGlob, wheelMetadataGlob, eggFileMetadataGlob} {
		matches, err := resolver.FilesByGlob(glob)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find files by glob: %s", glob)
		}
		fileMatches = append(fileMatches, matches...)
	}

	var pkgs []pkg.Package
	for _, location := range fileMatches {
		p, err := c.catalogEggOrWheel(resolver, location)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to catalog python package=%+v: %w", location.RealPath, err)
		}
		if pkg.IsValid(p) {
			pkgs = append(pkgs, *p)
		}
	}
	return pkgs, nil, nil
}

func (c *PackageCataloger) catalogEggOrWheel(resolver source.FileResolver, metadataLocation source.Location) (*pkg.Package, error) {
	metadata, sources, err := c.assembleEggOrWheelMetadata(resolver, metadataLocation)
	if err != nil {
		return nil, err
	}

	if metadata.Name == "Python" {
		return nil, nil
	}

	var licenses []string
	if metadata.License != "" {
		licenses = []string{metadata.License}
	}

	p := &pkg.Package{
		Name:         metadata.Name,
		Version:      metadata.Version,
		FoundBy:      c.Name(),
		Locations:    source.NewLocationSet(sources...),
		Licenses:     licenses,
		Language:     pkg.Python,
		Type:         pkg.PythonPkg,
		MetadataType: pkg.PythonPackageMetadataType,
		Metadata:     *metadata,
	}

	p.SetID()

	return p, nil
}

func (c *PackageCataloger) fetchInstalledFiles(resolver source.FileResolver, metadataLocation source.Location, sitePackagesRootPath string) (files []pkg.PythonFileRecord, sources []source.Location, err error) {

	installedFilesPath := filepath.Join(filepath.Dir(metadataLocation.RealPath), "installed-files.txt")
	installedFilesRef := resolver.RelativeFileByPath(metadataLocation, installedFilesPath)

	if installedFilesRef != nil {
		sources = append(sources, *installedFilesRef)

		installedFilesContents, err := resolver.FileContentsByLocation(*installedFilesRef)
		if err != nil {
			return nil, nil, err
		}
		defer internal.CloseAndLogError(installedFilesContents, installedFilesPath)

		installedFiles, err := parseInstalledFiles(installedFilesContents, metadataLocation.RealPath, sitePackagesRootPath)
		if err != nil {
			log.Warnf("unable to parse installed-files.txt for python package=%+v: %w", metadataLocation.RealPath, err)
			return files, sources, nil
		}

		files = append(files, installedFiles...)
	}
	return files, sources, nil
}

func (c *PackageCataloger) fetchRecordFiles(resolver source.FileResolver, metadataLocation source.Location) (files []pkg.PythonFileRecord, sources []source.Location, err error) {

	recordPath := filepath.Join(filepath.Dir(metadataLocation.RealPath), "RECORD")
	recordRef := resolver.RelativeFileByPath(metadataLocation, recordPath)

	if recordRef != nil {
		sources = append(sources, *recordRef)

		recordContents, err := resolver.FileContentsByLocation(*recordRef)
		if err != nil {
			return nil, nil, err
		}
		defer internal.CloseAndLogError(recordContents, recordPath)

		records, err := parseWheelOrEggRecord(recordContents)
		if err != nil {
			return nil, nil, err
		}

		files = append(files, records...)
	}
	return files, sources, nil
}

func (c *PackageCataloger) fetchTopLevelPackages(resolver source.FileResolver, metadataLocation source.Location) (pkgs []string, sources []source.Location, err error) {

	parentDir := filepath.Dir(metadataLocation.RealPath)
	topLevelPath := filepath.Join(parentDir, "top_level.txt")
	topLevelLocation := resolver.RelativeFileByPath(metadataLocation, topLevelPath)

	if topLevelLocation == nil {
		return nil, nil, nil
	}

	sources = append(sources, *topLevelLocation)

	topLevelContents, err := resolver.FileContentsByLocation(*topLevelLocation)
	if err != nil {
		return nil, nil, err
	}
	defer internal.CloseAndLogError(topLevelContents, topLevelLocation.VirtualPath)

	scanner := bufio.NewScanner(topLevelContents)
	for scanner.Scan() {
		pkgs = append(pkgs, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("could not read python package top_level.txt: %w", err)
	}

	return pkgs, sources, nil
}

func (c *PackageCataloger) fetchDirectURLData(resolver source.FileResolver, metadataLocation source.Location) (d *pkg.PythonDirectURLOriginInfo, sources []source.Location, err error) {
	parentDir := filepath.Dir(metadataLocation.RealPath)
	directURLPath := filepath.Join(parentDir, "direct_url.json")
	directURLLocation := resolver.RelativeFileByPath(metadataLocation, directURLPath)

	if directURLLocation == nil {
		return nil, nil, nil
	}

	sources = append(sources, *directURLLocation)

	directURLContents, err := resolver.FileContentsByLocation(*directURLLocation)
	if err != nil {
		return nil, nil, err
	}
	defer internal.CloseAndLogError(directURLContents, directURLLocation.VirtualPath)

	buffer, err := io.ReadAll(directURLContents)
	if err != nil {
		return nil, nil, err
	}

	var directURLJson pkg.DirectURLOrigin
	if err := json.Unmarshal(buffer, &directURLJson); err != nil {
		return nil, nil, err
	}

	return &pkg.PythonDirectURLOriginInfo{
		URL:      directURLJson.URL,
		CommitID: directURLJson.VCSInfo.CommitID,
		VCS:      directURLJson.VCSInfo.VCS,
	}, sources, nil
}

func (c *PackageCataloger) assembleEggOrWheelMetadata(resolver source.FileResolver, metadataLocation source.Location) (*pkg.PythonPackageMetadata, []source.Location, error) {
	var sources = []source.Location{metadataLocation}

	metadataContents, err := resolver.FileContentsByLocation(metadataLocation)
	if err != nil {
		return nil, nil, err
	}
	defer internal.CloseAndLogError(metadataContents, metadataLocation.VirtualPath)

	metadata, err := parseWheelOrEggMetadata(metadataLocation.RealPath, metadataContents)
	if err != nil {
		return nil, nil, err
	}

	r, s, err := c.fetchRecordFiles(resolver, metadataLocation)
	if err != nil {
		return nil, nil, err
	}
	if len(r) == 0 {
		r, s, err = c.fetchInstalledFiles(resolver, metadataLocation, metadata.SitePackagesRootPath)
		if err != nil {
			return nil, nil, err
		}
	}

	sources = append(sources, s...)
	metadata.Files = r

	p, s, err := c.fetchTopLevelPackages(resolver, metadataLocation)
	if err != nil {
		return nil, nil, err
	}
	sources = append(sources, s...)
	metadata.TopLevelPackages = p

	d, s, err := c.fetchDirectURLData(resolver, metadataLocation)
	if err != nil {
		return nil, nil, err
	}
	sources = append(sources, s...)
	metadata.DirectURLOrigin = d

	return &metadata, sources, nil
}
