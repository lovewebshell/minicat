package file

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lovewebshell/minicat/internal/log"
)

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
)

type errZipSlipDetected struct {
	Prefix   string
	JoinArgs []string
}

func (e *errZipSlipDetected) Error() string {
	return fmt.Sprintf("paths are not allowed to resolve outside of the root prefix (%q). Destination: %q", e.Prefix, e.JoinArgs)
}

type zipTraversalRequest map[string]struct{}

func newZipTraverseRequest(paths ...string) zipTraversalRequest {
	results := make(zipTraversalRequest)
	for _, p := range paths {
		results[p] = struct{}{}
	}
	return results
}

func TraverseFilesInZip(archivePath string, visitor func(*zip.File) error, paths ...string) error {
	request := newZipTraverseRequest(paths...)

	zipReader, err := OpenZip(archivePath)
	if err != nil {
		return fmt.Errorf("unable to open zip archive (%s): %w", archivePath, err)
	}
	defer func() {
		err = zipReader.Close()
		if err != nil {
			log.Errorf("unable to close zip archive (%s): %+v", archivePath, err)
		}
	}()

	for _, file := range zipReader.Reader.File {

		if len(paths) > 0 {
			if _, ok := request[file.Name]; !ok {

				continue
			}
		}

		if err = visitor(file); err != nil {
			return err
		}
	}
	return nil
}

func ExtractFromZipToUniqueTempFile(archivePath, dir string, paths ...string) (map[string]Opener, error) {
	results := make(map[string]Opener)

	if len(paths) == 0 {
		return results, nil
	}

	visitor := func(file *zip.File) error {
		tempfilePrefix := filepath.Base(filepath.Clean(file.Name)) + "-"

		tempFile, err := os.CreateTemp(dir, tempfilePrefix)
		if err != nil {
			return fmt.Errorf("unable to create temp file: %w", err)
		}

		defer tempFile.Close()

		zippedFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("unable to read file=%q from zip=%q: %w", file.Name, archivePath, err)
		}
		defer func() {
			err := zippedFile.Close()
			if err != nil {
				log.Errorf("unable to close source file=%q from zip=%q: %+v", file.Name, archivePath, err)
			}
		}()

		if file.FileInfo().IsDir() {
			return fmt.Errorf("unable to extract directories, only files: %s", file.Name)
		}

		if err := safeCopy(tempFile, zippedFile); err != nil {
			return fmt.Errorf("unable to copy source=%q for zip=%q: %w", file.Name, archivePath, err)
		}

		results[file.Name] = Opener{path: tempFile.Name()}

		return nil
	}

	return results, TraverseFilesInZip(archivePath, visitor, paths...)
}

func ContentsFromZip(archivePath string, paths ...string) (map[string]string, error) {
	results := make(map[string]string)

	if len(paths) == 0 {
		return results, nil
	}

	visitor := func(file *zip.File) error {
		zippedFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("unable to read file=%q from zip=%q: %w", file.Name, archivePath, err)
		}

		if file.FileInfo().IsDir() {
			return fmt.Errorf("unable to extract directories, only files: %s", file.Name)
		}

		var buffer bytes.Buffer
		if err := safeCopy(&buffer, zippedFile); err != nil {
			return fmt.Errorf("unable to copy source=%q for zip=%q: %w", file.Name, archivePath, err)
		}

		results[file.Name] = buffer.String()

		err = zippedFile.Close()
		if err != nil {
			return fmt.Errorf("unable to close source file=%q from zip=%q: %w", file.Name, archivePath, err)
		}
		return nil
	}

	return results, TraverseFilesInZip(archivePath, visitor, paths...)
}

func UnzipToDir(archivePath, targetDir string) error {
	visitor := func(file *zip.File) error {
		joinedPath, err := safeJoin(targetDir, file.Name)
		if err != nil {
			return err
		}

		if err = extractSingleFile(file, joinedPath, archivePath); err != nil {
			return err
		}
		return nil
	}

	return TraverseFilesInZip(archivePath, visitor)
}

func safeJoin(prefix string, dest ...string) (string, error) {
	joinResult := filepath.Join(append([]string{prefix}, dest...)...)
	cleanJoinResult := filepath.Clean(joinResult)
	if !strings.HasPrefix(cleanJoinResult, filepath.Clean(prefix)) {
		return "", &errZipSlipDetected{
			Prefix:   prefix,
			JoinArgs: dest,
		}
	}

	return joinResult, nil
}

func extractSingleFile(file *zip.File, expandedFilePath, archivePath string) error {
	zippedFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("unable to read file=%q from zip=%q: %w", file.Name, archivePath, err)
	}

	if file.FileInfo().IsDir() {
		err = os.MkdirAll(expandedFilePath, file.Mode())
		if err != nil {
			return fmt.Errorf("unable to create dir=%q from zip=%q: %w", expandedFilePath, archivePath, err)
		}
	} else {

		outputFile, err := os.OpenFile(
			expandedFilePath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			file.Mode(),
		)
		if err != nil {
			return fmt.Errorf("unable to create dest file=%q from zip=%q: %w", expandedFilePath, archivePath, err)
		}

		if err := safeCopy(outputFile, zippedFile); err != nil {
			return fmt.Errorf("unable to copy source=%q to dest=%q for zip=%q: %w", file.Name, outputFile.Name(), archivePath, err)
		}

		err = outputFile.Close()
		if err != nil {
			return fmt.Errorf("unable to close dest file=%q from zip=%q: %w", outputFile.Name(), archivePath, err)
		}
	}

	err = zippedFile.Close()
	if err != nil {
		return fmt.Errorf("unable to close source file=%q from zip=%q: %w", file.Name, archivePath, err)
	}
	return nil
}
