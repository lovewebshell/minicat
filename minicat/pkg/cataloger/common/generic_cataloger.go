/*
Package common provides generic utilities used by multiple catalogers.
*/
package common

import (
	"fmt"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/source"
)

type GenericCataloger struct {
	globParsers       map[string]ParserFn
	pathParsers       map[string]ParserFn
	postProcessors    []PostProcessFunc
	upstreamCataloger string
}

type PostProcessFunc func(resolver source.FileResolver, location source.Location, p *pkg.Package) error

func NewGenericCataloger(pathParsers map[string]ParserFn, globParsers map[string]ParserFn, upstreamCataloger string, postProcessors ...PostProcessFunc) *GenericCataloger {
	return &GenericCataloger{
		globParsers:       globParsers,
		pathParsers:       pathParsers,
		postProcessors:    postProcessors,
		upstreamCataloger: upstreamCataloger,
	}
}

func (c *GenericCataloger) Name() string {
	return c.upstreamCataloger
}

func (c *GenericCataloger) Catalog(resolver source.FileResolver) ([]pkg.Package, []artifact.Relationship, error) {
	var packages []pkg.Package
	var relationships []artifact.Relationship

	for location, parser := range c.selectFiles(resolver) {
		contentReader, err := resolver.FileContentsByLocation(location)
		if err != nil {

			return nil, nil, fmt.Errorf("unable to fetch contents at location=%v: %w", location, err)
		}

		discoveredPackages, discoveredRelationships, err := parser(location.RealPath, contentReader)
		internal.CloseAndLogError(contentReader, location.VirtualPath)
		if err != nil {

			log.Warnf("cataloger '%s' failed to parse entries at location=%+v: %+v", c.upstreamCataloger, location, err)
			continue
		}

		pkgsForRemoval := make(map[artifact.ID]struct{})
		var cleanedRelationships []artifact.Relationship
		for _, p := range discoveredPackages {
			p.FoundBy = c.upstreamCataloger
			p.Locations.Add(location)
			p.SetID()

			if !pkg.IsValid(p) {
				pkgsForRemoval[p.ID()] = struct{}{}
				continue
			}

			for _, postProcess := range c.postProcessors {
				err = postProcess(resolver, location, p)
				if err != nil {
					return nil, nil, err
				}
			}

			packages = append(packages, *p)
		}

		cleanedRelationships = removeRelationshipsWithArtifactIDs(pkgsForRemoval, discoveredRelationships)
		relationships = append(relationships, cleanedRelationships...)
	}
	return packages, relationships, nil
}

func removeRelationshipsWithArtifactIDs(artifactsToExclude map[artifact.ID]struct{}, relationships []artifact.Relationship) []artifact.Relationship {
	if len(artifactsToExclude) == 0 || len(relationships) == 0 {

		return relationships
	}

	var cleanedRelationships []artifact.Relationship
	for _, r := range relationships {
		_, removeTo := artifactsToExclude[r.To.ID()]
		_, removaFrom := artifactsToExclude[r.From.ID()]
		if !removeTo && !removaFrom {
			cleanedRelationships = append(cleanedRelationships, r)
		}
	}

	return cleanedRelationships
}

func (c *GenericCataloger) selectFiles(resolver source.FilePathResolver) map[source.Location]ParserFn {
	var parserByLocation = make(map[source.Location]ParserFn)

	for path, parser := range c.pathParsers {
		files, err := resolver.FilesByPath(path)
		if err != nil {
			log.Warnf("cataloger failed to select files by path: %+v", err)
		}
		for _, f := range files {
			parserByLocation[f] = parser
		}
	}

	for globPattern, parser := range c.globParsers {
		fileMatches, err := resolver.FilesByGlob(globPattern)
		if err != nil {
			log.Warnf("failed to find files by glob: %s", globPattern)
		}
		for _, f := range fileMatches {
			parserByLocation[f] = parser
		}
	}

	return parserByLocation
}
