package java

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/pkg"
)

//

//

//

//

//

//

//

//

var nameAndVersionPattern = regexp.MustCompile(`(?Ui)^(?P<name>(?:[[:alpha:]][[:word:].]*(?:\.[[:alpha:]][[:word:].]*)*-?)+)(?:-(?P<version>(\d.*|(build\d*.*)|(rc?\d+(?:^[[:alpha:]].*)?))))?$`)
var secondaryVersionPattern = regexp.MustCompile(`(?:[._-](?P<version>(\d.*|(build\d*.*)|(rc?\d+(?:^[[:alpha:]].*)?))))?$`)

type archiveFilename struct {
	raw     string
	name    string
	version string
}

func getSubexp(matches []string, subexpName string, re *regexp.Regexp, raw string) string {
	if len(matches) < 1 {
		log.Warnf("unexpectedly empty matches for archive '%s'", raw)
		return ""
	}

	index := re.SubexpIndex(subexpName)
	if index < 1 {
		log.Warnf("unexpected index of '%s' capture group for Java archive '%s'", subexpName, raw)
		return ""
	}

	if len(matches) < index+1 {
		log.Warnf("no match found for '%s' in '%s'", subexpName, matches[0])
		return ""
	}

	return matches[index]
}

func newJavaArchiveFilename(raw string) archiveFilename {

	cleanedFileName := strings.TrimSuffix(filepath.Base(raw), filepath.Ext(raw))

	matches := nameAndVersionPattern.FindStringSubmatch(cleanedFileName)

	name := getSubexp(matches, "name", nameAndVersionPattern, raw)
	version := getSubexp(matches, "version", nameAndVersionPattern, raw)

	if version == "" {
		matches = secondaryVersionPattern.FindStringSubmatch(name)
		version = getSubexp(matches, "version", secondaryVersionPattern, raw)
		if version != "" {
			name = name[0 : len(name)-len(version)-1]
		}
	}

	return archiveFilename{
		raw:     raw,
		name:    name,
		version: version,
	}
}

func (a archiveFilename) extension() string {
	return strings.TrimPrefix(filepath.Ext(a.raw), ".")
}

func (a archiveFilename) pkgType() pkg.Type {
	switch strings.ToLower(a.extension()) {
	case "jar", "war", "ear", "lpkg", "par", "sar":
		return pkg.JavaPkg
	default:
		return pkg.UnknownPkg
	}
}
