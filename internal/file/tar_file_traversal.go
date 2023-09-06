package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/mholt/archiver/v3"
)

func ExtractGlobsFromTarToUniqueTempFile(archivePath, dir string, globs ...string) (map[string]Opener, error) {
	results := make(map[string]Opener)

	if len(globs) == 0 {
		return results, nil
	}

	visitor := func(file archiver.File) error {
		defer file.Close()

		if file.FileInfo.IsDir() {
			return nil
		}

		if !matchesAnyGlob(file.Name(), globs...) {
			return nil
		}

		tempfilePrefix := filepath.Base(filepath.Clean(file.Name())) + "-"
		tempFile, err := os.CreateTemp(dir, tempfilePrefix)
		if err != nil {
			return fmt.Errorf("unable to create temp file: %w", err)
		}

		defer tempFile.Close()

		if err := safeCopy(tempFile, file.ReadCloser); err != nil {
			return fmt.Errorf("unable to copy source=%q for tar=%q: %w", file.Name(), archivePath, err)
		}

		results[file.Name()] = Opener{path: tempFile.Name()}

		return nil
	}

	return results, archiver.Walk(archivePath, visitor)
}

func matchesAnyGlob(name string, globs ...string) bool {
	for _, glob := range globs {
		if matches, err := doublestar.PathMatch(glob, name); err == nil && matches {
			return true
		}
	}
	return false
}
