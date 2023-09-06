package java

import (
	"crypto"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/lovewebshell/minicat/internal/file"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
	miniCatFile "github.com/lovewebshell/minicat/minicat/file"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

var _ common.ParserFn = parseJavaArchive

var archiveFormatGlobs = []string{
	"**/*.jar",
	"**/*.war",
	"**/*.ear",
	"**/*.par",
	"**/*.sar",
	"**/*.jpi",
	"**/*.hpi",
	"**/*.lpkg",
}

var javaArchiveHashes = []crypto.Hash{
	crypto.SHA1,
}

type archiveParser struct {
	fileManifest file.ZipFileManifest
	virtualPath  string
	archivePath  string
	contentPath  string
	fileInfo     archiveFilename
	detectNested bool
}

func parseJavaArchive(virtualPath string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {
	parser, cleanupFn, err := newJavaArchiveParser(virtualPath, reader, true)

	defer cleanupFn()
	if err != nil {
		return nil, nil, err
	}
	return parser.parse()
}

func uniquePkgKey(p *pkg.Package) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%s|%s", p.Name, p.Version)
}

func newJavaArchiveParser(virtualPath string, reader io.Reader, detectNested bool) (*archiveParser, func(), error) {

	virtualElements := strings.Split(virtualPath, ":")
	currentFilepath := virtualElements[len(virtualElements)-1]

	contentPath, archivePath, cleanupFn, err := saveArchiveToTmp(currentFilepath, reader)
	if err != nil {
		return nil, cleanupFn, fmt.Errorf("unable to process java archive: %w", err)
	}

	fileManifest, err := file.NewZipFileManifest(archivePath)
	if err != nil {
		return nil, cleanupFn, fmt.Errorf("unable to read files from java archive: %w", err)
	}

	return &archiveParser{
		fileManifest: fileManifest,
		virtualPath:  virtualPath,
		archivePath:  archivePath,
		contentPath:  contentPath,
		fileInfo:     newJavaArchiveFilename(currentFilepath),
		detectNested: detectNested,
	}, cleanupFn, nil
}

func (j *archiveParser) parse() ([]*pkg.Package, []artifact.Relationship, error) {
	var pkgs []*pkg.Package
	var relationships []artifact.Relationship

	parentPkg, err := j.discoverMainPackage()

	if err != nil {
		return nil, nil, fmt.Errorf("could not generate package from %s: %w", j.virtualPath, err)
	}
	//先不从maven文件里获取组件，因为存在META-INF\maven文件下虽然包含某个组件，但是实际上却不包含的情况
	//如：nacos-client的META-INF\maven\io.netty包含netty-handler，但是用mvn dependency:tree查看依赖树却没有netty-handler
	//auxPkgs, err := j.discoverPkgsFromAllMavenFiles(parentPkg)
	//if err != nil {
	//	return nil, nil, err
	//}
	//if len(auxPkgs) > 0 {
	//	fmt.Println("xxx")
	//}
	//pkgs = append(pkgs, auxPkgs...)

	if parentPkg != nil {
		properties, err := pomPropertiesByParentPath(j.archivePath, j.virtualPath, j.fileManifest.GlobMatch(pomPropertiesGlob))
		if err != nil {
			return nil, nil, err
		}
		if metadata, ok := parentPkg.Metadata.(pkg.JavaMetadata); ok {

			for _, propertiesObj := range properties {
				if propertiesObj.ArtifactID == parentPkg.Name {
					metadata.PomProperties = &propertiesObj
					parentPkg.Metadata = metadata
					break
				}
			}

		}
	}

	if j.detectNested {

		nestedPkgs, nestedRelationships, err := j.discoverPkgsFromNestedArchives(parentPkg)
		if err != nil {
			return nil, nil, err
		}
		pkgs = append(pkgs, nestedPkgs...)
		relationships = append(relationships, nestedRelationships...)
	}

	if parentPkg != nil {
		pkgs = append([]*pkg.Package{parentPkg}, pkgs...)
	}

	for _, p := range pkgs {
		addPURL(p)
	}

	return pkgs, relationships, nil
}

func (j *archiveParser) discoverMainPackage() (*pkg.Package, error) {

	manifestMatches := j.fileManifest.GlobMatch(manifestGlob)
	if len(manifestMatches) > 1 {
		return nil, fmt.Errorf("found multiple manifests in the jar: %+v", manifestMatches)
	} else if len(manifestMatches) == 0 {

		return nil, nil
	}

	contents, err := file.ContentsFromZip(j.archivePath, manifestMatches...)
	if err != nil {
		return nil, fmt.Errorf("unable to extract java manifests (%s): %w", j.virtualPath, err)
	}

	manifestContents := contents[manifestMatches[0]]
	manifest, err := parseJavaManifest(j.archivePath, strings.NewReader(manifestContents))
	if err != nil {
		log.Warnf("failed to parse java manifest (%s): %+v", j.virtualPath, err)
		return nil, nil
	}

	archiveCloser, err := os.Open(j.archivePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open archive path (%s): %w", j.archivePath, err)
	}
	defer archiveCloser.Close()

	digests, err := miniCatFile.DigestsFromFile(archiveCloser, javaArchiveHashes)
	if err != nil {
		log.Warnf("failed to create digest for file=%q: %+v", j.archivePath, err)
	}

	return &pkg.Package{
		Name:         selectName(manifest, j.fileInfo),
		Version:      selectVersion(manifest, j.fileInfo),
		Language:     pkg.Java,
		Type:         j.fileInfo.pkgType(),
		MetadataType: pkg.JavaMetadataType,
		Metadata: pkg.JavaMetadata{
			VirtualPath:    j.virtualPath,
			Manifest:       manifest,
			ArchiveDigests: digests,
		},
	}, nil
}

func (j *archiveParser) discoverPkgsFromAllMavenFiles(parentPkg *pkg.Package) ([]*pkg.Package, error) {
	if parentPkg == nil {
		return nil, nil
	}

	var pkgs []*pkg.Package

	properties, err := pomPropertiesByParentPath(j.archivePath, j.virtualPath, j.fileManifest.GlobMatch(pomPropertiesGlob))
	if err != nil {
		return nil, err
	}

	projects, err := pomProjectByParentPath(j.archivePath, j.virtualPath, j.fileManifest.GlobMatch(pomXMLGlob))
	if err != nil {
		return nil, err
	}

	for parentPath, propertiesObj := range properties {
		var pomProject *pkg.PomProject
		if proj, exists := projects[parentPath]; exists {
			pomProject = &proj
		}

		pkgFromPom := newPackageFromMavenData(propertiesObj, pomProject, parentPkg, j.virtualPath)
		if pkgFromPom != nil {
			pkgs = append(pkgs, pkgFromPom)
		}
	}

	return pkgs, nil
}

func (j *archiveParser) discoverPkgsFromNestedArchives(parentPkg *pkg.Package) ([]*pkg.Package, []artifact.Relationship, error) {

	return discoverPkgsFromZip(j.virtualPath, j.archivePath, j.contentPath, j.fileManifest, parentPkg)
}

func discoverPkgsFromZip(virtualPath, archivePath, contentPath string, fileManifest file.ZipFileManifest, parentPkg *pkg.Package) ([]*pkg.Package, []artifact.Relationship, error) {

	openers, err := file.ExtractFromZipToUniqueTempFile(archivePath, contentPath, fileManifest.GlobMatch(archiveFormatGlobs...)...)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to extract files from zip: %w", err)
	}

	return discoverPkgsFromOpeners(virtualPath, openers, parentPkg)
}

