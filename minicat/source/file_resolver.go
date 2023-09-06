package source

import (
	"io"
)

type FileResolver interface {
	FileContentResolver
	FilePathResolver
	FileLocationResolver
	FileMetadataResolver
}

type FileContentResolver interface {
	FileContentsByLocation(Location) (io.ReadCloser, error)
}

type FileMetadataResolver interface {
	FileMetadataByLocation(Location) (FileMetadata, error)
}

type FilePathResolver interface {
	HasPath(string) bool

	FilesByPath(paths ...string) ([]Location, error)

	FilesByGlob(patterns ...string) ([]Location, error)

	FilesByMIMEType(types ...string) ([]Location, error)

	RelativeFileByPath(_ Location, path string) *Location
}

type FileLocationResolver interface {
	AllLocations() <-chan Location
}
