package locale

import "golang.org/x/text/language"

func ToLocaleName(tag language.Tag) string {
	if tag == language.Und || tag.IsRoot() {
		return "root"
	}
	return tag.String()
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
