package source

import (
	"fmt"

	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/artifact"
)

type Coordinates struct {
	RealPath     string `json:"path" cyclonedx:"path"`
	FileSystemID string `json:"layerID,omitempty" cyclonedx:"layerID"`
}

func (c Coordinates) ID() artifact.ID {
	f, err := artifact.IDByHash(c)
	if err != nil {

		log.Warnf("unable to get fingerprint of location coordinate=%+v: %+v", c, err)
		return ""
	}

	return f
}

func (c Coordinates) String() string {
	str := fmt.Sprintf("RealPath=%q", c.RealPath)

	if c.FileSystemID != "" {
		str += fmt.Sprintf(" Layer=%q", c.FileSystemID)
	}
	return fmt.Sprintf("Location<%s>", str)
}
