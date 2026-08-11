package locale

import (
	"strings"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

var localeMap = map[language.Tag]Locale{}

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
	locale, ok := localeMap[tag]
	if !ok {
		supportedTag := GetSupportedTag(tag)
		locale = locales[strings.ReplaceAll(supportedTag.String(), "-", "_")]
		localeMap[tag] = locale
	}
	return locale
}

func GetCurrency(unit currency.Unit) CurrencyInfo {
	d, ok := currencies[unit.String()]
	if !ok {
		d, _ = currencies["DEFAULT"]
	}
	return d
}
