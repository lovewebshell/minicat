package apkdb

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/file"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

var _ common.ParserFn = parseApkDB

func newApkDBPackage(d *pkg.ApkMetadata) *pkg.Package {
	return &pkg.Package{
		Name:         d.Package,
		Version:      d.Version,
		Licenses:     strings.Split(d.License, " "),
		Type:         pkg.ApkPkg,
		MetadataType: pkg.ApkMetadataType,
		Metadata:     *d,
	}
}

func parseApkDB(_ string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {

	const maxScannerCapacity = 1024 * 1024

	bufScan := make([]byte, maxScannerCapacity)
	packages := make([]*pkg.Package, 0)

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(bufScan, maxScannerCapacity)
	onDoubleLF := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		for i := 0; i < len(data); i++ {
			if i > 0 && data[i-1] == '\n' && data[i] == '\n' {
				return i + 1, data[:i-1], nil
			}
		}
		if !atEOF {
			return 0, nil, nil
		}

		return 0, data, bufio.ErrFinalToken
	}

	scanner.Split(onDoubleLF)
	for scanner.Scan() {
		metadata, err := parseApkDBEntry(strings.NewReader(scanner.Text()))
		if err != nil {
			return nil, nil, err
		}
		if metadata != nil {
			packages = append(packages, newApkDBPackage(metadata))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse APK DB file: %w", err)
	}

	return packages, nil, nil
}

//

func parseApkDBEntry(reader io.Reader) (*pkg.ApkMetadata, error) {
	var entry pkg.ApkMetadata
	pkgFields := make(map[string]interface{})
	files := make([]pkg.ApkFileRecord, 0)

	var fileRecord *pkg.ApkFileRecord
	lastFile := "/"

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.SplitN(line, ":", 2)
		if len(fields) != 2 {
			continue
		}

		key := fields[0]
		value := strings.TrimSpace(fields[1])

		switch key {
		case "F":
			currentFile := "/" + value

			newFileRecord := pkg.ApkFileRecord{
				Path: currentFile,
			}
			files = append(files, newFileRecord)
			fileRecord = &files[len(files)-1]

			lastFile = currentFile
			continue
		case "R":
			newFileRecord := pkg.ApkFileRecord{
				Path: path.Join(lastFile, value),
			}
			files = append(files, newFileRecord)
			fileRecord = &files[len(files)-1]
		case "a", "M":
			ownershipFields := strings.Split(value, ":")
			if len(ownershipFields) < 3 {
				log.Warnf("unexpected APK ownership field: %q", value)
				continue
			}
			if fileRecord == nil {
				log.Warnf("ownership field with no parent record: %q", value)
				continue
			}
			fileRecord.OwnerUID = ownershipFields[0]
			fileRecord.OwnerGID = ownershipFields[1]
			fileRecord.Permissions = ownershipFields[2]

		case "Z":
			if fileRecord == nil {
				log.Warnf("checksum field with no parent record: %q", value)
				continue
			}
			fileRecord.Digest = &file.Digest{
				Algorithm: "sha1",
				Value:     value,
			}
		case "I", "S":

			iVal, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse APK int: '%+v'", value)
			}
			pkgFields[key] = iVal
		default:
			pkgFields[key] = value
		}
	}

	if err := mapstructure.Decode(pkgFields, &entry); err != nil {
		return nil, fmt.Errorf("unable to parse APK metadata: %w", err)
	}
	if entry.Package == "" {
		return nil, nil
	}

	entry.Files = files

	return &entry, nil
}
