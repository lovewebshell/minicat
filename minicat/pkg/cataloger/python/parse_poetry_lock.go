package python

import (
	"fmt"
	"io"

	"github.com/pelletier/go-toml"

	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

var _ common.ParserFn = parsePoetryLock

func parsePoetryLock(_ string, reader io.Reader) ([]*pkg.Package, []artifact.Relationship, error) {
	tree, err := toml.LoadReader(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to load poetry.lock for parsing: %v", err)
	}

	metadata := PoetryMetadata{}
	err = tree.Unmarshal(&metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse poetry.lock: %v", err)
	}

	return metadata.Pkgs(), nil, nil
}
