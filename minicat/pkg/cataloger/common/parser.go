package common

import (
	"io"

	"github.com/lovewebshell/minicat/minicat/artifact"
	"github.com/lovewebshell/minicat/minicat/pkg"
)

type ParserFn func(string, io.Reader) ([]*pkg.Package, []artifact.Relationship, error)
