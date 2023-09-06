package java

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/lovewebshell/minicat/minicat/pkg"
)

const pomPropertiesGlob = "*pom.properties"

func parsePomProperties(path string, reader io.Reader) (*pkg.PomProperties, error) {
	var props pkg.PomProperties
	propMap := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimLeft(line, " "), "#") {
			continue
		}

		idx := strings.IndexAny(line, "=:")
		if idx == -1 {
			return nil, fmt.Errorf("unable to split pom.properties key-value pairs: %q", line)
		}

		key := strings.TrimSpace(line[0:idx])
		value := strings.TrimSpace(line[idx+1:])
		propMap[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("unable to read pom.properties: %w", err)
	}

	if err := mapstructure.Decode(propMap, &props); err != nil {
		return nil, fmt.Errorf("unable to parse pom.properties: %w", err)
	}

	if props.Extra == nil {
		props.Extra = make(map[string]string)
	}

	props.Path = path

	return &props, nil
}
