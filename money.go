package locale

import (
	"database/sql/driver"
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"

	"github.com/tdewolff/parse/v2/strconv"
)

// MoneyPrecision is the number of extra decimals after the currency's default number of digits
// Most currencies have 2 digits (cents) and thus will use 5 digits for arithmetics
const MoneyPrecision = 3

type Money struct {
	cur    currency.Unit
	amount Amount
	digits int // decimal digits for display
	incr   int // rounding increment
}

func ParseMoney(s string) (Money, error) {
	return ParseMoneyBytes([]byte(s))
}

func ParseMoneyBytes(b []byte) (Money, error) {
	if len(b) < 4 {
		return Money{}, fmt.Errorf("invalid money: %v", string(b))
	}
	unit, err := currency.ParseISO(string(b[:3]))
	if err != nil {
		return Money{}, fmt.Errorf("%v: %v", err, string(b))
	}
	i := 3
	if b[i] == ' ' {
		i++
	}
	return ParseMoneyAmountBytes(unit, b[i:])
}

func ParseMoneyAmount(unit currency.Unit, s string) (Money, error) {
	return ParseMoneyAmountBytes(unit, []byte(s))
}

func ParseMoneyAmountBytes(unit currency.Unit, b []byte) (Money, error) {
	num, dec, n := strconv.ParseNumber(b, ',', '.')
	if n != len(b) {
		return Money{}, fmt.Errorf("invalid money: %v", string(b))
	}
	return MakeMoney(unit, num, dec)
}

func ParseMoneyIn(tag language.Tag, s string) (Money, error) {
	return ParseMoneyBytesIn(tag, []byte(s))
}

func ParseMoneyBytesIn(tag language.Tag, b []byte) (Money, error) {
	if len(b) < 4 {
		return Money{}, fmt.Errorf("invalid money: %v", string(b))
	}
	unit, err := currency.ParseISO(string(b[:3]))
	if err != nil {
		return Money{}, fmt.Errorf("%v: %v", err, string(b))
	}
	i := 3
	if b[i] == ' ' {
		i++
	}
	return ParseMoneyAmountBytesIn(tag, unit, b[i:])
}

func ParseMoneyAmountIn(tag language.Tag, unit currency.Unit, s string) (Money, error) {
	return ParseMoneyAmountBytesIn(tag, unit, []byte(s))
}

func ParseMoneyAmountBytesIn(tag language.Tag, unit currency.Unit, b []byte) (Money, error) {
	locale := GetLocale(tag)
	num, dec, n := strconv.ParseNumber(b, locale.GroupSymbol, locale.DecimalSymbol)
	if n != len(b) {
		return Money{}, fmt.Errorf("invalid money: %v", string(b))
	}
	return MakeMoney(unit, num, dec)
}

func MustMakeZeroMoney(unit currency.Unit) Money {
	money, err := MakeZeroMoney(unit)
	if err != nil {
		panic(err)
	}
	return money
}

func MakeZeroMoney(unit currency.Unit) (Money, error) {
	cur := GetCurrency(unit)
	if cur.Rounding != 0 && cur.Rounding != 1 && cur.Rounding != 5 && cur.Rounding != 10 && cur.Rounding != 50 && cur.Rounding != 100 {
		return Money{}, fmt.Errorf("unsupported currency rounding: %v", cur.Rounding)
	}
	return Money{unit, Amount{0, cur.Digits + MoneyPrecision}, cur.Digits, cur.Rounding}, nil
}

func MustMakeMoney(unit currency.Unit, num int64, dec int) Money {
	money, err := MakeMoney(unit, num, dec)
	if err != nil {
		panic(err)
	}
	return money
}

// MakeMoney returns a new currency amount in the given unit and an integer number including 'dec' decimal digits.
func MakeMoney(unit currency.Unit, num int64, dec int) (Money, error) {
	cur := GetCurrency(unit)
	if cur.Rounding != 0 && cur.Rounding != 1 && cur.Rounding != 5 && cur.Rounding != 10 && cur.Rounding != 50 && cur.Rounding != 100 {
		return Money{}, fmt.Errorf("unsupported currency rounding: %v", cur.Rounding)
	}
	amount := Amount{num, dec}.ShiftTo(cur.Digits + MoneyPrecision)
	if amount.IsInvalid() {
		return Money{}, fmt.Errorf("invalid amount: %#v", amount)
	}
	return Money{unit, amount, cur.Digits, cur.Rounding}, nil
}

