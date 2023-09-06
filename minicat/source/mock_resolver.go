package source

import (
	"fmt"
	"io"
	"os"

	"github.com/bmatcuk/doublestar/v4"
)

var _ FileResolver = (*MockResolver)(nil)

type MockResolver struct {
	locations     []Location
	metadata      map[Location]FileMetadata
	mimeTypeIndex map[string][]Location
}

func NewMockResolverForPaths(paths ...string) *MockResolver {
	var locations []Location
	for _, p := range paths {
		locations = append(locations, NewLocation(p))
	}

	return &MockResolver{
		locations: locations,
		metadata:  make(map[Location]FileMetadata),
	}
}

func NewMockResolverForPathsWithMetadata(metadata map[Location]FileMetadata) *MockResolver {
	var locations []Location
	var mimeTypeIndex = make(map[string][]Location)
	for l, m := range metadata {
		locations = append(locations, l)
		mimeTypeIndex[m.MIMEType] = append(mimeTypeIndex[m.MIMEType], l)
	}

	return &MockResolver{
		locations:     locations,
		metadata:      metadata,
		mimeTypeIndex: mimeTypeIndex,
	}
}

func (r MockResolver) HasPath(path string) bool {
	for _, l := range r.locations {
		if l.RealPath == path {
			return true
		}
	}
	return false
}

func (r MockResolver) String() string {
	return fmt.Sprintf("mock:(%s,...)", r.locations[0].RealPath)
}

func (r MockResolver) FileContentsByLocation(location Location) (io.ReadCloser, error) {
	for _, l := range r.locations {
		if l == location {
			return os.Open(location.RealPath)
		}
	}

	return nil, fmt.Errorf("no file for location: %v", location)
}

func (r MockResolver) FilesByPath(paths ...string) ([]Location, error) {
	var results []Location
	for _, p := range paths {
		for _, location := range r.locations {
			if p == location.RealPath {
				results = append(results, NewLocation(p))
			}
		}
	}

	return results, nil
}

func (r MockResolver) FilesByGlob(patterns ...string) ([]Location, error) {
	var results []Location
	for _, pattern := range patterns {
		for _, location := range r.locations {
			matches, err := doublestar.Match(pattern, location.RealPath)
			if err != nil {
				return nil, err
			}
			if matches {
				results = append(results, location)
			}
		}
	}

	return results, nil
}

func (r MockResolver) RelativeFileByPath(_ Location, path string) *Location {
	paths, err := r.FilesByPath(path)
	if err != nil {
		return nil
	}

	if len(paths) < 1 {
		return nil
	}

	return &paths[0]
}

func (r MockResolver) AllLocations() <-chan Location {
	results := make(chan Location)
	go func() {
		defer close(results)
		for _, l := range r.locations {
			results <- l
		}
	}()
	return results
}

func (r MockResolver) FileMetadataByLocation(l Location) (FileMetadata, error) {
	info, err := os.Stat(l.RealPath)
	if err != nil {
		return FileMetadata{}, err
	}

	ty := RegularFile
	if info.IsDir() {
		ty = Directory
	}

	return FileMetadata{
		Mode:    info.Mode(),
		Type:    ty,
		UserID:  0,
		GroupID: 0,
		Size:    info.Size(),
	}, nil
}

func (r MockResolver) FilesByMIMEType(types ...string) ([]Location, error) {
	var locations []Location
	for _, ty := range types {
		locations = append(r.mimeTypeIndex[ty], locations...)
	}
	return locations, nil
}
