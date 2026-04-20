package locale

import (
	"fmt"
	"unicode/utf8"

	"github.com/tdewolff/parse/v2/strconv"
)

func roundToInt64(f float64, dec int) int64 {
	return int64(f*float64(int64Scales[dec]) + 0.5)
}

type DecimalFormatter struct {
	Num float64
}

func (f DecimalFormatter) Format(state fmt.State, verb rune) {
	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}
	pattern := locale.DecimalFormat

	var b []byte
	num := f.Num
	if num < 0.0 {
		b = append(b, '-')
		num = -num
	}

	dec := 6
	if precision, ok := state.Precision(); ok {
		dec = precision
	}
	for i := 0; i < len(pattern); {
		r, n := utf8.DecodeRuneInString(pattern[i:])
		switch r {
		case '0', '#':
			j := i + 1
			group, decimal := -1, -1
			for j < len(pattern) {
				switch pattern[j] {
				case '.':
					if decimal != -1 {
						break
					}
					decimal = j
				case ',':
					if decimal != -1 {
						break
					}
					group = j
				}
				j++
			}

			groupSize := 3
			if decimal != -1 && group != -1 {
				groupSize = decimal - group
			}
			amount := roundToInt64(num, dec)
			b = strconv.AppendNumber(b, amount, dec, groupSize, locale.GroupSymbol, locale.DecimalSymbol)
			i = j - 1
		case ' ':
			b = utf8.AppendRune(b, '\u00A0') // non-breaking space
		case '\'':
			j := i + 1
			for j < len(pattern) {
				if pattern[j] == '\'' {
					break
				}
				j++
			}
			b = append(b, pattern[i+1:j]...)
			i = j - 1
		default:
			b = append(b, []byte(pattern[i-n:i])...)
		}
		i += n
	}
	if width, ok := state.Width(); ok && len(b) < width {
		c := byte(' ')
		if state.Flag('0') {
			c = '0'
		}
		pad := make([]byte, width-len(b))
		for i := range pad {
			pad[i] = c
		}
		state.Write(pad)
	}
	state.Write(b)
}
