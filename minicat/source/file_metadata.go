package source

import (
	"os"

	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/image"
	"github.com/lovewebshell/minicat/internal/log"
)

type FileMetadata struct {
	Mode            os.FileMode
	Type            FileType
	UserID          int
	GroupID         int
	LinkDestination string
	Size            int64
	MIMEType        string
}

func fileMetadataByLocation(img *image.Image, location Location) (FileMetadata, error) {
	entry, err := img.FileCatalog.Get(location.ref)
	if err != nil {
		return FileMetadata{}, err
	}

	return FileMetadata{
		Mode:            entry.Metadata.Mode,
		Type:            newFileTypeFromTarHeaderTypeFlag(entry.Metadata.TypeFlag),
		UserID:          entry.Metadata.UserID,
		GroupID:         entry.Metadata.GroupID,
		LinkDestination: entry.Metadata.Linkname,
		Size:            entry.Metadata.Size,
		MIMEType:        entry.Metadata.MIMEType,
	}, nil
}

func fileMetadataFromPath(path string, info os.FileInfo, withMIMEType bool) FileMetadata {
	var mimeType string
	uid, gid := GetXid(info)

	if withMIMEType {
		f, err := os.Open(path)
		if err != nil {

			f = nil
		} else {
			defer func() {
				if err := f.Close(); err != nil {
					log.Warnf("unable to close file while obtaining metadata: %s", path)
				}
			}()
		}

		mimeType = file.MIMEType(f)
	}

	return FileMetadata{
		Mode: info.Mode(),
		Type: newFileTypeFromMode(info.Mode()),

		UserID:   uid,
		GroupID:  gid,
		Size:     info.Size(),
		MIMEType: mimeType,
	}
}
