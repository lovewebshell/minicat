package pkg

import "github.com/lovewebshell/minicat/minicat/artifact"

func NewRelationships(catalog *Catalog) []artifact.Relationship {
	return RelationshipsByFileOwnership(catalog)
}