func discoverPkgsFromOpeners(virtualPath string, openers map[string]file.Opener, parentPkg *pkg.Package) ([]*pkg.Package, []artifact.Relationship, error) {
	var pkgs []*pkg.Package
	var relationships []artifact.Relationship

	for pathWithinArchive, archiveOpener := range openers {
		nestedPkgs, nestedRelationships, err := discoverPkgsFromOpener(virtualPath, pathWithinArchive, archiveOpener)
		if err != nil {
			log.Warnf("unable to discover java packages from opener (%s): %+v", virtualPath, err)
			continue
		}

		for _, p := range nestedPkgs {
			if metadata, ok := p.Metadata.(pkg.JavaMetadata); ok {
				if metadata.Parent == nil {
					metadata.Parent = parentPkg
				}
				p.Metadata = metadata
			}
			pkgs = append(pkgs, p)
		}

		relationships = append(relationships, nestedRelationships...)
	}

	return pkgs, relationships, nil
}

func discoverPkgsFromOpener(virtualPath, pathWithinArchive string, archiveOpener file.Opener) ([]*pkg.Package, []artifact.Relationship, error) {
	archiveReadCloser, err := archiveOpener.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open archived file from tempdir: %w", err)
	}
	defer func() {
		if closeErr := archiveReadCloser.Close(); closeErr != nil {
			log.Warnf("unable to close archived file from tempdir: %+v", closeErr)
		}
	}()

	nestedPath := fmt.Sprintf("%s:%s", virtualPath, pathWithinArchive)
	nestedPkgs, nestedRelationships, err := parseJavaArchive(nestedPath, archiveReadCloser)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to process nested java archive (%s): %w", pathWithinArchive, err)
	}

	return nestedPkgs, nestedRelationships, nil
}

