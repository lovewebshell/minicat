package javascript

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

// a
var _ common.ParserFn = parseYarnLock

var (
	packageNameExp = regexp.MustCompile(`^"?((?:@\w[\w-_.]*\/)?\w[\w-_.]*)@`)

	versionExp = regexp.MustCompile(`^\W+version(?:\W+"|:\W+)([\w-_.]+)"?`)

	packageURLExp = regexp.MustCompile(`^\s+resolved\s+"https://registry\.(?:yarnpkg\.com|npmjs\.org)/(.+?)/-/(?:.+?)-(\d+\..+?)\.tgz`)
)

const (
	noPackage = ""
	noVersion = ""
)

func parseYarnLock(path string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {

	if pathContainsNodeModulesDirectory(path) {
		return nil, nil, nil
	}

	var packages []*pkg.Package
	scanner := bufio.NewScanner(reader)
	parsedPackages := internal.NewStringSet()
	currentPackage := noPackage
	currentVersion := noVersion

	for scanner.Scan() {
		line := scanner.Text()

		if packageName := findPackageName(line); packageName != noPackage {

			if currentPackage != noPackage && currentVersion != noVersion && !parsedPackages.Contains(currentPackage+"@"+currentVersion) {
				packages = append(packages, newYarnLockPackage(currentPackage, currentVersion))
				parsedPackages.Add(currentPackage + "@" + currentVersion)
			}

			currentPackage = packageName
		} else if version := findPackageVersion(line); version != noVersion {
			currentVersion = version
		} else if packageName, version := findPackageAndVersion(line); packageName != noPackage && version != noVersion && !parsedPackages.Contains(packageName+"@"+version) {
			packages = append(packages, newYarnLockPackage(packageName, version))
			parsedPackages.Add(packageName + "@" + version)

			currentPackage = noPackage
			currentVersion = noVersion
		}
	}

	if currentPackage != noPackage && currentVersion != noVersion && !parsedPackages.Contains(currentPackage+"@"+currentVersion) {
		packages = append(packages, newYarnLockPackage(currentPackage, currentVersion))
		parsedPackages.Add(currentPackage + "@" + currentVersion)
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse yarn.lock file: %w", err)
	}

	return packages, nil, nil
}

func findPackageName(line string) string {
	if matches := packageNameExp.FindStringSubmatch(line); len(matches) >= 2 {
		return matches[1]
	}

	return noPackage
}

func findPackageVersion(line string) string {
	if matches := versionExp.FindStringSubmatch(line); len(matches) >= 2 {
		return matches[1]
	}

	return noVersion
}

func findPackageAndVersion(line string) (string, string) {
	if matches := packageURLExp.FindStringSubmatch(line); len(matches) >= 2 {
		return matches[1], matches[2]
	}

	return noPackage, noVersion
}

func newYarnLockPackage(name, version string) *pkg.Package {
	return &pkg.Package{
		Name:     name,
		Version:  version,
		Language: pkg.JavaScript,
		Type:     pkg.NpmPkg,
	}
}
