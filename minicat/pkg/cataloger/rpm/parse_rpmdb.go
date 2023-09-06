package rpm

import (
	"fmt"
	"io"
	"os"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/file"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/source"
)

func parseRpmDB(resolver source.FilePathResolver, dbLocation source.Location, reader io.Reader) ([]pkg.Package, error) {
	f, err := os.CreateTemp("", internal.ApplicationName+"-rpmdb")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp rpmdb file: %w", err)
	}

	defer func() {
		err = os.Remove(f.Name())
		if err != nil {
			log.Errorf("failed to remove temp rpmdb file: %+v", err)
		}
	}()

	_, err = io.Copy(f, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to copy rpmdb contents to temp file: %w", err)
	}

	db, err := rpmdb.Open(f.Name())
	if err != nil {
		return nil, err
	}

	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, err
	}

	var allPkgs []pkg.Package

	for _, entry := range pkgList {
		p := newPkg(resolver, dbLocation, entry)

		if !pkg.IsValid(&p) {
			log.Warnf("ignoring invalid package found in RPM DB: location=%q name=%q version=%q", dbLocation, entry.Name, entry.Version)
			continue
		}

		p.SetID()
		allPkgs = append(allPkgs, p)
	}

	return allPkgs, nil
}

func newPkg(resolver source.FilePathResolver, dbLocation source.Location, entry *rpmdb.PackageInfo) pkg.Package {
	metadata := pkg.RpmMetadata{
		Name:            entry.Name,
		Version:         entry.Version,
		Epoch:           entry.Epoch,
		Arch:            entry.Arch,
		Release:         entry.Release,
		SourceRpm:       entry.SourceRpm,
		Vendor:          entry.Vendor,
		License:         entry.License,
		Size:            entry.Size,
		ModularityLabel: entry.Modularitylabel,
		Files:           extractRpmdbFileRecords(resolver, entry),
	}

	p := pkg.Package{
		Name:         entry.Name,
		Version:      toELVersion(metadata),
		Locations:    source.NewLocationSet(dbLocation),
		FoundBy:      dbCatalogerName,
		Type:         pkg.RpmPkg,
		MetadataType: pkg.RpmMetadataType,
		Metadata:     metadata,
	}

	if entry.License != "" {
		p.Licenses = append(p.Licenses, entry.License)
	}

	p.SetID()
	return p
}

func toELVersion(metadata pkg.RpmMetadata) string {
	if metadata.Epoch != nil {
		return fmt.Sprintf("%d:%s-%s", *metadata.Epoch, metadata.Version, metadata.Release)
	}
	return fmt.Sprintf("%s-%s", metadata.Version, metadata.Release)
}

func extractRpmdbFileRecords(resolver source.FilePathResolver, entry *rpmdb.PackageInfo) []pkg.RpmdbFileRecord {
	var records = make([]pkg.RpmdbFileRecord, 0)

	files, err := entry.InstalledFiles()
	if err != nil {
		log.Warnf("unable to parse listing of installed files for RPM DB entry: %s", err.Error())
		return records
	}

	for _, record := range files {

		if resolver.HasPath(record.Path) {
			records = append(records, pkg.RpmdbFileRecord{
				Path: record.Path,
				Mode: pkg.RpmdbFileMode(record.Mode),
				Size: int(record.Size),
				Digest: file.Digest{
					Value:     record.Digest,
					Algorithm: entry.DigestAlgorithm.String(),
				},
				UserName:  record.Username,
				GroupName: record.Groupname,
				Flags:     record.Flags.String(),
			})
		}
	}
	return records
}