func pomPropertiesByParentPath(archivePath, virtualPath string, extractPaths []string) (map[string]pkg.PomProperties, error) {
	contentsOfMavenPropertiesFiles, err := file.ContentsFromZip(archivePath, extractPaths...)
	if err != nil {
		return nil, fmt.Errorf("unable to extract maven files: %w", err)
	}

	propertiesByParentPath := make(map[string]pkg.PomProperties)
	for filePath, fileContents := range contentsOfMavenPropertiesFiles {
		pomProperties, err := parsePomProperties(filePath, strings.NewReader(fileContents))
		if err != nil {
			log.Warnf("failed to parse pom.properties virtualPath=%q path=%q: %+v", virtualPath, filePath, err)
			continue
		}

		if pomProperties == nil {
			continue
		}

		if pomProperties.Version == "" || pomProperties.ArtifactID == "" {

			continue
		}

		propertiesByParentPath[path.Dir(filePath)] = *pomProperties
	}

	return propertiesByParentPath, nil
}

func pomProjectByParentPath(archivePath, virtualPath string, extractPaths []string) (map[string]pkg.PomProject, error) {
	contentsOfMavenProjectFiles, err := file.ContentsFromZip(archivePath, extractPaths...)
	if err != nil {
		return nil, fmt.Errorf("unable to extract maven files: %w", err)
	}

	projectByParentPath := make(map[string]pkg.PomProject)
	for filePath, fileContents := range contentsOfMavenProjectFiles {
		pomProject, err := parsePomXMLProject(filePath, strings.NewReader(fileContents))
		if err != nil {
			log.Warnf("failed to parse pom.xml virtualPath=%q path=%q: %+v", virtualPath, filePath, err)
			continue
		}

		if pomProject == nil {
			continue
		}

		if pomProject.Version == "" || pomProject.ArtifactID == "" {

			continue
		}

		projectByParentPath[path.Dir(filePath)] = *pomProject
	}
	return projectByParentPath, nil
}

func newPackageFromMavenData(pomProperties pkg.PomProperties, pomProject *pkg.PomProject, parentPkg *pkg.Package, virtualPath string) *pkg.Package {

	vPathSuffix := ""
	if !strings.HasPrefix(pomProperties.ArtifactID, parentPkg.Name) {
		vPathSuffix += ":" + pomProperties.ArtifactID
	}
	virtualPath += vPathSuffix

	p := pkg.Package{
		Name:         pomProperties.ArtifactID,
		Version:      pomProperties.Version,
		Language:     pkg.Java,
		Type:         pomProperties.PkgTypeIndicated(),
		MetadataType: pkg.JavaMetadataType,
		Metadata: pkg.JavaMetadata{
			VirtualPath:   virtualPath,
			PomProperties: &pomProperties,
			PomProject:    pomProject,
			Parent:        parentPkg,
		},
	}

	if packageIdentitiesMatch(p, parentPkg) {
		updateParentPackage(p, parentPkg)
		return nil
	}

	return &p
}

func packageIdentitiesMatch(p pkg.Package, parentPkg *pkg.Package) bool {

	if uniquePkgKey(&p) == uniquePkgKey(parentPkg) {
		return true
	}

	metadata := p.Metadata.(pkg.JavaMetadata)

	if parentPkg.Metadata.(pkg.JavaMetadata).VirtualPath == metadata.VirtualPath {
		return true
	}

	if metadata.PomProperties.ArtifactID != "" && parentPkg.Name == metadata.PomProperties.ArtifactID {
		return true
	}

	return false
}

func updateParentPackage(p pkg.Package, parentPkg *pkg.Package) {

	parentPkg.Name = p.Name
	parentPkg.Version = p.Version

	parentPkg.Type = p.Type

	metadata, ok := p.Metadata.(pkg.JavaMetadata)
	if !ok {
		return
	}
	pomPropertiesCopy := *metadata.PomProperties

	parentMetadata, ok := parentPkg.Metadata.(pkg.JavaMetadata)
	if ok && parentMetadata.PomProperties == nil {
		parentMetadata.PomProperties = &pomPropertiesCopy
		parentPkg.Metadata = parentMetadata
	}
}

func addPURL(p *pkg.Package) {
	purl := packageURL(*p)
	if purl == "" {
		return
	}

	metadata, ok := p.Metadata.(pkg.JavaMetadata)
	if !ok {
		return
	}
	metadata.PURL = purl
	p.Metadata = metadata
}
