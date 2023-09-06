/*
Package python provides a concrete Cataloger implementation for Python ecosystem files (egg, wheel, requirements.txt).
*/
package python

import (
	"github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"
)

func NewPythonIndexCataloger() *common.GenericCataloger {
	globParsers := map[string]common.ParserFn{
		"**/*requirements*.txt": parseRequirementsTxt,
		"**/poetry.lock":        parsePoetryLock,
		"**/Pipfile.lock":       parsePipfileLock,
		"**/setup.py":           parseSetup,
	}

	return common.NewGenericCataloger(nil, globParsers, "python-index-cataloger")
}
