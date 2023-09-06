/*
Package java provides a concrete Cataloger implementation for Java archives (jar, war, ear, par, sar, jpi, hpi formats).
*/
package java

import (
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

func NewJavaCataloger(cfg Config) *common.GenericCataloger {
	globParsers := make(map[string]common.ParserFn)

	for _, pattern := range archiveFormatGlobs {
		globParsers[pattern] = parseJavaArchive
	}

	if cfg.SearchIndexedArchives {

		for _, pattern := range genericZipGlobs {
			globParsers[pattern] = parseZipWrappedJavaArchive
		}
	}

	if cfg.SearchUnindexedArchives {

		for _, pattern := range genericTarGlobs {
			globParsers[pattern] = parseTarWrappedJavaArchive
		}
	}

	return common.NewGenericCataloger(nil, globParsers, "java-cataloger")
}
