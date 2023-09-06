package source

import (
	"archive/tar"
	"fmt"
	"io"

	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/filetree"
	"github.com/anchore/stereoscope/pkg/image"
	"github.com/lovewebshell/minicat/internal/log"
)

var _ FileResolver = (*allLayersResolver)(nil)

type allLayersResolver struct {
	img    *image.Image
	layers []int
}

func newAllLayersResolver(img *image.Image) (*allLayersResolver, error) {
	if len(img.Layers) == 0 {
		return nil, fmt.Errorf("the image does not contain any layers")
	}

	var layers = make([]int, 0)
	for idx := range img.Layers {
		layers = append(layers, idx)
	}
	return &allLayersResolver{
		img:    img,
		layers: layers,
	}, nil
}

func (r *allLayersResolver) HasPath(path string) bool {
	p := file.Path(path)
	for _, layerIdx := range r.layers {
		tree := r.img.Layers[layerIdx].Tree
		if tree.HasPath(p) {
			return true
		}
	}
	return false
}

func (r *allLayersResolver) fileByRef(ref file.Reference, uniqueFileIDs file.ReferenceSet, layerIdx int) ([]file.Reference, error) {
	uniqueFiles := make([]file.Reference, 0)

	entry, err := r.img.FileCatalog.Get(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch metadata (ref=%+v): %w", ref, err)
	}

	if entry.Metadata.TypeFlag == tar.TypeLink || entry.Metadata.TypeFlag == tar.TypeSymlink {

		for _, subLayerIdx := range r.layers[layerIdx:] {
			resolvedRef, err := r.img.ResolveLinkByLayerSquash(ref, subLayerIdx)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve link from layer (layer=%d ref=%+v): %w", subLayerIdx, ref, err)
			}
			if resolvedRef != nil && !uniqueFileIDs.Contains(*resolvedRef) {
				uniqueFileIDs.Add(*resolvedRef)
				uniqueFiles = append(uniqueFiles, *resolvedRef)
			}
		}
	} else if !uniqueFileIDs.Contains(ref) {
		uniqueFileIDs.Add(ref)
		uniqueFiles = append(uniqueFiles, ref)
	}

	return uniqueFiles, nil
}

func (r *allLayersResolver) FilesByPath(paths ...string) ([]Location, error) {
	uniqueFileIDs := file.NewFileReferenceSet()
	uniqueLocations := make([]Location, 0)

	for _, path := range paths {
		for idx, layerIdx := range r.layers {
			tree := r.img.Layers[layerIdx].Tree
			_, ref, err := tree.File(file.Path(path), filetree.FollowBasenameLinks, filetree.DoNotFollowDeadBasenameLinks)
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

			results, err := r.fileByRef(*ref, uniqueFileIDs, idx)
			if err != nil {
				return nil, err
			}
			for _, result := range results {
				uniqueLocations = append(uniqueLocations, NewLocationFromImage(path, result, r.img))
			}
		}
	}
	return uniqueLocations, nil
}

func (r *allLayersResolver) FilesByGlob(patterns ...string) ([]Location, error) {
	uniqueFileIDs := file.NewFileReferenceSet()
	uniqueLocations := make([]Location, 0)

	for _, pattern := range patterns {
		for idx, layerIdx := range r.layers {
			results, err := r.img.Layers[layerIdx].Tree.FilesByGlob(pattern, filetree.FollowBasenameLinks, filetree.DoNotFollowDeadBasenameLinks)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve files by glob (%s): %w", pattern, err)
			}

			for _, result := range results {

				if result.RealPath == "/" {
					continue
				} else if r.img.FileCatalog.Exists(result.Reference) {
					metadata, err := r.img.FileCatalog.Get(result.Reference)
					if err != nil {
						return nil, fmt.Errorf("unable to get file metadata for path=%q: %w", result.MatchPath, err)
					}
					if metadata.Metadata.IsDir {
						continue
					}
				}

				refResults, err := r.fileByRef(result.Reference, uniqueFileIDs, idx)
				if err != nil {
					return nil, err
				}
				for _, refResult := range refResults {
					uniqueLocations = append(uniqueLocations, NewLocationFromImage(string(result.MatchPath), refResult, r.img))
				}
			}
		}
	}

	return uniqueLocations, nil
}

func (r *allLayersResolver) RelativeFileByPath(location Location, path string) *Location {
	entry, err := r.img.FileCatalog.Get(location.ref)
	if err != nil {
		return nil
	}

	exists, relativeRef, err := entry.Layer.SquashedTree.File(file.Path(path), filetree.FollowBasenameLinks)
	if err != nil {
		log.Errorf("failed to find path=%q in squash: %+w", path, err)
		return nil
	}
	if !exists && relativeRef == nil {
		return nil
	}

	relativeLocation := NewLocationFromImage(path, *relativeRef, r.img)

	return &relativeLocation
}

func (r *allLayersResolver) FileContentsByLocation(location Location) (io.ReadCloser, error) {
	entry, err := r.img.FileCatalog.Get(location.ref)
	if err != nil {
		return nil, fmt.Errorf("unable to get metadata for path=%q from file catalog: %w", location.RealPath, err)
	}

	switch entry.Metadata.TypeFlag {
	case tar.TypeSymlink, tar.TypeLink:

		newLocation := r.RelativeFileByPath(location, location.VirtualPath)
		if newLocation == nil {

			return nil, fmt.Errorf("no contents for location=%q", location.VirtualPath)
		}
		location = *newLocation
	}

	return r.img.FileContentsByRef(location.ref)
}

func (r *allLayersResolver) FilesByMIMEType(types ...string) ([]Location, error) {
	var locations []Location
	for _, layerIdx := range r.layers {
		layer := r.img.Layers[layerIdx]

		refs, err := layer.FilesByMIMEType(types...)
		if err != nil {
			return nil, err
		}

		for _, ref := range refs {
			locations = append(locations, NewLocationFromImage(string(ref.RealPath), ref, r.img))
		}
	}

	return locations, nil
}

func (r *allLayersResolver) AllLocations() <-chan Location {
	results := make(chan Location)
	go func() {
		defer close(results)
		for _, layerIdx := range r.layers {
			tree := r.img.Layers[layerIdx].Tree
			for _, ref := range tree.AllFiles(file.AllTypes...) {
				results <- NewLocationFromImage(string(ref.RealPath), ref, r.img)
			}
		}
	}()
	return results
}

func (r *allLayersResolver) FileMetadataByLocation(location Location) (FileMetadata, error) {
	return fileMetadataByLocation(r.img, location)
}