func MustMoneyFromFloat64(unit currency.Unit, f float64) Money {
	m, err := MoneyFromFloat64(unit, f)
	if err != nil {
		panic(err)
	}
	return m
}

func MoneyFromFloat64(unit currency.Unit, f float64) (Money, error) {
	cur := GetCurrency(unit)
	if cur.Rounding != 0 && cur.Rounding != 1 && cur.Rounding != 5 && cur.Rounding != 10 && cur.Rounding != 50 && cur.Rounding != 100 {
		return Money{}, fmt.Errorf("unsupported currency rounding: %v", cur.Rounding)
	}
	amount, err := AmountFromFloat64(f, cur.Digits+MoneyPrecision)
	if err != nil {
		return Money{}, err
	}
	return Money{unit, amount, cur.Digits, cur.Rounding}, nil
}

func (a Money) Currency() currency.Unit {
	return a.cur
}

func (a Money) Amount() Amount {
	return a.amount
}

func (a Money) IsZero() bool {
	return a.amount.IsZero()
}

func (a Money) IsPositive() bool {
	return a.amount.IsPositive()
}

func (a Money) IsNegative() bool {
	return a.amount.IsNegative()
}

func (a Money) Equal(b Money) bool {
	if a.cur != b.cur {
		return false
	}
	return a.amount.Equal(b.amount)
}

func (a Money) ApproxEqual(b Money, prec int) bool {
	if a.cur != b.cur {
		return false
	}
	return a.amount.ShiftTo(a.digits + prec).Equal(b.amount.ShiftTo(a.digits + prec))
}

func (a Money) Compare(b Money) int {
	if a.cur != b.cur {
		return 0
	}
	return a.amount.Compare(b.amount)
}

func (a Money) Neg() Money {
	a.amount = a.amount.Neg()
	return a
}

func (a Money) MustAdd(b Money) Money {
	c, err := a.Add(b)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) Add(b Money) (Money, error) {
	if a.cur != b.cur {
		if b.amount == ZeroAmount {
			return a, nil
		} else if a.amount == ZeroAmount {
			return b, nil
		}
		return Money{}, fmt.Errorf("currencies don't match: %v != %v", a.cur, b.cur)
	}
	a.amount = a.amount.Add(b.amount)
	if a.amount.IsInvalid() {
		return Money{}, fmt.Errorf("invalid operation: %#v + %#v", a, b)
	}
	return a, nil
}

func (a Money) MustSub(b Money) Money {
	c, err := a.Sub(b)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) Sub(b Money) (Money, error) {
	if a.cur != b.cur {
		if b.amount == ZeroAmount {
			return a, nil
		} else if a.amount == ZeroAmount {
			return b.Neg(), nil
		}
		return Money{}, fmt.Errorf("currencies don't match: %v != %v", a.cur, b.cur)
	}
	a.amount = a.amount.Sub(b.amount)
	if a.amount.IsInvalid() {
		return Money{}, fmt.Errorf("invalid operation: %#v - %#v", a, b)
	}
	return a, nil
}

func (a Money) MustMul(amount Amount) Money {
	c, err := a.Mul(amount)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) Mul(amount Amount) (Money, error) {
	c := a
	c.amount = a.amount.Mul(amount)
	if c.amount.IsInvalid() {
		return Money{}, fmt.Errorf("invalid operation: %#v * %#v", a, amount)
	}
	return c, nil
}

func (a Money) MustMuli(i int) Money {
	c, err := a.Muli(i)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) Muli(i int) (Money, error) {
	c := a
	c.amount = a.amount.Muli(i)
	if c.amount.IsInvalid() {
		return Money{}, fmt.Errorf("invalid operation: %#v * %#v", a, i)
	}
	return c, nil
}

func (a Money) MustMulf(f float64) Money {
	c, err := a.Mulf(f)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) Mulf(f float64) (Money, error) {
	c := a
	c.amount = a.amount.Mulf(f)
	if c.amount.IsInvalid() {
		return Money{}, fmt.Errorf("invalid operation: %#v * %#v", a, f)
	}
	return c, nil
}

