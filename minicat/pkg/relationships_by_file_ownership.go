package pkg

import (
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/scylladb/go-set/strset"

	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
)

const AltRpmDBGlob = "**/rpm/{Packages,Packages.db,rpmdb.sqlite}"

var globsForbiddenFromBeingOwned = []string{
	ApkDBGlob,
	RpmDBGlob,
	AltRpmDBGlob,
	"/usr/share/doc/**/copyright",
}

type ownershipByFilesMetadata struct {
	Files []string `json:"files"`
}

func RelationshipsByFileOwnership(catalog *Catalog) []artifact.Relationship {
	var relationships = findOwnershipByFilesRelationships(catalog)

	var edges []artifact.Relationship
	for parentID, children := range relationships {
		for childID, files := range children {
			fs := files.List()
			sort.Strings(fs)
			edges = append(edges, artifact.Relationship{
				From: catalog.byID[parentID],
				To:   catalog.byID[childID],
				Type: artifact.OwnershipByFileOverlapRelationship,
				Data: ownershipByFilesMetadata{
					Files: fs,
				},
			})
		}
	}

	return edges
}

func findOwnershipByFilesRelationships(catalog *Catalog) map[artifact.ID]map[artifact.ID]*strset.Set {
	var relationships = make(map[artifact.ID]map[artifact.ID]*strset.Set)

	if catalog == nil {
		return relationships
	}

	for _, candidateOwnerPkg := range catalog.Sorted() {
		id := candidateOwnerPkg.ID()
		if candidateOwnerPkg.Metadata == nil {
			continue
		}

		pkgFileOwner, ok := candidateOwnerPkg.Metadata.(FileOwner)
		if !ok {
			continue
		}
		for _, ownedFilePath := range pkgFileOwner.OwnedFiles() {
			if matchesAny(ownedFilePath, globsForbiddenFromBeingOwned) {
				continue
			}

			for _, subPackage := range catalog.PackagesByPath(ownedFilePath) {
				subID := subPackage.ID()
				if subID == id {
					continue
				}
				if _, exists := relationships[id]; !exists {
					relationships[id] = make(map[artifact.ID]*strset.Set)
				}

				if _, exists := relationships[id][subID]; !exists {
					relationships[id][subID] = strset.New()
				}
				relationships[id][subID].Add(ownedFilePath)
			}
		}
	}

	return relationships
}

func matchesAny(s string, globs []string) bool {
	for _, g := range globs {
		matches, err := doublestar.Match(g, s)
		if err != nil {
			log.Errorf("failed to match glob=%q : %+v", g, err)
		}
		if matches {
			return true
		}
	}
	return false
}
