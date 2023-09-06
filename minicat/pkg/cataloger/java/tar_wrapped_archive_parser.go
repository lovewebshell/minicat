package java

import (
	"fmt"
	"io"

	"github.com/lovewebshell/minicat/internal/file"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

var _ common.ParserFn = parseTarWrappedJavaArchive

var genericTarGlobs = []string{
	"**/*.tar",

	"**/*.tar.gz",
	"**/*.tgz",

	"**/*.tar.bz",
	"**/*.tar.bz2",
	"**/*.tbz",
	"**/*.tbz2",

	"**/*.tar.br",
	"**/*.tbr",

	"**/*.tar.lz4",
	"**/*.tlz4",

	"**/*.tar.sz",
	"**/*.tsz",

	"**/*.tar.xz",
	"**/*.txz",

	"**/*.tar.zst",
}

func parseTarWrappedJavaArchive(virtualPath string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {
	contentPath, archivePath, cleanupFn, err := saveArchiveToTmp(virtualPath, reader)

	defer cleanupFn()
	if err != nil {
		return nil, nil, err
	}

	return discoverPkgsFromTar(virtualPath, archivePath, contentPath)
}

func discoverPkgsFromTar(virtualPath, archivePath, contentPath string) ([]*pkg.Package, []artifact.Relationship, error) {
	openers, err := file.ExtractGlobsFromTarToUniqueTempFile(archivePath, contentPath, archiveFormatGlobs...)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to extract files from tar: %w", err)
	}

	return discoverPkgsFromOpeners(virtualPath, openers, nil)
}
