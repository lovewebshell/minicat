package pkg

import "github.com/anchore/packageurl-go"

type Type string

const (
	UnknownPkg  Type = "UnknownPackage"
	ApkPkg      Type = "apk"
	AlpmPkg     Type = "alpm"
	GemPkg      Type = "gem"
	DebPkg      Type = "deb"
	RpmPkg      Type = "rpm"
	NpmPkg      Type = "npm"
	PythonPkg   Type = "python"
	JavaPkg     Type = "java-archive"
	GoModulePkg Type = "go-module"
)

func (t Type) PackageURLType() string {
	switch t {
	case ApkPkg:
		return "alpine"
	case AlpmPkg:
		return "alpm"
	case GemPkg:
		return packageurl.TypeGem
	case DebPkg:
		return "deb"
	case PythonPkg:
		return packageurl.TypePyPi
	case NpmPkg:
		return packageurl.TypeNPM
	case RpmPkg:
		return packageurl.TypeRPM
	case GoModulePkg:
		return packageurl.TypeGolang
	default:
		return ""
	}
}
