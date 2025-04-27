package locale

import (
	"time"

	"golang.org/x/text/language"
)

var tzCET = time.FixedZone("CET", 2*3600)
var tzPST = time.FixedZone("PST", -8*3600)

var en = NewPrinter(language.English, tzPST)
var es = NewPrinter(language.Spanish, tzCET)
