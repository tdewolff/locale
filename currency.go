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

	"golang.org/x/text/currency"
	"golang.org/x/text/language"

	"github.com/tdewolff/parse/v2/strconv"
)

var ErrOverflow = fmt.Errorf("overflow")
var ErrUnderflow = fmt.Errorf("underflow")

// AmountPrecision is the number of extra decimals after the currency's default number of digits
// Most currencies have 2 digits (cents) and thus will use 5 digitis for arithmetics
const AmountPrecision = 3

const MaxAmount = 1<<63 - 1

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

var ZeroAmount = Amount{}

type Amount struct {
	currency.Unit
	amount   int64 // amount multiplied by 10^(digits + AmountPrecision)
	digits   int   // decimal digits for display
	rounding int   // rounding increment
}

func ParseAmount(unit currency.Unit, s string) (Amount, error) {
	return ParseAmountBytes(unit, []byte(s))
}

func ParseAmountBytes(unit currency.Unit, b []byte) (Amount, error) {
	amount, dec, n := strconv.ParseNumber(b, ',', '.')
	if n != len(b) {
		return Amount{}, fmt.Errorf("invalid amount: %v", string(b))
	}
	return NewAmount(unit, amount, dec)
}

func ParseAmountLocale(tag language.Tag, unit currency.Unit, s string) (Amount, error) {
	locale := GetLocale(tag)
	amount, dec, n := strconv.ParseNumber([]byte(s), locale.GroupSymbol, locale.DecimalSymbol)
	if n != len(s) {
		return Amount{}, fmt.Errorf("invalid amount: %v", s)
	}
	return NewAmount(unit, amount, dec)
}

func MustNewZeroAmount(unit currency.Unit) Amount {
	a, err := NewZeroAmount(unit)
	if err != nil {
		panic(err)
	}
	return a
}

func NewZeroAmount(unit currency.Unit) (Amount, error) {
	cur := GetCurrency(unit)
	if cur.Rounding != 0 && cur.Rounding != 1 && cur.Rounding != 10 && cur.Rounding != 100 {
		panic(fmt.Sprintf("unsupported currency rounding: %v", cur.Rounding))
	}
	return Amount{unit, 0, cur.Digits, cur.Rounding}, nil
}

func MustNewAmount(unit currency.Unit, amount int64, dec int) Amount {
	a, err := NewAmount(unit, amount, dec)
	if err != nil {
		panic(err)
	}
	return a
}

func NewAmount(unit currency.Unit, amount int64, dec int) (Amount, error) {
	cur := GetCurrency(unit)
	if cur.Rounding != 0 && cur.Rounding != 1 && cur.Rounding != 10 && cur.Rounding != 100 {
		return Amount{}, fmt.Errorf("unsupported currency rounding: %v", cur.Rounding)
	}
	prec := cur.Digits + AmountPrecision
	if dec < prec {
		scale := int64Scales[prec-dec]
		if MaxAmount/scale < amount {
			return Amount{}, ErrOverflow
		} else if amount < -MaxAmount/scale {
			return Amount{}, ErrUnderflow
		}
		amount *= scale
	} else if prec < dec {
		amount = bankersRounding(amount, dec-prec)
		amount /= int64Scales[dec-prec]
	}
	return Amount{unit, amount, cur.Digits, cur.Rounding}, nil
}

func MustNewAmountFromFloat64(unit currency.Unit, amount float64) Amount {
	a, err := NewAmountFromFloat64(unit, amount)
	if err != nil {
		panic(err)
	}
	return a
}

func NewAmountFromFloat64(unit currency.Unit, amount float64) (Amount, error) {
	cur := GetCurrency(unit)
	if cur.Rounding != 0 && cur.Rounding != 1 && cur.Rounding != 10 && cur.Rounding != 100 {
		return Amount{}, fmt.Errorf("unsupported currency rounding: %v", cur.Rounding)
	}
	prec := cur.Digits + AmountPrecision
	amount = math.RoundToEven(amount * math.Pow10(prec))
	if float64(MaxAmount) < amount {
		return Amount{}, ErrOverflow
	} else if amount < float64(-MaxAmount) {
		return Amount{}, ErrUnderflow
	}
	return Amount{unit, int64(amount), cur.Digits, cur.Rounding}, nil
}

// Zero returns the zero value for the amount (keep the currency).
func (a Amount) Zero() Amount {
	a.amount = 0
	return a
}

func (a Amount) IsZero() bool {
	return a.amount == 0
}

func (a Amount) IsNegative() bool {
	return a.amount < 0
}

func (a Amount) IsPositive() bool {
	return 0 < a.amount
}

