package file

import (
	"github.com/lovewebshell/minicat/minicat/source"
)

type MetadataCataloger struct {
}

func NewMetadataCataloger() *MetadataCataloger {
	return &MetadataCataloger{}
}

func (i *MetadataCataloger) Catalog(resolver source.FileResolver) (map[source.Coordinates]source.FileMetadata, error) {
	results := make(map[source.Coordinates]source.FileMetadata)
	var locations []source.Location
	for location := range resolver.AllLocations() {
		locations = append(locations, location)
	}

	for _, location := range locations {
		metadata, err := resolver.FileMetadataByLocation(location)
		if err != nil {
			return nil, err
		}

		results[location.Coordinates] = metadata

	}

	return results, nil
}
