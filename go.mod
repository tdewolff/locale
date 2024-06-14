module github.com/tdewolff/locale

go 1.21.2

replace github.com/tdewolff/parse/v2 => ../parse

require (
	github.com/tdewolff/parse/v2 v2.7.15
	github.com/tdewolff/test v1.0.11-0.20231101010635-f1265d231d52
	golang.org/x/text v0.16.0
)