func (a Money) MustDiv(amount Amount) Money {
	c, err := a.Div(amount)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) Div(amount Amount) (Money, error) {
	c := a
	c.amount = a.amount.Div(amount)
	if c.amount.IsInvalid() {
		return Money{}, fmt.Errorf("invalid operation: %#v / %#v", a, amount)
	}
	return c, nil
}

func (a Money) MustDivi(i int) Money {
	c, err := a.Divi(i)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) Divi(i int) (Money, error) {
	c := a
	c.amount = a.amount.Divi(i)
	if c.amount.IsInvalid() {
		return Money{}, fmt.Errorf("invalid operation: %#v / %#v", a, i)
	}
	return c, nil
}

func (a Money) MustDivf(f float64) Money {
	c, err := a.Divf(f)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) Divf(f float64) (Money, error) {
	c := a
	c.amount = a.amount.Divf(f)
	if c.amount.IsInvalid() {
		return Money{}, fmt.Errorf("invalid operation: %#v / %#v", a, f)
	}
	return c, nil
}

func (a Money) Float64() float64 {
	return a.amount.Float64()
}

func (a Money) String() string {
	return a.cur.String() + a.amount.String()
}

// bankersRounding performs banker's rounding with the given increment (minimal amount for currency)
func (a Money) bankersRounding() (Amount, error) {
	amount := a.amount.ShiftTo(a.digits)
	var prec int
	switch a.incr {
	case 0, 1:
		// no-op
	case 5, 10:
		prec++
	case 50, 100:
		prec += 2
	default:
		return InvalidAmount, fmt.Errorf("unexpected increment: %v", a.incr)
	}
	if a.incr == 5 || a.incr == 50 {
		if amount.num&0x4000000000000000 != 0 {
			return InvalidAmount, fmt.Errorf("invalid operation: round(%#v,%v,%v)", a.amount, a.digits, a.incr)
		}
		amount.num <<= 1
		amount = amount.RoundTo(a.digits - prec)
		amount.num >>= 1
	} else {
		amount = amount.RoundTo(a.digits - prec)
	}
	if amount.IsInvalid() {
		return amount, fmt.Errorf("invalid operation: round(%#v,%v,%v)", a.amount, a.digits, a.incr)
	}
	return amount, nil
}

func (a Money) RoundedAmount() Amount {
	amount, err := a.bankersRounding()
	if err != nil {
		return InvalidAmount
	}
	return amount
}

func (a Money) RoundedString() string {
	amount, err := a.bankersRounding()
	if err != nil {
		return ""
	}
	return a.cur.String() + amount.String()
}

func (a *Money) Scan(isrc interface{}) error {
	var b []byte
	switch src := isrc.(type) {
	case Money:
		*a = src
		return nil
	case []byte:
		b = src
	case string:
		b = []byte(src)
	default:
		return fmt.Errorf("unexpected type for money: %T", isrc)
	}

	money, err := ParseMoneyBytes(b)
	if err != nil {
		return err
	}
	*a = money
	return nil
}

func (a Money) Value() (driver.Value, error) {
	return a.cur.String() + a.amount.MinString(), nil
}

type NullMoney struct {
	Money
	Valid bool
}

// Scan implements the Scanner interface.
func (n *NullMoney) Scan(value any) error {
	if value == nil {
		n.Money, n.Valid = Money{}, false
		return nil
	} else if s, ok := value.(string); ok && s == "" {
		n.Money, n.Valid = Money{}, false
		return nil
	} else if b, ok := value.([]byte); ok && len(b) == 0 {
		n.Money, n.Valid = Money{}, false
		return nil
	}
	n.Valid = true
	return n.Money.Scan(value)
}

// Value implements the driver Valuer interface.
func (n NullMoney) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Money.Value()
}

func (n NullMoney) String() string {
	if !n.Valid {
		return ""
	}
	return n.Money.String()
}

