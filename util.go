package locale

import (
	"strings"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

type Languager interface {
	Language() language.Tag
}

func GetSupportedTag(tag language.Tag) language.Tag {
	loc := strings.ReplaceAll(tag.String(), "-", "_")
	_, ok := locales[loc]
	for !ok && loc != "root" {
		tag = tag.Parent()
		if tag == language.Und {
			loc = "root"
		} else {
			loc = strings.ReplaceAll(tag.String(), "-", "_")
		}
		_, ok = locales[loc]
	}
	return language.Make(loc)
}

func GetLocale(tag language.Tag) Locale {
	tag = GetSupportedTag(tag)
	loc := strings.ReplaceAll(tag.String(), "-", "_")
	return locales[loc]
}

func GetCurrency(unit currency.Unit) CurrencyInfo {
	d, ok := currencies[unit.String()]
	if !ok {
		d, _ = currencies["DEFAULT"]
	}
	return d
}
