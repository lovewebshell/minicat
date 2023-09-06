package pkg

import (
	"github.com/anchore/packageurl-go"
	"strings"
)

type Language string

const (
	UnknownLanguage Language = ""
	Java            Language = "java"
	JavaScript      Language = "javascript"
	Python          Language = "python"
	Go              Language = "go"
	Maven           Language = "maven"
	Gradle          Language = "gradle"
)

var AllLanguages = []Language{
	Java,
	JavaScript,
	Python,
	Go,
	Maven,
	Gradle,
}

func (l Language) String() string {
	return string(l)
}

func LanguageFromPURL(p string) Language {
	purl, err := packageurl.FromString(p)
	if err != nil {
		return UnknownLanguage
	}

	return LanguageByName(purl.Type)
}

func LanguageByName(name string) Language {
	switch strings.ToLower(name) {
	case packageurl.TypeMaven, string(purlGradlePkgType), string(JavaPkg), string(Java):
		return Java
	case packageurl.TypeGolang, string(GoModulePkg), string(Go):
		return Go
	case packageurl.TypeNPM, string(JavaScript), "nodejs", "node.js":
		return JavaScript
	case packageurl.TypePyPi, string(Python):
		return Python
	default:
		return UnknownLanguage
	}
}
