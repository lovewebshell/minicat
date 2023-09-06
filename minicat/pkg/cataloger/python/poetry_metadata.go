package python

import "github.com/lovewebshell/minicat/minicat/pkg"

type PoetryMetadata struct {
	Packages []PoetryMetadataPackage `toml:"package"`
}

func (m PoetryMetadata) Pkgs() []*pkg.Package {
	pkgs := make([]*pkg.Package, 0)

	for _, p := range m.Packages {
		pkgs = append(pkgs, p.Pkg())
	}

	return pkgs
}
