/*
Package golang provides a concrete Cataloger implementation for go.mod files.
*/
package golang

import (
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

func NewGoModFileCataloger() *common.GenericCataloger {
	globParsers := map[string]common.ParserFn{
		"**/go.mod": parseGoMod,
	}

	return common.NewGenericCataloger(nil, globParsers, "go-mod-file-cataloger")
}
