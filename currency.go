package locale

import (
	"database/sql/driver"
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tdewolff/parse/v2/strconv"
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

const AmountPrecision = 3 // extra decimals for arithmetics

var int64Scales = [...]int64{
	1,
	10,
	100,
	1000,
	10000,
	100000,
	1000000, // 1e6
	10000000,
	100000000,
	1000000000,
	10000000000,
	100000000000,
	1000000000000, // 1e12
	10000000000000,
	100000000000000,
	1000000000000000,
	10000000000000000,
	100000000000000000,
	1000000000000000000, // 1e18
}

type Amount struct {
	currency.Unit
	amount           int64
	rounding, digits int
}

func ParseAmount(unit currency.Unit, s string) (Amount, error) {
	return ParseAmountBytes(unit, []byte(s))
}

func ParseAmountBytes(unit currency.Unit, b []byte) (Amount, error) {
	amount, dec, n := strconv.ParseNumber(b, ',', '.')
	if n != len(b) {
		return Amount{}, fmt.Errorf("invalid amount: %v", string(b))
	}
	return NewAmount(unit, amount, dec), nil
}

func ParseAmountLocale(tag language.Tag, unit currency.Unit, s string) (Amount, error) {
	locale := GetLocale(tag)
	amount, dec, n := strconv.ParseNumber([]byte(s), locale.GroupSymbol, locale.DecimalSymbol)
	if n != len(s) {
		return Amount{}, fmt.Errorf("invalid amount: %v", s)
	}
	return NewAmount(unit, amount, dec), nil
}

func NewAmount(unit currency.Unit, amount int64, dec int) Amount {
	cur := GetCurrency(unit)
	scale := cur.Digits + AmountPrecision
	if dec < scale {
		scaleMul := int64Scales[scale-dec]
		if math.MaxInt64/scaleMul < amount {
			panic("overflow")
		} else if amount < math.MinInt64/scaleMul {
			panic("underflow")
		}
		amount *= scaleMul
	} else if scale < dec {
		amount /= int64Scales[dec-scale]
	}
	return Amount{unit, amount, cur.Rounding, cur.Digits}
}

func NewAmountFromFloat64(unit currency.Unit, amount float64) Amount {
	cur := GetCurrency(unit)
	scale := cur.Digits + AmountPrecision
	a := int64(math.RoundToEven(amount * math.Pow10(scale)))
	return Amount{unit, a, cur.Rounding, cur.Digits}
}

func (a Amount) IsZero() bool {
	return a.amount == 0
}

// Round performs banker's rounding to the currency's increments
func (a Amount) Round() Amount {
	return a.round(a.rounding)
}

func (a Amount) round(incr int) Amount {
	scale := int64Scales[AmountPrecision]
	switch incr {
	case 0, 1:
		// no-op
	case 10, 100:
		scale *= int64(incr)
	default:
		panic(fmt.Sprintf("unexpected increment: %v", incr))
	}

	shift := int64(0)
	if carry := (a.amount / (scale / 10)) % 10; carry == 5 {
		if isEven := ((a.amount / scale) % 2) == 0; !isEven {
			shift = scale
		}
	} else if 5 < carry {
		shift = scale
	}
	a.amount += -(a.amount % scale) + shift
	return a
}

func (a Amount) Neg() Amount {
	if a.amount == math.MinInt64 {
		panic("overflow")
	}
	a.amount = -a.amount
	return a
}

func (a Amount) Abs() Amount {
	if a.amount < 0 {
		a.Neg()
	}
	return a
}

func (a Amount) Add(b Amount) Amount {
	if a.Unit != b.Unit {
		panic(fmt.Sprintf("currencies don't match: %v != %v", a.Unit, b.Unit))
	} else if 0 < b.amount && math.MaxInt64-b.amount < a.amount {
		panic("overflow")
	} else if b.amount == math.MinInt64 || b.amount < 0 && a.amount < math.MinInt64-b.amount {
		panic("underflow")
	}
	a.amount += b.amount
	return a
}

func (a Amount) Sub(b Amount) Amount {
	if a.Unit != b.Unit {
		panic(fmt.Sprintf("currencies don't match: %v != %v", a.Unit, b.Unit))
	} else if 0 < b.amount && a.amount < math.MinInt64+b.amount {
		panic("underflow")
	} else if b.amount == math.MinInt64 || b.amount < 0 && math.MaxInt64+b.amount < a.amount {
		panic("overflow")
	}
	a.amount -= b.amount
	return a
}

func (a Amount) Mul(f int) Amount {
	// TODO: is this right?
	if 1 < f && 0 < a.amount && math.MaxInt64/int64(f) < a.amount {
		panic("overflow")
	} else if f < -1 && a.amount < 0 && math.MaxInt64/int64(-f) < -a.amount {
		panic("overflow")
	} else if f < -1 && 0 < a.amount && a.amount < math.MinInt64/int64(f) {
		panic("underflow")
	} else if 1 < f && a.amount < 0 && -a.amount < math.MinInt64/int64(-f) {
		panic("underflow")
	}
	a.amount *= int64(f)
	return a
}

func (a Amount) Div(f int) Amount {
	a.amount /= int64(f)
	return a
}

func (a Amount) Mulf(f float64) Amount {
	// TODO: is this right?
	incr := 1.0 < f && 0 < a.amount || f < -1.0 && a.amount < 0
	decr := f < -1.0 && 0 < a.amount || 1.0 < f && a.amount < 0
	if incr && float64(math.MaxInt64) < f*float64(a.amount) {
		panic("overflow")
	} else if decr && f*float64(a.amount) < float64(math.MinInt64) {
		panic("underflow")
	}
	a.amount = int64(float64(a.amount)*f + 0.5)
	return a
}

func (a Amount) Float64() float64 {
	return float64(a.amount) / math.Pow10(a.digits+AmountPrecision)
}

func (a Amount) Amount() (int64, int) {
	a = a.round(1)
	return a.amount / int64Scales[AmountPrecision], a.digits
}

func (a Amount) StringAmount() string {
	var b []byte
	amount, dec := a.Amount()
	b = strconv.AppendNumber(b, amount, dec, 0, 0, '.')
	return string(b)
}

func (a Amount) String() string {
	return a.Unit.String() + " " + a.StringAmount()
}

func (a *Amount) Scan(isrc interface{}) error {
	var b []byte
	switch src := isrc.(type) {
	case Amount:
		*a = src
		return nil
	case []byte:
		b = src
	case string:
		b = []byte(src)
	default:
		return fmt.Errorf("unexpected type for amount: %T", isrc)
	}

	if len(b) < 4 {
		return fmt.Errorf("invalid amount: %v", isrc)
	}
	unit, err := currency.ParseISO(string(b[:3]))
	if err != nil {
		return fmt.Errorf("invalid amount: %v", err)
	}
	i := 3
	if b[i] == ' ' {
		i++
	}
	amount, err := ParseAmountBytes(unit, b[i:])
	if err != nil {
		return err
	}
	*a = amount
	return nil
}

func (a Amount) Value() (driver.Value, error) {
	return a.String(), nil
}

type NullAmount struct {
	Amount
	Valid bool
}

// Scan implements the Scanner interface.
func (n *NullAmount) Scan(value any) error {
	if value == nil {
		n.Amount, n.Valid = Amount{}, false
		return nil
	}
	n.Valid = true
	return n.Amount.Scan(value)
}

// Value implements the driver Valuer interface.
func (n NullAmount) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Amount, nil
}

