package alpm

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/vbatts/go-mtree"

	"github.com/lovewebshell/minicat/minicat/file"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/source"
)

var (
	ignoredFiles = map[string]bool{
		"/set":       true,
		".BUILDINFO": true,
		".PKGINFO":   true,
		"":           true,
	}
)

func newAlpmDBPackage(d *pkg.AlpmMetadata) *pkg.Package {
	return &pkg.Package{
		Name:         d.Package,
		Version:      d.Version,
		FoundBy:      catalogerName,
		Type:         "alpm",
		Licenses:     strings.Split(d.License, " "),
		MetadataType: pkg.AlpmMetadataType,
		Metadata:     *d,
	}
}

func newScanner(reader io.Reader) *bufio.Scanner {

	const maxScannerCapacity = 1024 * 1024
	bufScan := make([]byte, maxScannerCapacity)
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
	return scanner
}

func getFileReader(path string, resolver source.FileResolver) (io.Reader, error) {
	locs, err := resolver.FilesByPath(path)
	if err != nil {
		return nil, err
	}

	dbContentReader, err := resolver.FileContentsByLocation(locs[0])
	if err != nil {
		return nil, err
	}
	return dbContentReader, nil
}

func parseDatabase(b *bufio.Scanner) (*pkg.AlpmMetadata, error) {
	var entry pkg.AlpmMetadata
	var err error
	pkgFields := make(map[string]interface{})
	for b.Scan() {
		fields := strings.SplitN(b.Text(), "\n", 2)

		if len(fields) == 1 {
			break
		}

		key := strings.ReplaceAll(fields[0], "%", "")
		key = strings.ToLower(key)
		value := strings.TrimSpace(fields[1])

		switch key {
		case "files":
			var files []map[string]string
			for _, f := range strings.Split(value, "\n") {
				path := fmt.Sprintf("/%s", f)
				if ok := ignoredFiles[path]; !ok {
					files = append(files, map[string]string{"path": path})
				}
			}
			pkgFields[key] = files
		case "backup":
			var backup []map[string]interface{}
			for _, f := range strings.Split(value, "\n") {
				fields := strings.SplitN(f, "\t", 2)
				path := fmt.Sprintf("/%s", fields[0])
				if ok := ignoredFiles[path]; !ok {
					backup = append(backup, map[string]interface{}{
						"path": path,
						"digests": []file.Digest{{
							Algorithm: "md5",
							Value:     fields[1],
						}}})
				}
			}
			pkgFields[key] = backup
		case "reason":
			fallthrough
		case "size":
			pkgFields[key], err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s to integer", value)
			}
		default:
			pkgFields[key] = value
		}
	}
	if err := mapstructure.Decode(pkgFields, &entry); err != nil {
		return nil, fmt.Errorf("unable to parse ALPM metadata: %w", err)
	}
	if entry.Package == "" && len(entry.Files) == 0 && len(entry.Backup) == 0 {
		return nil, nil
	}

	if entry.Backup == nil {
		entry.Backup = make([]pkg.AlpmFileRecord, 0)
	}
	return &entry, nil
}

func parseMtree(r io.Reader) ([]pkg.AlpmFileRecord, error) {
	var err error
	var entries []pkg.AlpmFileRecord

	r, err = gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	specDh, err := mtree.ParseSpec(r)
	if err != nil {
		return nil, err
	}
	for _, f := range specDh.Entries {
		var entry pkg.AlpmFileRecord
		entry.Digests = make([]file.Digest, 0)
		fileFields := make(map[string]interface{})
		if ok := ignoredFiles[f.Name]; ok {
			continue
		}
		path := fmt.Sprintf("/%s", f.Name)
		fileFields["path"] = path
		for _, kv := range f.Keywords {
			kw := string(kv.Keyword())
			switch kw {
			case "time":

				v := strings.Split(kv.Value(), ".")
				i, _ := strconv.ParseInt(v[0], 10, 64)
				tm := time.Unix(i, 0)
				fileFields[kw] = tm
			case "sha256digest":
				entry.Digests = append(entry.Digests, file.Digest{
					Algorithm: "sha256",
					Value:     kv.Value(),
				})
			case "md5digest":
				entry.Digests = append(entry.Digests, file.Digest{
					Algorithm: "md5digest",
					Value:     kv.Value(),
				})
			default:
				fileFields[kw] = kv.Value()
			}
		}
		if err := mapstructure.Decode(fileFields, &entry); err != nil {
			return nil, fmt.Errorf("unable to parse ALPM mtree data: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func parseAlpmDBEntry(reader io.Reader) (*pkg.AlpmMetadata, error) {
	scanner := newScanner(reader)
	metadata, err := parseDatabase(scanner)
	if err != nil {
		return nil, err
	}
	if metadata == nil {
		return nil, nil
	}
	return metadata, nil
}

func parseAlpmDB(resolver source.FileResolver, desc string, reader io.Reader) ([]pkg.Package, error) {
	metadata, err := parseAlpmDBEntry(reader)
	if err != nil {
		return nil, err
	}

	base := filepath.Dir(desc)
	mtree := filepath.Join(base, "mtree")
	r, err := getFileReader(mtree, resolver)
	if err != nil {
		return nil, err
	}
	pkgFiles, err := parseMtree(r)
	if err != nil {
		return nil, err
	}

	metadata.Files = pkgFiles

	files := filepath.Join(base, "files")
	_, err = getFileReader(files, resolver)
	if err != nil {
		return nil, err
	}
	filesMetadata, err := parseAlpmDBEntry(reader)
	if err != nil {
		return nil, err
	} else if filesMetadata != nil {
		metadata.Backup = filesMetadata.Backup
	}

	p := *newAlpmDBPackage(metadata)
	p.SetID()
	return []pkg.Package{p}, nil
}
