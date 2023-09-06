package python

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/lovewebshell/minicat/internal/file"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/pkg"
)

func parseWheelOrEggMetadata(path string, reader io.Reader) (pkg.PythonPackageMetadata, error) {
	fields := make(map[string]string)
	var key string

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimRight(line, "\n")

		if len(line) == 0 {
			if len(fields) > 0 {
				break
			}

			continue
		}

		switch {
		case strings.HasPrefix(line, " "):

			updatedValue, err := handleFieldBodyContinuation(key, line, fields)
			if err != nil {
				return pkg.PythonPackageMetadata{}, err
			}

			fields[key] = updatedValue
		default:

			if i := strings.Index(line, ":"); i > 0 {

				key = strings.ReplaceAll(strings.TrimSpace(line[0:i]), "-", "")
				val := strings.TrimSpace(line[i+1:])

				fields[key] = val
			} else {
				log.Warnf("cannot parse field from path: %q from line: %q", path, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return pkg.PythonPackageMetadata{}, fmt.Errorf("failed to parse python wheel/egg: %w", err)
	}

	var metadata pkg.PythonPackageMetadata
	if err := mapstructure.Decode(fields, &metadata); err != nil {
		return pkg.PythonPackageMetadata{}, fmt.Errorf("unable to parse APK metadata: %w", err)
	}

	metadata.SitePackagesRootPath = determineSitePackagesRootPath(path)

	return metadata, nil
}

func isEggRegularFile(path string) bool {
	return file.GlobMatch(eggFileMetadataGlob, path)
}

func determineSitePackagesRootPath(path string) string {
	if isEggRegularFile(path) {
		return filepath.Clean(filepath.Dir(path))
	}

	return filepath.Clean(filepath.Dir(filepath.Dir(path)))
}

func handleFieldBodyContinuation(key, line string, fields map[string]string) (string, error) {
	if len(key) == 0 {
		return "", fmt.Errorf("no match for continuation: line: '%s'", line)
	}

	val, ok := fields[key]
	if !ok {
		return "", fmt.Errorf("no previous key exists, expecting: %s", key)
	}

	return fmt.Sprintf("%s\n %s", val, strings.TrimSpace(line)), nil
}
