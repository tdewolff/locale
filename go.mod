module github.com/tdewolff/cldr

go 1.21.2

replace github.com/tdewolff/parse/v2 => ../parse

require (
	github.com/tdewolff/parse/v2 v2.6.8
	github.com/tdewolff/test v1.0.9
	golang.org/x/text v0.13.0
)
