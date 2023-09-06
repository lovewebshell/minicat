package python

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

var _ common.ParserFn = parseSetup

//

var pinnedDependency = regexp.MustCompile(`['"]\W?(\w+\W?==\W?[\w\.]*)`)

func parseSetup(_ string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {
	packages := make([]*pkg.Package, 0)

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimRight(line, "\n")

		for _, match := range pinnedDependency.FindAllString(line, -1) {
			parts := strings.Split(match, "==")
			if len(parts) != 2 {
				continue
			}
			name := strings.Trim(parts[0], "'\"")
			name = strings.TrimSpace(name)

			version := strings.TrimSpace(parts[len(parts)-1])
			packages = append(packages, &pkg.Package{
				Name:     strings.Trim(name, "'\""),
				Version:  strings.Trim(version, "'\""),
				Language: pkg.Python,
				Type:     pkg.PythonPkg,
			})
		}
	}

	return packages, nil, nil
}
