package minicat

import (
	"fmt"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/linux"
	"github.com/lovewebshell/minicat/minicat/logger"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger"
	"github.com/lovewebshell/minicat/minicat/source"
)

func CatalogPackages(src *source.Source, cfg cataloger.Config) (*pkg.Catalog, []artifact.Relationship, *linux.Release, error) {
	resolver, err := src.FileResolver(cfg.Search.Scope)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to determine resolver while cataloging packages: %w", err)
	}

	release := linux.IdentifyRelease(resolver)
	if release != nil {
		log.Infof("identified distro: %s", release.String())
	} else {
		log.Info("could not identify distro")
	}

	var catalogers []cataloger.Cataloger
	if len(cfg.Catalogers) > 0 {
		catalogers = cataloger.AllCatalogers(cfg)
	} else {

		switch src.Metadata.Scheme {
		case source.ImageScheme:
			log.Info("cataloging image")
			catalogers = cataloger.ImageCatalogers(cfg)
		case source.FileScheme:
			log.Info("cataloging file")
			catalogers = cataloger.AllCatalogers(cfg)
		case source.DirectoryScheme:
			log.Info("cataloging directory")
			catalogers = cataloger.DirectoryCatalogers(cfg)
		default:
			return nil, nil, nil, fmt.Errorf("unable to determine cataloger set from scheme=%+v", src.Metadata.Scheme)
		}
	}

	catalog, relationships, err := cataloger.Catalog(resolver, release, catalogers...)
	if err != nil {
		return nil, nil, nil, err
	}

	return catalog, relationships, release, nil
}

func SetLogger(logger logger.Logger) {
	log.Log = logger
}
