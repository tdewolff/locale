package locale

import (
	"strings"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

type Languager interface {
	Language() language.Tag
}

func GetLocale(tag language.Tag) Locale {
	loc := strings.ReplaceAll(tag.String(), "-", "_")
	d, ok := locales[loc]
	for !ok && loc != "root" {
		tag = tag.Parent()
		if tag == language.Und {
			loc = "root"
		} else {
			loc = strings.ReplaceAll(tag.String(), "-", "_")
		}
		d, ok = locales[loc]
	}
	return d
}

func GetCurrency(unit currency.Unit) CurrencyInfo {
	d, ok := currencies[unit.String()]
	if !ok {
		d, _ = currencies["DEFAULT"]
	}
	return d
}
