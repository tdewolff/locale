package locale

import (
	"fmt"
	"math"
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

	dec, exp := 6, 0
	if verb == 'v' {
		verb = 'g'
	}
	if verb == 'd' {
		dec = 0
	} else if precision, ok := state.Precision(); ok {
		dec = precision
	} else if verb == 'g' || verb == 'G' {
		dec = 15
		if f.Num != 0.0 {
			if exp = int(math.Log10(math.Abs(f.Num))); 3 < exp {
				f.Num *= math.Pow10(-exp)
			} else {
				exp = 0
			}
		}
	} else if verb == 'e' || verb == 'E' {
		dec = 6
		if f.Num != 0.0 {
			exp = int(math.Log10(math.Abs(f.Num)))
			f.Num *= math.Pow10(-exp)
		}
	} else if verb != 'f' && verb != 'F' {
		fmt.Fprintf(state, fmt.FormatString(state, verb), f.Num)
		return
	}

	var b []byte
	num := f.Num
	if num < 0.0 {
		b = append(b, '-')
		num = -num
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
				groupSize = decimal - group - 1
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
	if 0 < dec && verb == 'g' || verb == 'G' {
		// remove trailing zeros
		for i := len(b) - 1; 0 <= i; i-- {
			if b[i] != '0' {
				if b[i] < '0' || '9' < b[i] {
					_, n := utf8.DecodeLastRune(b[:i+1])
					i -= n // decimal symbol
				}
				b = b[:i+1]
				break
			}
		}
	}
	if exp != 0 || verb == 'e' || verb == 'E' {
		b = append(b, fmt.Sprintf("\u00A0×\u00A010")...)
		for _, c := range fmt.Sprintf("%d", exp) {
			switch c {
			case '0':
				b = append(b, "⁰"...)
			case '1':
				b = append(b, "¹"...)
			case '2':
				b = append(b, "²"...)
			case '3':
				b = append(b, "³"...)
			case '4':
				b = append(b, "⁴"...)
			case '5':
				b = append(b, "⁵"...)
			case '6':
				b = append(b, "⁶"...)
			case '7':
				b = append(b, "⁷"...)
			case '8':
				b = append(b, "⁸"...)
			case '9':
				b = append(b, "⁹"...)
			case '-':
				b = append(b, "⁻"...)
			}
		}
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
