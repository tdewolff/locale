package locale

import (
	"fmt"

	"golang.org/x/text/language"
)

type RegionFormatter struct {
	language.Region
}

func (f RegionFormatter) Format(state fmt.State, verb rune) {
	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}
	region := f.Region.String()
	if region != "ZZ" {
		territory, _ := locale.Territory[region]
		state.Write([]byte(territory))
	}
}
