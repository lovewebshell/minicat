package python

import "github.com/lovewebshell/minicat/minicat/pkg"

type PoetryMetadataPackage struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Category    string `toml:"category"`
	Description string `toml:"description"`
	Optional    bool   `toml:"optional"`
}

func (p PoetryMetadataPackage) Pkg() *pkg.Package {
	return &pkg.Package{
		Name:     p.Name,
		Version:  p.Version,
		Language: pkg.Python,
		Type:     pkg.PythonPkg,
	}
}
