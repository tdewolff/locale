package locale

import (
	"strings"

	"golang.org/x/text/language"
)

type Languager interface {
	Language() language.Tag
}

func ToLocaleName(tag language.Tag) string {
	return strings.ReplaceAll(tag.String(), "-", "_")
}

func GetLocale(loc string) Locale {
	d, ok := locales[loc]
	for !ok && loc != "root" {
		loc = ToLocaleName(language.MustParse(loc).Parent())
		d, ok = locales[loc]
	}
	return d
}

func GetCurrency(unit string) Currency {
	d, ok := currencies[unit]
	if !ok {
		d, _ = currencies["DEFAULT"]
	}
	return d
}
