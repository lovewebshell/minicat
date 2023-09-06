package file

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/internal/log"
)

type ZipFileManifest map[string]os.FileInfo

func NewZipFileManifest(archivePath string) (ZipFileManifest, error) {
	zipReader, err := OpenZip(archivePath)
	manifest := make(ZipFileManifest)
	if err != nil {
		return manifest, fmt.Errorf("unable to open zip archive (%s): %w", archivePath, err)
	}
	defer func() {
		err = zipReader.Close()
		if err != nil {
			log.Warnf("unable to close zip archive (%s): %+v", archivePath, err)
		}
	}()

	for _, file := range zipReader.Reader.File {
		manifest.Add(file.Name, file.FileInfo())
	}
	return manifest, nil
}

func (z ZipFileManifest) Add(entry string, info os.FileInfo) {
	z[entry] = info
}

func (z ZipFileManifest) GlobMatch(patterns ...string) []string {
	uniqueMatches := internal.NewStringSet()

	for _, pattern := range patterns {
		for entry := range z {

			normalizedEntry := normalizeZipEntryName(entry)

			if GlobMatch(pattern, normalizedEntry) {
				uniqueMatches.Add(entry)
			}
		}
	}

	results := uniqueMatches.ToSlice()
	sort.Strings(results)

	return results
}

func normalizeZipEntryName(entry string) string {
	if !strings.HasPrefix(entry, "/") {
		return "/" + entry
	}

	return entry
}
