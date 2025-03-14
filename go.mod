module github.com/tdewolff/locale

go 1.23.0

toolchain go1.24.1

replace github.com/tdewolff/parse/v2 => ../parse

require (
	github.com/tdewolff/parse/v2 v2.7.20
	github.com/tdewolff/test v1.0.11
	golang.org/x/text v0.23.0
)
