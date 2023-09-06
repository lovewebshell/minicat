package linux

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/acobaugh/osrelease"
	"github.com/google/go-cmp/cmp"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/source"
)

type parseFunc func(string) (*Release, error)

type parseEntry struct {
	path string
	fn   parseFunc
}

var identityFiles = []parseEntry{
	{

		path: "/etc/os-release",
		fn:   parseOsRelease,
	},
	{

		path: "/usr/lib/os-release",
		fn:   parseOsRelease,
	},
	{

		path: "/etc/system-release-cpe",
		fn:   parseSystemReleaseCPE,
	},
	{

		path: "/etc/redhat-release",
		fn:   parseRedhatRelease,
	},

	{

		path: "/bin/busybox",
		fn:   parseBusyBox,
	},
}

func IdentifyRelease(resolver source.FileResolver) *Release {
	for _, entry := range identityFiles {
		locations, err := resolver.FilesByPath(entry.path)
		if err != nil {
			log.Warnf("unable to get path locations from %s: %+v", entry.path, err)
			continue
		}

		for _, location := range locations {
			contentReader, err := resolver.FileContentsByLocation(location)
			if err != nil {
				log.Debugf("unable to get contents from %s: %s", entry.path, err)
				continue
			}

			content, err := io.ReadAll(contentReader)
			internal.CloseAndLogError(contentReader, location.VirtualPath)
			if err != nil {
				log.Warnf("unable to read %q: %+v", location.RealPath, err)
				break
			}

			release, err := entry.fn(string(content))
			if err != nil {
				log.Warnf("unable to parse %q", location.RealPath)
			}

			if release != nil {
				return release
			}
		}
	}

	return nil
}

func parseOsRelease(contents string) (*Release, error) {
	values, err := osrelease.ReadString(contents)
	if err != nil {
		return nil, fmt.Errorf("unable to read os-release file: %w", err)
	}

	var idLike []string
	for _, s := range strings.Split(values["ID_LIKE"], " ") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		idLike = append(idLike, s)
	}

	r := Release{
		PrettyName:       values["PRETTY_NAME"],
		Name:             values["NAME"],
		ID:               values["ID"],
		IDLike:           idLike,
		Version:          values["VERSION"],
		VersionID:        values["VERSION_ID"],
		VersionCodename:  values["VERSION_CODENAME"],
		BuildID:          values["BUILD_ID"],
		ImageID:          values["IMAGE_ID"],
		ImageVersion:     values["IMAGE_VERSION"],
		Variant:          values["VARIANT"],
		VariantID:        values["VARIANT_ID"],
		HomeURL:          values["HOME_URL"],
		SupportURL:       values["SUPPORT_URL"],
		BugReportURL:     values["BUG_REPORT_URL"],
		PrivacyPolicyURL: values["PRIVACY_POLICY_URL"],
		CPEName:          values["CPE_NAME"],
	}

	if cmp.Equal(r, Release{}) {
		return nil, nil
	}

	return &r, nil
}

var busyboxVersionMatcher = regexp.MustCompile(`BusyBox v[\d.]+`)

func parseBusyBox(contents string) (*Release, error) {
	matches := busyboxVersionMatcher.FindAllString(contents, -1)
	for _, match := range matches {
		parts := strings.Split(match, " ")
		version := strings.ReplaceAll(parts[1], "v", "")

		return simpleRelease(match, "busybox", version, ""), nil
	}
	return nil, nil
}

var systemReleaseCpeMatcher = regexp.MustCompile(`cpe:\/o:(.*?):.*?:(.*?):.*?$`)

func parseSystemReleaseCPE(contents string) (*Release, error) {
	matches := systemReleaseCpeMatcher.FindAllStringSubmatch(contents, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		return simpleRelease(match[1], strings.ToLower(match[1]), match[2], match[0]), nil
	}
	return nil, nil
}

var redhatReleaseMatcher = regexp.MustCompile(`(.*?)\srelease\s(\d\.\d+)`)

func parseRedhatRelease(contents string) (*Release, error) {
	matches := redhatReleaseMatcher.FindAllStringSubmatch(contents, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		return simpleRelease(match[1], strings.ToLower(match[1]), match[2], ""), nil
	}
	return nil, nil
}

func simpleRelease(prettyName, name, version, cpe string) *Release {
	return &Release{
		PrettyName: prettyName,
		Name:       name,
		ID:         name,
		IDLike:     []string{name},
		Version:    version,
		VersionID:  version,
		CPEName:    cpe,
	}
}
