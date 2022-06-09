package utils

import (
	"regexp"
	"strings"
)

var (
	idReg, _   = regexp.Compile(`/([\d]+)(/|$)`)
	typeReg, _ = regexp.Compile(`si=(profile|heap|allocs|black|mutex)_`)
)

func ExtractProfileID(path string) string {
	return strings.ReplaceAll(idReg.FindString(path), "/", "")
}

func RemovePrefixSampleType(rawQuery string) string {
	return typeReg.ReplaceAllString(rawQuery, "si=")
}
