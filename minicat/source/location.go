package source

import (
	"fmt"

	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/image"
	"github.com/lovewebshell/minicat/internal/log"
)

type Location struct {
	Coordinates `cyclonedx:""`

	VirtualPath string         `hash:"ignore"`
	ref         file.Reference `hash:"ignore"`
}

func NewLocation(realPath string) Location {
	return Location{
		Coordinates: Coordinates{
			RealPath: realPath,
		},
	}
}

func NewVirtualLocation(realPath, virtualPath string) Location {
	return Location{
		Coordinates: Coordinates{
			RealPath: realPath,
		},
		VirtualPath: virtualPath,
	}
}

func NewLocationFromCoordinates(coordinates Coordinates) Location {
	return Location{
		Coordinates: coordinates,
	}
}

func NewLocationFromImage(virtualPath string, ref file.Reference, img *image.Image) Location {
	entry, err := img.FileCatalog.Get(ref)
	if err != nil {
		log.Warnf("unable to find file catalog entry for ref=%+v", ref)
		return Location{
			Coordinates: Coordinates{
				RealPath: string(ref.RealPath),
			},
			VirtualPath: virtualPath,
			ref:         ref,
		}
	}

	return Location{
		Coordinates: Coordinates{
			RealPath:     string(ref.RealPath),
			FileSystemID: entry.Layer.Metadata.Digest,
		},
		VirtualPath: virtualPath,
		ref:         ref,
	}
}

func NewLocationFromDirectory(responsePath string, ref file.Reference) Location {
	return Location{
		Coordinates: Coordinates{
			RealPath: responsePath,
		},
		ref: ref,
	}
}

func NewVirtualLocationFromDirectory(responsePath, virtualResponsePath string, ref file.Reference) Location {
	if responsePath == virtualResponsePath {
		return NewLocationFromDirectory(responsePath, ref)
	}
	return Location{
		Coordinates: Coordinates{
			RealPath: responsePath,
		},
		VirtualPath: virtualResponsePath,
		ref:         ref,
	}
}

func (l Location) String() string {
	str := ""
	if l.ref.ID() != 0 {
		str += fmt.Sprintf("id=%d ", l.ref.ID())
	}

	str += fmt.Sprintf("RealPath=%q", l.RealPath)

	if l.VirtualPath != "" {
		str += fmt.Sprintf(" VirtualPath=%q", l.VirtualPath)
	}

	if l.FileSystemID != "" {
		str += fmt.Sprintf(" Layer=%q", l.FileSystemID)
	}
	return fmt.Sprintf("Location<%s>", str)
}

func (l Location) Equals(other Location) bool {
	return l.RealPath == other.RealPath &&
		l.VirtualPath == other.VirtualPath &&
		l.FileSystemID == other.FileSystemID
}