func AmountRegex(tag language.Tag, unit currency.Unit) string {
	cur := GetCurrency(unit)
	locale := GetLocale(tag)

	var decimals string
	if 0 < cur.Digits {
		decimals = fmt.Sprintf("(?:%s", regexp.QuoteMeta(string(locale.DecimalSymbol)))
		switch cur.Rounding {
		case 0, 1:
			decimals += fmt.Sprintf("[0-9]{,%d}", cur.Digits)
		case 10:
			if 1 < cur.Digits {
				decimals += fmt.Sprintf("(?:[0-9]{,%d}|[0-9]{%d}0)", cur.Digits-1, cur.Digits-1)
			} else {
				decimals += "0"
			}
		case 100:
			if 2 < cur.Digits {
				decimals += fmt.Sprintf("(?:[0-9]{,%d}|[0-9]{%d}00)", cur.Digits-2, cur.Digits-2)
			} else {
				decimals += "00"
			}
		default:
			panic(fmt.Sprintf("unexpected increment: %v", cur.Rounding))
		}
		decimals += ")?"
	}
	return fmt.Sprintf("^(?:[0-9]+%s)*[0-9]+%s$", regexp.QuoteMeta(string(locale.GroupSymbol)), decimals)
}

type CurrencyFormatter struct {
	currency.Unit
	Layout string
}

