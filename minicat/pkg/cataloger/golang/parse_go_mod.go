package golang

import (
	"fmt"
	"io"
	"sort"

	"golang.org/x/mod/modfile"

	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
)

func parseGoMod(path string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {
	packages := make(map[string]*pkg.Package)

	contents, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read go module: %w", err)
	}

	file, err := modfile.Parse(path, contents, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse go module: %w", err)
	}

	for _, m := range file.Require {
		packages[m.Mod.Path] = &pkg.Package{
			Name:     m.Mod.Path,
			Version:  m.Mod.Version,
			Language: pkg.Go,
			Type:     pkg.GoModulePkg,
		}
	}

	for _, m := range file.Replace {
		packages[m.New.Path] = &pkg.Package{
			Name:     m.New.Path,
			Version:  m.New.Version,
			Language: pkg.Go,
			Type:     pkg.GoModulePkg,
		}
	}

	for _, m := range file.Exclude {
		delete(packages, m.Mod.Path)
	}

	pkgsSlice := make([]*pkg.Package, len(packages))
	idx := 0
	for _, p := range packages {
		pkgsSlice[idx] = p
		idx++
	}

	sort.SliceStable(pkgsSlice, func(i, j int) bool {
		return pkgsSlice[i].Name < pkgsSlice[j].Name
	})

	return pkgsSlice, nil, nil
}
