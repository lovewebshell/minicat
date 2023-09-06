package source

import (
	"archive/tar"
	"fmt"
	"io"

	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/filetree"
	"github.com/anchore/stereoscope/pkg/image"
)

var _ FileResolver = (*imageSquashResolver)(nil)

type imageSquashResolver struct {
	img *image.Image
}

func newImageSquashResolver(img *image.Image) (*imageSquashResolver, error) {
	if img.SquashedTree() == nil {
		return nil, fmt.Errorf("the image does not have have a squashed tree")
	}

	return &imageSquashResolver{
		img: img,
	}, nil
}

func (r *imageSquashResolver) HasPath(path string) bool {
	return r.img.SquashedTree().HasPath(file.Path(path))
}

func (r *imageSquashResolver) FilesByPath(paths ...string) ([]Location, error) {
	uniqueFileIDs := file.NewFileReferenceSet()
	uniqueLocations := make([]Location, 0)

	for _, path := range paths {
		tree := r.img.SquashedTree()
		_, ref, err := tree.File(file.Path(path), filetree.FollowBasenameLinks)
		if err != nil {
			return nil, err
		}
		if ref == nil {

			continue
		}

		if ref.RealPath == "/" {
			continue
		} else if r.img.FileCatalog.Exists(*ref) {
			metadata, err := r.img.FileCatalog.Get(*ref)
			if err != nil {
				return nil, fmt.Errorf("unable to get file metadata for path=%q: %w", ref.RealPath, err)
			}
			if metadata.Metadata.IsDir {
				continue
			}
		}

		resolvedRef, err := r.img.ResolveLinkByImageSquash(*ref)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve link from img (ref=%+v): %w", ref, err)
		}

		if resolvedRef != nil && !uniqueFileIDs.Contains(*resolvedRef) {
			uniqueFileIDs.Add(*resolvedRef)
			uniqueLocations = append(uniqueLocations, NewLocationFromImage(path, *resolvedRef, r.img))
		}
	}

	return uniqueLocations, nil
}

func (r *imageSquashResolver) FilesByGlob(patterns ...string) ([]Location, error) {
	uniqueFileIDs := file.NewFileReferenceSet()
	uniqueLocations := make([]Location, 0)

	for _, pattern := range patterns {
		results, err := r.img.SquashedTree().FilesByGlob(pattern, filetree.FollowBasenameLinks)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve files by glob (%s): %w", pattern, err)
		}

		for _, result := range results {

			if result.MatchPath == "/" {
				continue
			}

			if r.img.FileCatalog.Exists(result.Reference) {
				metadata, err := r.img.FileCatalog.Get(result.Reference)
				if err != nil {
					return nil, fmt.Errorf("unable to get file metadata for path=%q: %w", result.MatchPath, err)
				}
				if metadata.Metadata.IsDir {
					continue
				}
			}

			resolvedLocations, err := r.FilesByPath(string(result.MatchPath))
			if err != nil {
				return nil, fmt.Errorf("failed to find files by path (result=%+v): %w", result, err)
			}
			for _, resolvedLocation := range resolvedLocations {
				if !uniqueFileIDs.Contains(resolvedLocation.ref) {
					uniqueFileIDs.Add(resolvedLocation.ref)
					uniqueLocations = append(uniqueLocations, resolvedLocation)
				}
			}
		}
	}

	return uniqueLocations, nil
}

func (r *imageSquashResolver) RelativeFileByPath(_ Location, path string) *Location {
	paths, err := r.FilesByPath(path)
	if err != nil {
		return nil
	}
	if len(paths) == 0 {
		return nil
	}

	return &paths[0]
}

func (r *imageSquashResolver) FileContentsByLocation(location Location) (io.ReadCloser, error) {
	entry, err := r.img.FileCatalog.Get(location.ref)
	if err != nil {
		return nil, fmt.Errorf("unable to get metadata for path=%q from file catalog: %w", location.RealPath, err)
	}

	switch entry.Metadata.TypeFlag {
	case tar.TypeSymlink, tar.TypeLink:

		locations, err := r.FilesByPath(location.RealPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve content location at location=%+v: %w", location, err)
		}

		switch len(locations) {
		case 0:
			return nil, fmt.Errorf("link resolution failed while resolving content location: %+v", location)
		case 1:
			location = locations[0]
		default:
			return nil, fmt.Errorf("link resolution resulted in multiple results while resolving content location: %+v", location)
		}
	}

	return r.img.FileContentsByRef(location.ref)
}

func (r *imageSquashResolver) AllLocations() <-chan Location {
	results := make(chan Location)
	go func() {
		defer close(results)
		for _, ref := range r.img.SquashedTree().AllFiles(file.AllTypes...) {
			results <- NewLocationFromImage(string(ref.RealPath), ref, r.img)
		}
	}()
	return results
}

func (r *imageSquashResolver) FilesByMIMEType(types ...string) ([]Location, error) {
	refs, err := r.img.FilesByMIMETypeFromSquash(types...)
	if err != nil {
		return nil, err
	}

	var locations []Location
	for _, ref := range refs {
		locations = append(locations, NewLocationFromImage(string(ref.RealPath), ref, r.img))
	}

	return locations, nil
}

func (r *imageSquashResolver) FileMetadataByLocation(location Location) (FileMetadata, error) {
	return fileMetadataByLocation(r.img, location)
}
