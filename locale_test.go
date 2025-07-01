package locale

import (
	"time"

	"golang.org/x/text/language"
)

var tzCET = time.FixedZone("Europe/Paris", 2*3600)
var tzPST = time.FixedZone("America/Los_Angeles", -8*3600)

var en = NewPrinter(language.English, tzPST)
var es = NewPrinter(language.Spanish, tzCET)
