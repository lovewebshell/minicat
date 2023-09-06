package artifact

const (
	OwnershipByFileOverlapRelationship RelationshipType = "ownership-by-file-overlap"

	ContainsRelationship RelationshipType = "contains"

	RuntimeDependencyOfRelationship RelationshipType = "runtime-dependency-of"

	DevDependencyOfRelationship RelationshipType = "dev-dependency-of"

	BuildDependencyOfRelationship RelationshipType = "build-dependency-of"

	DependencyOfRelationship RelationshipType = "dependency-of"
)

type RelationshipType string

type Relationship struct {
	From Identifiable
	To   Identifiable
	Type RelationshipType
	Data interface{}
}
