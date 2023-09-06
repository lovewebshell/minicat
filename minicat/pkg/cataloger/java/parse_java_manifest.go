package java

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/pkg"
)

const manifestGlob = "/META-INF/MANIFEST.MF"

//

func parseJavaManifest(path string, reader io.Reader) (*pkg.JavaManifest, error) {
	var manifest pkg.JavaManifest
	var sections []map[string]string

	currentSection := func() int {
		return len(sections) - 1
	}

	var lastKey string
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "" {

			lastKey = ""

			continue
		}

		if line[0] == ' ' {

			if lastKey == "" {
				log.Warnf("java manifest %q: found continuation with no previous key: %q", path, line)
				continue
			}

			sections[currentSection()][lastKey] += strings.TrimSpace(line)

			continue
		}

		idx := strings.Index(line, ":")
		if idx == -1 {
			log.Warnf("java manifest %q: unable to split java manifest key-value pairs: %q", path, line)
			continue
		}

		key := strings.TrimSpace(line[0:idx])
		value := strings.TrimSpace(line[idx+1:])

		if key == "" {

			continue
		}

		if lastKey == "" {

			sections = append(sections, make(map[string]string))
		}

		sections[currentSection()][key] = value

		lastKey = key
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("unable to read java manifest: %w", err)
	}

	if len(sections) > 0 {
		manifest.Main = sections[0]
		if len(sections) > 1 {
			manifest.NamedSections = make(map[string]map[string]string)
			for i, s := range sections[1:] {
				name, ok := s["Name"]
				if !ok {

					log.Warnf("java manifest section found without a name: %s", path)
					name = strconv.Itoa(i)
				} else {
					delete(s, "Name")
				}
				manifest.NamedSections[name] = s
			}
		}
	}

	return &manifest, nil
}

func selectName(manifest *pkg.JavaManifest, filenameObj archiveFilename) string {
	var name string
	switch {
	case filenameObj.name != "":
		name = filenameObj.name
	case manifest.Main["Name"] != "":

		name = manifest.Main["Name"]
	case manifest.Main["Bundle-Name"] != "":

		name = manifest.Main["Bundle-Name"]
	case manifest.Main["Short-Name"] != "":

		name = manifest.Main["Short-Name"]
	case manifest.Main["Extension-Name"] != "":

		name = manifest.Main["Extension-Name"]
	case manifest.Main["Implementation-Title"] != "":

		name = manifest.Main["Implementation-Title"]
	}
	return name
}

func selectVersion(manifest *pkg.JavaManifest, filenameObj archiveFilename) string {
	if v := filenameObj.version; v != "" {
		return v
	}

	if manifest == nil {
		return ""
	}

	fieldNames := []string{
		"Implementation-Version",
		"Specification-Version",
		"Plugin-Version",
		"Bundle-Version",
	}

	for _, fieldName := range fieldNames {
		if v := fieldValueFromManifest(*manifest, fieldName); v != "" {
			return v
		}
	}

	return ""
}

func fieldValueFromManifest(manifest pkg.JavaManifest, fieldName string) string {
	if value := manifest.Main[fieldName]; value != "" {
		return value
	}

	for _, section := range manifest.NamedSections {
		if value := section[fieldName]; value != "" {
			return value
		}
	}

	return ""
}