// normaliseAmounts returns the amounts of both number that are comparable, ie. they have the same unit are are in the same magnitude.
func normaliseAmounts(a, b Amount) (int64, int64, bool) {
	if a.Unit != b.Unit {
		return 0, 0, false
	}
	for a.digits < b.digits {
		if 0 < a.amount && MaxAmount/10 < a.amount {
			return 0, 0, false // overflow
		} else if a.amount < 0 && a.amount < -MaxAmount/10 {
			return 0, 0, false // underflow
		}
		a.amount *= 10
		a.digits++
	}
	for b.digits < a.digits {
		if 0 < b.amount && MaxAmount/10 < b.amount {
			return 0, 0, false // overflow
		} else if b.amount < 0 && b.amount < -MaxAmount/10 {
			return 0, 0, false // underflow
		}
		b.amount *= 10
		b.digits++
	}
	return a.amount, b.amount, true
}

func (a Amount) Equals(b Amount) bool {
	A, B, ok := normaliseAmounts(a, b)
	if !ok {
		return false
	}
	return A == B
}

func (a Amount) Compare(b Amount) int {
	A, B, ok := normaliseAmounts(a, b)
	if !ok {
		return 0
	} else if A < B {
		return -1
	}
	return 1
}

// bankersRounding performs bankers rounding, with amount the original amount, and prec the number
// of digits to round away. If the last digit is < 5 than the preceding digit stays put, if the
// last digit is > 5, the preceding digit is increased, and when the last digit = 5 the preceding
// digit will increase by 1 only when it is uneven.
func bankersRounding(amount int64, prec int) int64 {
	if prec <= 0 {
		return amount
	}
	shift := int64(0)
	scale := int64Scales[prec]
	if carry := (amount / (scale / 10)) % 10; carry == 5 {
		if isEven := ((amount / scale) % 2) == 0; !isEven {
			shift = scale
		}
	} else if 5 < carry {
		shift = scale
	}
	amount += -(amount % scale) + shift
	return amount
}

// round performs banker's rounding to the given increments
func (a Amount) round(incr int) (Amount, error) {
	prec := AmountPrecision
	switch incr {
	case 0, 1:
		// no-op
	case 10:
		prec++
	case 100:
		prec += 2
	default:
		return Amount{}, fmt.Errorf("unexpected increment: %v", incr)
	}
	a.amount = bankersRounding(a.amount, prec)
	return a, nil
}

// Round performs banker's rounding to the currency's increments
func (a Amount) Round() Amount {
	a, _ = a.round(a.rounding)
	return a
}

func (a Amount) Neg() Amount {
	a.amount = -a.amount // can never overflow
	return a
}

func (a Amount) Abs() Amount {
	if a.amount < 0 {
		return a.Neg()
	}
	return a
}

func (a Amount) MustAdd(b Amount) Amount {
	c, err := a.Add(b)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Amount) Add(b Amount) (Amount, error) {
	if a.Unit != b.Unit {
		if a == ZeroAmount {
			return b, nil
		}
		return Amount{}, fmt.Errorf("currencies don't match: %v != %v", a.Unit, b.Unit)
	} else if 0 < b.amount && MaxAmount-b.amount < a.amount {
		return Amount{}, ErrOverflow
	} else if b.amount < 0 && a.amount < -MaxAmount-b.amount {
		return Amount{}, ErrUnderflow
	}
	a.amount += b.amount
	return a, nil
}

func (a Amount) MustSub(b Amount) Amount {
	c, err := a.Sub(b)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Amount) Sub(b Amount) (Amount, error) {
	if a.Unit != b.Unit {
		if a == ZeroAmount {
			return b.Neg(), nil
		}
		return Amount{}, fmt.Errorf("currencies don't match: %v != %v", a.Unit, b.Unit)
	} else if 0 < b.amount && a.amount < -MaxAmount+b.amount {
		return Amount{}, ErrUnderflow
	} else if b.amount < 0 && MaxAmount+b.amount < a.amount {
		return Amount{}, ErrOverflow
	}
	a.amount -= b.amount
	return a, nil
}

func (a Amount) MustMul(f int) Amount {
	c, err := a.Mul(f)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Amount) Mul(f int) (Amount, error) {
	// TODO: is this right?
	if 1 < f && 0 < a.amount && MaxAmount/int64(f) < a.amount {
		return Amount{}, ErrOverflow
	} else if 1 < f && a.amount < 0 && a.amount < -MaxAmount/int64(f) {
		return Amount{}, ErrUnderflow
	} else if f < -1 && a.amount < 0 && MaxAmount/int64(-f) < -a.amount {
		return Amount{}, ErrOverflow
	} else if f < -1 && 0 < a.amount && -MaxAmount/int64(f) < a.amount {
		return Amount{}, ErrUnderflow
	}
	a.amount *= int64(f)
	return a, nil
}