func MoneyAmountRegex(tag language.Tag, unit currency.Unit) string {
	cur := GetCurrency(unit)
	locale := GetLocale(tag)

	sb := &strings.Builder{}
	sb.WriteString("^(?:[0-9]+")
	sb.WriteString(regexp.QuoteMeta(string(locale.GroupSymbol)))
	sb.WriteString(")*[0-9]+")
	if 0 < cur.Digits || cur.Digits == 0 && 1 < cur.Rounding {
		switch cur.Rounding {
		case 100:
			if cur.Digits == 0 {
				sb.WriteString("00")
			} else if cur.Digits == 1 {
				sb.WriteString("0")
			}
		case 50:
			if cur.Digits == 0 {
				sb.WriteString("(0|5)0")
			} else if cur.Digits == 1 {
				sb.WriteString("(0|5)")
			}
		case 10:
			if cur.Digits == 0 {
				sb.WriteString("0")
			}
		case 5:
			if cur.Digits == 0 {
				sb.WriteString("(0|5)")
			}
		}
		if 0 < cur.Digits {
			sb.WriteString("(?:")
			sb.WriteString(regexp.QuoteMeta(string(locale.DecimalSymbol)))
			switch cur.Rounding {
			case 0, 1:
				fmt.Fprintf(sb, "[0-9]{,%d}", cur.Digits)
			case 5:
				if 1 < cur.Digits {
					fmt.Fprintf(sb, "[0-9]{,%d}", cur.Digits-1)
				}
				sb.WriteString("(0|5)")
			case 10:
				if 1 < cur.Digits {
					fmt.Fprintf(sb, "[0-9]{,%d}", cur.Digits-1)
				}
				sb.WriteString("0")
			case 50:
				if 2 < cur.Digits {
					fmt.Fprintf(sb, "[0-9]{,%d}(0|5)0", cur.Digits-2)
				} else if cur.Digits == 2 {
					sb.WriteString("(0|5)0")
				} else {
					sb.WriteString("0")
				}
			case 100:
				if 2 < cur.Digits {
					fmt.Fprintf(sb, "[0-9]{,%d}00", cur.Digits-2)
				} else if cur.Digits == 2 {
					sb.WriteString("00")
				} else {
					sb.WriteString("0")
				}
			default:
				panic(fmt.Sprintf("unexpected increment: %v", cur.Rounding))
			}
			sb.WriteString(")?")
		}
	}
	sb.WriteString("$")
	return sb.String()
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

// Available money formats. A trailing . will add the appropriate number of decimals for that language/currency. Any additional zeros will indicate the minimum number of decimals, while additional nines indices the maximum number of decimals. Thus "USD 100.09" would always print at least one decimal, but at most two and only if the second decimal is non-zero.
// TODO: support accounting formats?
const (
	MoneyAmount   string = "100"
	MoneyISO             = "USD 100"
	MoneyStandard        = "US$ 100"
	MoneyNarrow          = "$100"
)

type MoneyFormatter struct {
	Money
	Layout string
}

func (f MoneyFormatter) Format(state fmt.State, verb rune) {
	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}

	// parse trailing .00 (force decimals) or .99 (allow decimals)
	minDecimals, maxDecimals := 0, f.Money.digits
	if dot := strings.IndexByte(f.Layout, '.'); dot == len(f.Layout)-1 {
		minDecimals = f.Money.digits
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

	unit := f.cur.String()
	var symbol, pattern string
	switch f.Layout {
	case MoneyISO:
		symbol = unit
		pattern = locale.CurrencyFormat.ISO
	case MoneyStandard:
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
	case MoneyNarrow:
		symbol = locale.Currency[unit].Narrow
		pattern = locale.CurrencyFormat.Standard
	case MoneyAmount:
		pattern = locale.CurrencyFormat.Amount
	default:
		log.Printf("INFO: locale: unsupported currency format: %v\n", f.Layout)
	}

	if idx := strings.IndexByte(pattern, ';'); idx != -1 {
		if f.Money.IsNegative() {
			pattern = pattern[idx+1:]
			f.Money = f.Money.Neg()
		} else {
			pattern = pattern[:idx]
		}
	}

	amount := f.Money.amount.RoundTo(maxDecimals)
	for minDecimals < amount.dec && amount.num%10 == 0 {
		amount.num /= 10
		amount.dec--
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
			b = strconv.AppendNumber(b, amount.num, amount.dec, groupSize, locale.GroupSymbol, locale.DecimalSymbol)
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
