package pkg

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"
)

type CPE = wfn.Attributes

const (
	allowedCPEPunctuation = "-!\"#$%&'()+,./:;<=>@[]^`{|}~"
)

const cpeRegexString = ((`^([c][pP][eE]:/[AHOaho]?(:[A-Za-z0-9\._\-~%]*){0,6})`) +
	`|(cpe:2\.3:[aho\*\-](:(((\?*|\*?)([a-zA-Z0-9\-\._]|(\\[\\\*\?!"#$$%&'\(\)\+,/:;<=>@\[\]\^\x60\{\|}~]))+(\?*|\*?))|[\*\-])){5}(:(([a-zA-Z]{2,3}(-([a-zA-Z]{2}|[0-9]{3}))?)|[\*\-]))(:(((\?*|\*?)([a-zA-Z0-9\-\._]|(\\[\\\*\?!"#$$%&'\(\)\+,/:;<=>@\[\]\^\x60\{\|}~]))+(\?*|\*?))|[\*\-])){4})$`)

var cpeRegex = regexp.MustCompile(cpeRegexString)

func NewCPE(cpeStr string) (CPE, error) {
	c, err := newCPEWithoutValidation(cpeStr)
	if err != nil {
		return CPE{}, fmt.Errorf("unable to parse CPE string: %w", err)
	}

	if ValidateCPEString(CPEString(c)) != nil {
		return CPE{}, err
	}

	return c, nil
}

func ValidateCPEString(cpeStr string) error {
	if !cpeRegex.MatchString(cpeStr) {
		return fmt.Errorf("failed to parse CPE=%q as it doesn't match the regex=%s", cpeStr, cpeRegexString)
	}
	return nil
}

func newCPEWithoutValidation(cpeStr string) (CPE, error) {
	value, err := wfn.Parse(cpeStr)
	if err != nil {
		return CPE{}, fmt.Errorf("failed to parse CPE=%q: %w", cpeStr, err)
	}

	if value == nil {
		return CPE{}, fmt.Errorf("failed to parse CPE=%q", cpeStr)
	}

	value.Vendor = normalizeCpeField(value.Vendor)
	value.Product = normalizeCpeField(value.Product)
	value.Language = normalizeCpeField(value.Language)
	value.Version = normalizeCpeField(value.Version)
	value.TargetSW = normalizeCpeField(value.TargetSW)
	value.Part = normalizeCpeField(value.Part)
	value.Edition = normalizeCpeField(value.Edition)
	value.Other = normalizeCpeField(value.Other)
	value.SWEdition = normalizeCpeField(value.SWEdition)
	value.TargetHW = normalizeCpeField(value.TargetHW)
	value.Update = normalizeCpeField(value.Update)

	return *value, nil
}

func normalizeCpeField(field string) string {

	field = strings.ReplaceAll(field, " ", "_")

	if field == "*" {
		return wfn.Any
	}
	return stripSlashes(field)
}

func stripSlashes(s string) string {
	sb := strings.Builder{}
	for i, c := range s {
		if c == '\\' && i+1 < len(s) && strings.ContainsRune(allowedCPEPunctuation, rune(s[i+1])) {
			continue
		} else {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}

func CPEString(c CPE) string {
	output := CPE{}
	output.Vendor = sanitize(c.Vendor)
	output.Product = sanitize(c.Product)
	output.Language = sanitize(c.Language)
	output.Version = sanitize(c.Version)
	output.TargetSW = sanitize(c.TargetSW)
	output.Part = sanitize(c.Part)
	output.Edition = sanitize(c.Edition)
	output.Other = sanitize(c.Other)
	output.SWEdition = sanitize(c.SWEdition)
	output.TargetHW = sanitize(c.TargetHW)
	output.Update = sanitize(c.Update)
	return output.BindToFmtString()
}

func sanitize(s string) string {

	in := strings.ReplaceAll(s, " ", "_")

	sb := strings.Builder{}
	for _, c := range in {
		if strings.ContainsRune(allowedCPEPunctuation, c) {
			sb.WriteRune('\\')
		}
		sb.WriteRune(c)
	}
	return sb.String()
}
