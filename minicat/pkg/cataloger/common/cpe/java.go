package cpe

import (
	"strings"

	"github.com/scylladb/go-set/strset"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/minicat/pkg"
)

var (
	forbiddenProductGroupIDFields = strset.New("plugin", "plugins", "client")
	forbiddenVendorGroupIDFields  = strset.New("plugin", "plugins")

	domains = []string{
		"com",
		"org",
		"net",
		"io",
		"be",
	}

	primaryJavaManifestGroupIDFields = []string{
		"Extension-Name",
		"Specification-Vendor",
		"Implementation-Vendor",
		"Bundle-SymbolicName",
		"Implementation-Vendor-Id",
		"Implementation-Title",
		"Bundle-Activator",
	}
	secondaryJavaManifestGroupIDFields = []string{
		"Automatic-Module-Name",
		"Main-Class",
		"Package",
	}
	javaManifestNameFields = []string{
		"Specification-Vendor",
		"Implementation-Vendor",
	}
)

func candidateProductsForJava(p pkg.Package) []string {
	return productsFromArtifactAndGroupIDs(artifactIDFromJavaPackage(p), GroupIDsFromJavaPackage(p))
}

func candidateVendorsForJava(p pkg.Package) fieldCandidateSet {
	gidVendors := vendorsFromGroupIDs(GroupIDsFromJavaPackage(p))
	nameVendors := vendorsFromJavaManifestNames(p)
	return newFieldCandidateSetFromSets(gidVendors, nameVendors)
}

func vendorsFromJavaManifestNames(p pkg.Package) fieldCandidateSet {
	vendors := newFieldCandidateSet()

	metadata, ok := p.Metadata.(pkg.JavaMetadata)
	if !ok {
		return vendors
	}

	if metadata.Manifest == nil {
		return vendors
	}

	for _, name := range javaManifestNameFields {
		if metadata.Manifest.Main != nil {
			if value, exists := metadata.Manifest.Main[name]; exists {
				if !startsWithTopLevelDomain(value) {
					vendors.add(fieldCandidate{
						value:                 normalizePersonName(value),
						disallowSubSelections: true,
					})
				}
			}
		}
		if metadata.Manifest.NamedSections != nil {
			for _, section := range metadata.Manifest.NamedSections {
				if section == nil {
					continue
				}
				if value, exists := section[name]; exists {
					if !startsWithTopLevelDomain(value) {
						vendors.add(fieldCandidate{
							value:                 normalizePersonName(value),
							disallowSubSelections: true,
						})
					}
				}
			}
		}
	}

	return vendors
}

func vendorsFromGroupIDs(groupIDs []string) fieldCandidateSet {
	vendors := newFieldCandidateSet()
	for _, groupID := range groupIDs {
		for i, field := range strings.Split(groupID, ".") {
			field = strings.TrimSpace(field)

			if len(field) == 0 {
				continue
			}

			if forbiddenVendorGroupIDFields.Has(strings.ToLower(field)) {
				continue
			}

			if i == 0 {
				continue
			}

			vendors.addValue(field)
		}
	}

	return vendors
}

func productsFromArtifactAndGroupIDs(artifactID string, groupIDs []string) []string {
	products := strset.New()
	if artifactID != "" {
		products.Add(artifactID)
	}

	for _, groupID := range groupIDs {
		isPlugin := strings.Contains(artifactID, "plugin") || strings.Contains(groupID, "plugin")

		for i, field := range strings.Split(groupID, ".") {
			field = strings.TrimSpace(field)

			if len(field) == 0 {
				continue
			}

			if forbiddenProductGroupIDFields.Has(strings.ToLower(field)) {
				continue
			}

			if i <= 1 {
				continue
			}

			couldBeProjectName := strings.HasPrefix(artifactID, field) || strings.HasSuffix(artifactID, field)
			if artifactID == "" || (couldBeProjectName && !isPlugin) {
				products.Add(field)
			}
		}
	}

	return products.List()
}