func (a Amount) Div(f int) Amount {
	if a.amount < MaxAmount/10 {
		a.amount = bankersRounding(a.amount*10/int64(f), 1) / 10
	} else {
		// TODO: to proper rounding
		a.amount /= int64(f)
	}
	return a
}

func (a Amount) DivAmount(b Amount) float64 {
	A, B, ok := normaliseAmounts(a, b)
	if !ok {
		return math.NaN()
	}
	return float64(A) / float64(B)
}

func (a Amount) MustMulf(f float64) Amount {
	c, err := a.Mulf(f)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Amount) Mulf(f float64) (Amount, error) {
	// TODO: is this right?
	incr := 1.0 < f && 0 < a.amount || f < -1.0 && a.amount < 0
	decr := f < -1.0 && 0 < a.amount || 1.0 < f && a.amount < 0
	if incr && float64(MaxAmount) < f*float64(a.amount) {
		return Amount{}, ErrOverflow
	} else if decr && f*float64(a.amount) < float64(-MaxAmount) {
		return Amount{}, ErrUnderflow
	}
	a.amount = int64(math.RoundToEven(float64(a.amount) * f))
	return a, nil
}

func (a Amount) Float64() float64 {
	return float64(a.amount) / math.Pow10(a.digits+AmountPrecision)
}

func (a Amount) Amount() (int64, int) {
	return a.amount, a.digits + AmountPrecision
}

func (a Amount) AmountRounded() (int64, int, error) {
	var err error
	a, err = a.round(1)
	if err != nil {
		return 0, 0, err
	}
	return a.amount / int64Scales[AmountPrecision], a.digits, nil
}

func (a Amount) StringAmount() string {
	var b []byte
	amount, dec := a.Amount()
	b = strconv.AppendNumber(b, amount, dec, 0, 0, '.')

	// remove superfluous trailing zeros
	if 0 < dec {
		for b[len(b)-1] == '0' {
			b = b[:len(b)-1]
		}
		if b[len(b)-1] == '.' {
			b = b[:len(b)-1]
		}
	}
	return string(b)
}

func (a Amount) String() string {
	var b []byte
	amount, dec, err := a.AmountRounded()
	if err != nil {
		return ""
	}
	b = strconv.AppendNumber(b, amount, dec, 3, ',', '.')
	return a.Unit.String() + " " + string(b)
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
		return fmt.Errorf("%v: %v", err, string(b))
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
	return a.Unit.String() + a.StringAmount(), nil
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
	} else if s, ok := value.(string); ok && s == "" {
		n.Amount, n.Valid = Amount{}, false
		return nil
	} else if b, ok := value.([]byte); ok && len(b) == 0 {
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
	return n.Amount.Value()
}

func (n NullAmount) String() string {
	if !n.Valid {
		return ""
	}
	return n.Amount.String()
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
	case "US Dollar":
		s = locale.Currency[unit].Name
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

// Available currency formats. A trailing . will add the appropriate number of decimals for that language/currency. Any additional zeros will indicate the minimum number of decimals, while additional nines indices the maximum number of decimals. Thus "USD 100.09" would always print at least one decimal, but at most two and only if the second decimal is non-zero.
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

	// parse trailing .00 (force decimals) or .99 (allow decimals)
	minDecimals, maxDecimals := 0, f.Amount.digits
	if dot := strings.IndexByte(f.Layout, '.'); dot == len(f.Layout)-1 {
		minDecimals = f.Amount.digits
		f.Layout = f.Layout[:dot]
	} else if dot != -1 {
		maxDecimals = 0
		for _, c := range f.Layout[dot+1:] {
			if c == '0' {
				maxDecimals++
				minDecimals = maxDecimals
			} else if c == '9' {
				maxDecimals++
			} else {
				log.Printf("INFO: locale: unsupported currency format: %v\n", f.Layout)
				break
			}
		}
		f.Layout = f.Layout[:dot]
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
		if f.Amount.IsNegative() {
			pattern = pattern[idx+1:]
			f.Amount = f.Amount.Neg()
		} else {
			pattern = pattern[:idx]
		}
	}

	var amount int64
	if prec := AmountPrecision + f.Amount.digits - maxDecimals; 0 < prec {
		amount = bankersRounding(f.Amount.amount, prec)
		amount /= int64Scales[prec]
	}
	dec := maxDecimals
	for minDecimals < dec && amount%10 == 0 {
		amount /= 10
		dec--
	}

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