func (f CurrencyFormatter) Format(state fmt.State, verb rune) {
	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}

	s := ""
	unit := f.Unit.String()
	switch f.Layout {
	case "USD":
		s = unit
	case "US$":
		s = locale.Currency[unit].Standard
	case "$":
		s = locale.Currency[unit].Narrow
	default:
		s = locale.Currency[f.Unit.String()].Name
	}
	state.Write([]byte(s))
}

// Available currency formats
// TODO: support accounting formats?
const (
	CurrencyAmount   string = "100"
	CurrencyISO             = "USD 100"
	CurrencyStandard        = "US$ 100"
	CurrencyNarrow          = "$100"
)

type AmountFormatter struct {
	Amount
	Layout string
}

func (f AmountFormatter) Format(state fmt.State, verb rune) {
	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}

	unit := f.Unit.String()
	var symbol, pattern string
	switch f.Layout {
	case CurrencyISO:
		symbol = unit
		pattern = locale.CurrencyFormat.ISO
	case CurrencyStandard:
		symbol = locale.Currency[unit].Standard
		hasLetter := false
		for _, r := range symbol {
			if unicode.IsLetter(r) {
				hasLetter = true
				break
			}
		}
		if hasLetter {
			pattern = locale.CurrencyFormat.ISO
		} else {
			pattern = locale.CurrencyFormat.Standard
		}
	case CurrencyNarrow:
		symbol = locale.Currency[unit].Narrow
		pattern = locale.CurrencyFormat.Standard
	case CurrencyAmount:
		pattern = locale.CurrencyFormat.Amount
	default:
		log.Printf("INFO: locale: unsupported currency format: %v\n", f.Layout)
	}

	if idx := strings.IndexByte(pattern, ';'); idx != -1 {
		if f.amount < 0 {
			pattern = pattern[idx+1:]
		} else {
			pattern = pattern[:idx]
		}
	}

	a := f.Amount.round(f.rounding)
	dec := f.digits
	amount := a.amount / int64Scales[AmountPrecision]

	var b []byte
	for i := 0; i < len(pattern); {
		r, n := utf8.DecodeRuneInString(pattern[i:])
		switch r {
		// TODO: handle negative amounts
		case '¤':
			b = append(b, symbol...)
		case '0', '#':
			j := i + 1
			group, decimal := -1, -1
			for j < len(pattern) {
				if pattern[j] == '.' {
					if decimal != -1 {
						break
					}
					decimal = j
				} else if pattern[j] == ',' {
					if decimal != -1 {
						break
					}
					group = j
				} else if pattern[j] != '0' && pattern[j] != '#' {
					break
				}
				j++
			}

			groupSize := 3
			if decimal != -1 && group != -1 {
				groupSize = decimal - group - 1
			}
			b = strconv.AppendNumber(b, amount, dec, groupSize, locale.GroupSymbol, locale.DecimalSymbol)
			i = j - 1
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
			b = append(b, []byte(pattern[i:i+n])...)
		}
		i += n
	}
	state.Write(b)
}
