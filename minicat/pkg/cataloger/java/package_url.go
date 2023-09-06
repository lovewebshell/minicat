package java

import (
	"github.com/anchore/packageurl-go"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common/cpe"
)

func packageURL(p pkg.Package) string {
	var groupID = p.Name
	groupIDs := cpe.GroupIDsFromJavaPackage(p)
	if len(groupIDs) > 0 {
		groupID = groupIDs[0]
	}

	pURL := packageurl.NewPackageURL(
		packageurl.TypeMaven,
		groupID,
		p.Name,
		p.Version,
		nil,
		"")
	return pURL.ToString()
}
