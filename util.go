package locale

import "golang.org/x/text/language"

type Languager interface {
	Language() language.Tag
}
