/*
Package cataloger provides the ability to process files from a container image or file system and discover packages
(gems, wheels, jars, rpms, debs, etc). Specifically, this package contains both a catalog function to utilize all
catalogers defined in child packages as well as the interface definition to implement a cataloger.
*/
package cataloger

import (
	"strings"

	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/alpm"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/apkdb"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/deb"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/golang"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/java"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/javascript"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/python"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/rpm"
	"github.com/lovewebshell/minicat/minicat/source"
)

const AllCatalogersPattern = "all"

type Cataloger interface {
	Name() string

	Catalog(resolver source.FileResolver) ([]pkg.Package, []artifact.Relationship, error)
}

func ImageCatalogers(cfg Config) []Cataloger {
	return filterCatalogers([]Cataloger{
		alpm.NewAlpmdbCataloger(),
		python.NewPythonPackageCataloger(),
		javascript.NewJavascriptPackageCataloger(),
		deb.NewDpkgdbCataloger(),
		rpm.NewRpmdbCataloger(),
		java.NewJavaCataloger(cfg.Java()),
		apkdb.NewApkdbCataloger(),
	}, cfg.Catalogers)
}

func DirectoryCatalogers(cfg Config) []Cataloger {
	return filterCatalogers([]Cataloger{
		alpm.NewAlpmdbCataloger(),
		python.NewPythonIndexCataloger(),
		python.NewPythonPackageCataloger(),
		javascript.NewJavascriptLockCataloger(),
		deb.NewDpkgdbCataloger(),
		rpm.NewRpmdbCataloger(),
		rpm.NewFileCataloger(),
		java.NewJavaCataloger(cfg.Java()),
		java.NewJavaPomCataloger(),
		apkdb.NewApkdbCataloger(),
		golang.NewGoModFileCataloger(),
	}, cfg.Catalogers)
}

func AllCatalogers(cfg Config) []Cataloger {
	return filterCatalogers([]Cataloger{
		alpm.NewAlpmdbCataloger(),
		python.NewPythonIndexCataloger(),
		python.NewPythonPackageCataloger(),
		javascript.NewJavascriptLockCataloger(),
		javascript.NewJavascriptPackageCataloger(),
		deb.NewDpkgdbCataloger(),
		rpm.NewRpmdbCataloger(),
		rpm.NewFileCataloger(),
		java.NewJavaCataloger(cfg.Java()),
		java.NewJavaPomCataloger(),
		apkdb.NewApkdbCataloger(),
		golang.NewGoModFileCataloger(),
	}, cfg.Catalogers)
}

func RequestedAllCatalogers(cfg Config) bool {
	for _, enableCatalogerPattern := range cfg.Catalogers {
		if enableCatalogerPattern == AllCatalogersPattern {
			return true
		}
	}
	return false
}

func filterCatalogers(catalogers []Cataloger, enabledCatalogerPatterns []string) []Cataloger {

	if len(enabledCatalogerPatterns) == 0 {
		return catalogers
	}
	for _, enableCatalogerPattern := range enabledCatalogerPatterns {
		if enableCatalogerPattern == AllCatalogersPattern {
			return catalogers
		}
	}
	var keepCatalogers []Cataloger
	for _, cataloger := range catalogers {
		if contains(enabledCatalogerPatterns, cataloger.Name()) {
			keepCatalogers = append(keepCatalogers, cataloger)
			continue
		}
		log.Infof("skipping cataloger %q", cataloger.Name())
	}
	return keepCatalogers
}

func contains(enabledPartial []string, catalogerName string) bool {
	catalogerName = strings.TrimSuffix(catalogerName, "-cataloger")
	for _, partial := range enabledPartial {
		partial = strings.TrimSuffix(partial, "-cataloger")
		if partial == "" {
			continue
		}
		if strings.Contains(catalogerName, partial) {
			return true
		}
	}
	return false
}
