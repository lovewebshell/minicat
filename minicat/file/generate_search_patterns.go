package file

import (
	"fmt"
	"regexp"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/hashicorp/go-multierror"
)

func GenerateSearchPatterns(basePatterns map[string]string, additionalPatterns map[string]string, excludePatternNames []string) (map[string]*regexp.Regexp, error) {
	var regexObjs = make(map[string]*regexp.Regexp)
	var errs error

	addFn := func(name, pattern string) {

		obj, err := regexp.Compile(`(?m)` + pattern)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("unable to parse %q regular expression: %w", name, err))
		}
		regexObjs[name] = obj
	}

	for name, pattern := range basePatterns {
		if !matchesExclusion(excludePatternNames, name) {
			addFn(name, pattern)
		}
	}

	for name, pattern := range additionalPatterns {
		addFn(name, pattern)
	}

	if errs != nil {
		return nil, errs
	}

	return regexObjs, nil
}

func matchesExclusion(excludePatternNames []string, name string) bool {
	for _, exclude := range excludePatternNames {
		matches, err := doublestar.Match(exclude, name)
		if err != nil {
			return false
		}
		if matches {
			return true
		}
	}
	return false
}
