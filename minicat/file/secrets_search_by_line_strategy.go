package file

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/minicat/source"
)

func catalogLocationByLine(resolver source.FileResolver, location source.Location, patterns map[string]*regexp.Regexp) ([]SearchResult, error) {
	readCloser, err := resolver.FileContentsByLocation(location)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch reader for location=%q : %w", location, err)
	}
	defer internal.CloseAndLogError(readCloser, location.VirtualPath)

	var scanner = bufio.NewReader(readCloser)
	var position int64
	var allSecrets []SearchResult
	var lineNo int64
	var readErr error
	for !errors.Is(readErr, io.EOF) {
		lineNo++
		var line []byte

		line, readErr = scanner.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}

		lineSecrets, err := searchForSecretsWithinLine(resolver, location, patterns, line, lineNo, position)
		if err != nil {
			return nil, err
		}
		position += int64(len(line))
		allSecrets = append(allSecrets, lineSecrets...)
	}

	return allSecrets, nil
}

func searchForSecretsWithinLine(resolver source.FileResolver, location source.Location, patterns map[string]*regexp.Regexp, line []byte, lineNo int64, position int64) ([]SearchResult, error) {
	var secrets []SearchResult
	for name, pattern := range patterns {
		matches := pattern.FindAllIndex(line, -1)
		for i, match := range matches {
			if i%2 == 1 {

				continue
			}

			lineOffset := int64(match[0])
			seekLocation := position + lineOffset
			reader, err := readerAtPosition(resolver, location, seekLocation)
			if err != nil {
				return nil, err
			}

			secret := extractSecretFromPosition(reader, name, pattern, lineNo, lineOffset, seekLocation)
			if secret != nil {
				secrets = append(secrets, *secret)
			}
			internal.CloseAndLogError(reader, location.VirtualPath)
		}
	}

	return secrets, nil
}

func readerAtPosition(resolver source.FileResolver, location source.Location, seekPosition int64) (io.ReadCloser, error) {
	readCloser, err := resolver.FileContentsByLocation(location)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch reader for location=%q : %w", location, err)
	}
	if seekPosition > 0 {
		n, err := io.CopyN(io.Discard, readCloser, seekPosition)
		if err != nil {
			return nil, fmt.Errorf("unable to read contents for location=%q while searching for secrets: %w", location, err)
		}
		if n != seekPosition {
			return nil, fmt.Errorf("unexpected seek location for location=%q while searching for secrets: %d != %d", location, n, seekPosition)
		}
	}
	return readCloser, nil
}

func extractSecretFromPosition(readCloser io.ReadCloser, name string, pattern *regexp.Regexp, lineNo, lineOffset, seekPosition int64) *SearchResult {
	reader := &newlineCounter{RuneReader: bufio.NewReader(readCloser)}
	positions := pattern.FindReaderSubmatchIndex(reader)
	if len(positions) == 0 {

		return nil
	}

	index := pattern.SubexpIndex("value")
	var indexOffset int
	if index != -1 {

		indexOffset = index * 2
	}

	start, stop := int64(positions[indexOffset]), int64(positions[indexOffset+1])

	if start < 0 || stop < 0 {

		return nil
	}

	var lineNoOfSecret = lineNo + int64(reader.newlinesBefore(start))

	var lineOffsetOfSecret = start - reader.newlinePositionBefore(start)
	if lineNoOfSecret == lineNo {

		lineOffsetOfSecret += lineOffset
	}

	return &SearchResult{
		Classification: name,
		SeekPosition:   start + seekPosition,
		Length:         stop - start,
		LineNumber:     lineNoOfSecret,
		LineOffset:     lineOffsetOfSecret,
	}
}
