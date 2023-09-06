package java

import (
	"fmt"
	"io"

	"github.com/lovewebshell/minicat/internal/file"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

var _ common.ParserFn = parseZipWrappedJavaArchive

var genericZipGlobs = []string{
	"**/*.zip",
}

func parseZipWrappedJavaArchive(virtualPath string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {
	contentPath, archivePath, cleanupFn, err := saveArchiveToTmp(virtualPath, reader)

	defer cleanupFn()
	if err != nil {
		return nil, nil, err
	}

	fileManifest, err := file.NewZipFileManifest(archivePath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read files from java archive: %w", err)
	}

	return discoverPkgsFromZip(virtualPath, archivePath, contentPath, fileManifest, nil)
}