func artifactIDFromJavaPackage(p pkg.Package) string {
	metadata, ok := p.Metadata.(pkg.JavaMetadata)
	if !ok {
		return ""
	}

	if metadata.PomProperties == nil {
		return ""
	}

	artifactID := strings.TrimSpace(metadata.PomProperties.ArtifactID)
	if startsWithTopLevelDomain(artifactID) && len(strings.Split(artifactID, ".")) > 1 {

		return ""
	}
	return artifactID
}

func GroupIDsFromJavaPackage(p pkg.Package) (groupIDs []string) {
	metadata, ok := p.Metadata.(pkg.JavaMetadata)
	if !ok {
		return nil
	}

	groupIDs = append(groupIDs, groupIDsFromPomProperties(metadata.PomProperties)...)
	groupIDs = append(groupIDs, groupIDsFromPomProject(metadata.PomProject)...)
	groupIDs = append(groupIDs, groupIDsFromJavaManifest(metadata.Manifest)...)

	return groupIDs
}

func groupIDsFromPomProperties(properties *pkg.PomProperties) (groupIDs []string) {
	if properties == nil {
		return nil
	}

	if startsWithTopLevelDomain(properties.GroupID) {
		groupIDs = append(groupIDs, cleanGroupID(properties.GroupID))
	}

	if startsWithTopLevelDomain(properties.ArtifactID) && len(strings.Split(properties.ArtifactID, ".")) > 1 {

		groupIDs = append(groupIDs, cleanGroupID(properties.ArtifactID))
	}

	return groupIDs
}

func groupIDsFromPomProject(project *pkg.PomProject) (groupIDs []string) {
	if project == nil {
		return nil
	}

	groupIDs = addGroupIDsFromGroupIDsAndArtifactID(project.GroupID, project.ArtifactID)

	if project.Parent == nil {
		return groupIDs
	}

	groupIDs = append(groupIDs, addGroupIDsFromGroupIDsAndArtifactID(project.Parent.GroupID, project.Parent.ArtifactID)...)

	return groupIDs
}

func addGroupIDsFromGroupIDsAndArtifactID(groupID, artifactID string) (groupIDs []string) {
	if startsWithTopLevelDomain(groupID) {
		groupIDs = append(groupIDs, cleanGroupID(groupID))
	}

	if startsWithTopLevelDomain(artifactID) && len(strings.Split(artifactID, ".")) > 1 {

		groupIDs = append(groupIDs, cleanGroupID(artifactID))
	}
	return groupIDs
}

func groupIDsFromJavaManifest(manifest *pkg.JavaManifest) []string {
	if manifest == nil {
		return nil
	}

	groupIDs := getManifestFieldGroupIDs(manifest, primaryJavaManifestGroupIDFields)

	if len(groupIDs) != 0 {
		return groupIDs
	}

	return getManifestFieldGroupIDs(manifest, secondaryJavaManifestGroupIDFields)
}

func getManifestFieldGroupIDs(manifest *pkg.JavaManifest, fields []string) (groupIDs []string) {
	if manifest == nil {
		return nil
	}

	for _, name := range fields {
		if value, exists := manifest.Main[name]; exists {
			if startsWithTopLevelDomain(value) {
				groupIDs = append(groupIDs, cleanGroupID(value))
			}
		}
		for _, section := range manifest.NamedSections {
			if value, exists := section[name]; exists {
				if startsWithTopLevelDomain(value) {
					groupIDs = append(groupIDs, cleanGroupID(value))
				}
			}
		}
	}

	return groupIDs
}

func cleanGroupID(groupID string) string {
	return strings.TrimSpace(removeOSCIDirectives(groupID))
}

func removeOSCIDirectives(groupID string) string {

	return strings.Split(groupID, ";")[0]
}

func startsWithTopLevelDomain(value string) bool {
	return internal.HasAnyOfPrefixes(value, domains...)
}
