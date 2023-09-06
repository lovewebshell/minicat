/*
Package apkdb provides a concrete Cataloger implementation for Alpine DB files.
*/
package apkdb

import (
	"github.com/lovewebshell/minicat/minicat/pkg"
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

func NewApkdbCataloger() *common.GenericCataloger {
	globParsers := map[string]common.ParserFn{
		pkg.ApkDBGlob: parseApkDB,
	}

	return common.NewGenericCataloger(nil, globParsers, "apkdb-cataloger")
}
