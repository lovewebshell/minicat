package python

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

var _ common.ParserFn = parseRequirementsTxt

func parseRequirementsTxt(_ string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {
	packages := make([]*pkg.Package, 0)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		line = trimRequirementsTxtLine(line)

		if line == "" {

			continue
		}

		if strings.HasPrefix(line, "-e") {

			continue
		}

		if !strings.Contains(line, "==") {

			continue
		}

		parts := strings.Split(line, "==")
		name := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])
		packages = append(packages, &pkg.Package{
			Name:     name,
			Version:  version,
			Language: pkg.Python,
			Type:     pkg.PythonPkg,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse python requirements file: %w", err)
	}

	return packages, nil, nil
}

func trimRequirementsTxtLine(line string) string {
	line = strings.TrimSpace(line)
	line = removeTrailingComment(line)
	line = removeEnvironmentMarkers(line)

	return line
}

func removeTrailingComment(line string) string {
	parts := strings.SplitN(line, "#", 2)
	if len(parts) < 2 {

		return line
	}

	return parts[0]
}

func removeEnvironmentMarkers(line string) string {
	parts := strings.SplitN(line, ";", 2)
	if len(parts) < 2 {

		return line
	}

	return parts[0]
}
