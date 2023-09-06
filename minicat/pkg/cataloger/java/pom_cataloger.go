package java

import "github.com/lovewebshell/minicat/minicat/pkg/cataloger/common"

const javaPomCataloger = "java-pom-cataloger"

func NewJavaPomCataloger() *common.GenericCataloger {
	globParsers := make(map[string]common.ParserFn)

	globParsers[pomXMLDirGlob] = parserPomXML

	return common.NewGenericCataloger(nil, globParsers, javaPomCataloger)
}
