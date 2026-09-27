package locale

import (
	"database/sql/driver"
	"fmt"
	"log"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/tdewolff/parse/v2/strconv"
)

const AmountMax = math.MaxInt64     // ±9 223 372 036 854 775 807
const AmountInvalid = math.MinInt64 // -9 223 372 036 854 775 808
const AmountPrecision = 3           // extra digits to use for Mul/Div

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

var ZeroAmount = Amount{0, 0}
var UnitAmount = Amount{1, 0}
var InvalidAmount = Amount{AmountInvalid, 0}

type Amount struct {
	num int64 // amount multiplied by 10^dec
	dec int   // decimal digits
}

func ParseAmount(s string, prec int) (Amount, error) {
	return ParseAmountBytes([]byte(s), prec)
}

func ParseAmountBytes(b []byte, prec int) (Amount, error) {
	num, dec, n := strconv.ParseNumber(b, ',', '.')
	if n != len(b) {
		return InvalidAmount, fmt.Errorf("invalid amount: %v", string(b))
	}
	amount := Amount{num, dec}.ShiftTo(prec)
	if amount.IsInvalid() {
		return InvalidAmount, fmt.Errorf("invalid amount: %v prec=%v", string(b), prec)
	}
	return amount, nil
}

func MakeAmount(num int64, dec int) Amount {
	return Amount{num, dec}
}

func AmountFromFloat64(num float64, dec int) (Amount, error) {
	fnum := math.RoundToEven(num * math.Pow10(dec))
	if fnum < float64(-AmountMax) || float64(AmountMax) < fnum {
		return InvalidAmount, fmt.Errorf("invalid amount: %v", fnum)
	}
	return Amount{int64(fnum), dec}, nil
}

func (a Amount) IsInvalid() bool {
	return a.num == AmountInvalid
}

func (a Amount) IsZero() bool {
	return a.num == 0
}

func (a Amount) IsPositive() bool {
	return 0 < a.num
}

func (a Amount) IsNegative() bool {
	return a.num < 0
}

// RoundTo uses banker's rounding towards even integers for the given number of digits after the decimal point. If prec is negative it will round before the decimal point.
func (a Amount) RoundTo(prec int) Amount {
	if a.IsInvalid() || a.dec == prec {
		return a
	} else if a.dec-prec < 0 {
		a.num = 0
		return a
	} else if 18 < a.dec-prec {
		if a.dec-prec == 19 && (a.num < -5000000000000000000 || 5000000000000000000 < a.num) {
			return InvalidAmount
		}
		a.num = 0
		return a
	}

	// banker's rounding, round towards even
	scale := int64Scales[a.dec-prec]
	half := scale >> 1
	carry := a.num % scale
	a.num -= carry
	if half < carry || carry == half && (a.num/scale)%2 == 1 {
		a.num += scale
	} else if carry < -half || carry == -half && (a.num/scale)%2 == -1 {
		a.num -= scale
	}
	return a
}

// TruncTo rounds to the given number of decimals (or digits before the decimal point if negative) to round towards zero.
func (a Amount) TruncTo(prec int) Amount {
	if prec <= a.dec || a.IsInvalid() {
		return a
	} else if 18 < a.dec-prec {
		a.num = 0
		return a
	}
	if prec < a.dec {
		scale := int64Scales[a.dec-prec]
		a.num -= a.num % scale
	}
	return a
}

// FloorTo rounds to the given number of decimals (or digits before the decimal point if negative) to round towards negative infinity.
func (a Amount) FloorTo(prec int) Amount {
	if prec <= a.dec || a.IsInvalid() {
		return a
	} else if 18 < a.dec-prec {
		if a.num < 0 {
			return InvalidAmount
		}
		a.num = 0
		return a
	}
	if prec < a.dec {
		scale := int64Scales[a.dec-prec]
		carry := a.num % scale
		a.num -= carry
		if carry < 0 {
			a.num -= scale
		}
	}
	return a
}

// CeilTo rounds to the given number of decimals (or digits before the decimal point if negative) to round towards infinity.
func (a Amount) CeilTo(prec int) Amount {
	if prec <= a.dec || a.IsInvalid() {
		return a
	} else if 18 < a.dec-prec {
		if 0 < a.num {
			return InvalidAmount
		}
		a.num = 0
		return a
	}
	if prec < a.dec {
		scale := int64Scales[a.dec-prec]
		carry := a.num % scale
		a.num -= carry
		if 0 < carry {
			a.num += scale
		}
	}
	return a
}

// ShiftTo sets the amount of decimals used for calculations without changing the number.
func (a Amount) ShiftTo(dec int) Amount {
	if a.IsInvalid() {
		return a
	} else if a.dec < dec {
		f := int64Scales[dec-a.dec]
		if 0 < a.num && AmountMax/f < a.num {
			return InvalidAmount // overflow
		} else if a.num < 0 && a.num < -AmountMax/f {
			return InvalidAmount // underflow
		}
		a.num *= f
	} else if dec < a.dec {
		if 18 < a.dec-dec {
			return InvalidAmount
		}
		a = a.RoundTo(dec)
		a.num /= int64Scales[a.dec-dec]
	}
	a.dec = dec
	return a
}

// ResetPrecision resets the number of decimals to remove all trailing zero decimals.
func (a Amount) ResetPrecision() Amount {
	for 0 < a.dec && a.num%10 == 0 {
		a.num /= 10
		a.dec--
	}
	return a
}

func (a Amount) Equal(b Amount) bool {
	if a.dec < b.dec {
		a = a.ShiftTo(b.dec)
	} else if b.dec < a.dec {
		b = b.ShiftTo(a.dec)
	}
	return !a.IsInvalid() && a.num == b.num
}

func (a Amount) Compare(b Amount) int {
	if a.dec < b.dec {
		a = a.ShiftTo(b.dec)
	} else if b.dec < a.dec {
		b = b.ShiftTo(a.dec)
	}
	// Invalid is always less than a valid and equal to another invalid amount
	if a.num < b.num {
		return -1
	} else if b.num < a.num {
		return 1
	}
	return 0
}

// Round performs banker's rounding to the currency's increments
func (a Amount) Round() Amount {
	return a.RoundTo(0)
}

func (a Amount) Neg() Amount {
	if !a.IsInvalid() {
		a.num = -a.num // can never overflow
	}
	return a
}

func (a Amount) Abs() Amount {
	if a.num < 0 && !a.IsInvalid() {
		a.num = -a.num // can never overflow
	}
	return a
}

func (a Amount) Add(b Amount) Amount {
	if a.IsInvalid() || b.IsInvalid() {
		return InvalidAmount
	}
	b = b.ShiftTo(a.dec)
	if 0 < b.num && AmountMax-b.num < a.num {
		return InvalidAmount
	} else if b.num < 0 && a.num < -AmountMax-b.num {
		return InvalidAmount
	}
	a.num += b.num
	return a
}

func (a Amount) Sub(b Amount) Amount {
	if a.IsInvalid() || b.IsInvalid() {
		return InvalidAmount
	}
	b = b.ShiftTo(a.dec)
	if 0 < b.num && a.num < -AmountMax+b.num {
		return InvalidAmount
	} else if b.num < 0 && AmountMax+b.num < a.num {
		return InvalidAmount
	}
	a.num -= b.num
	return a
}

func (a Amount) Mul(b Amount) Amount {
	origDec := a.dec
	if a.IsInvalid() || b.IsInvalid() {
		return InvalidAmount
	} else if 1 < b.num && 0 < a.num && AmountMax/b.num < a.num {
		return InvalidAmount
	} else if 1 < b.num && a.num < 0 && a.num < -AmountMax/b.num {
		return InvalidAmount
	} else if b.num < -1 && a.num < 0 && -AmountMax/b.num < -a.num {
		return InvalidAmount
	} else if b.num < -1 && 0 < a.num && -AmountMax/b.num < a.num {
		return InvalidAmount
	}
	a.num *= b.num
	a.dec += b.dec
	a = a.ShiftTo(origDec)
	return a
}

func (a Amount) Muli(i int) Amount {
	return a.Mul(Amount{int64(i), 0})
}

func (a Amount) Mulf(f float64) Amount {
	if a.IsInvalid() {
		return a
	}
	origDec := a.dec
	fnum := math.RoundToEven(float64(a.num) * f * float64(int64Scales[AmountPrecision]))
	if fnum < float64(-AmountMax) || float64(AmountMax) < fnum {
		return InvalidAmount
	}
	a.num = int64(fnum)
	a.dec += AmountPrecision
	a = a.ShiftTo(origDec)
	return a
}

func (a Amount) Div(b Amount) Amount {
	origDec := a.dec
	a = a.ShiftTo(a.dec + b.dec + AmountPrecision)
	if a.IsInvalid() || b.IsInvalid() {
		return InvalidAmount
	}
	a.num /= b.num
	a.dec -= b.dec
	a = a.ShiftTo(origDec)
	return a
}

func (a Amount) Divi(i int) Amount {
	return a.Div(Amount{int64(i), 0})
}

func (a Amount) Divf(f float64) Amount {
	if a.IsInvalid() {
		return a
	}
	origDec := a.dec
	a.num = int64(math.RoundToEven(float64(a.num) / f * float64(int64Scales[AmountPrecision])))
	a.dec += AmountPrecision
	a = a.ShiftTo(origDec)
	return a
}

func (a Amount) Float64() float64 {
	if a.IsInvalid() {
		return math.NaN()
	}
	return float64(a.num) / float64(int64Scales[a.dec])
}

func (a Amount) String() string {
	if a.IsInvalid() {
		return "NaN"
	}
	b := strconv.AppendNumber([]byte{}, a.num, a.dec, 3, ',', '.')
	return string(b)
}

func (a Amount) MinString() string {
	if a.IsInvalid() {
		return "NaN"
	}
	a = a.ResetPrecision() // remove superfluous trailing zeros
	b := strconv.AppendNumber([]byte{}, a.num, a.dec, 3, ',', '.')
	return string(b)
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
	num, dec, n := strconv.ParseNumber(b, ',', '.')
	if n != len(b) {
		return fmt.Errorf("invalid amount: %v", string(b))
	}
	*a = Amount{num, dec}
	return nil
}

func (a Amount) Value() (driver.Value, error) {
	return a.MinString(), nil
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

// AmountFormatter's Layout with a trailing . will add all decimals. Any additional zeros will indicate the minimum number of decimals, while additional nines indices the maximum number of decimals. Thus "100.09" would always print at least one decimal, but at most two and only if the second decimal is non-zero.
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
	minDecimals, maxDecimals := 0, f.Amount.dec
	if dot := strings.IndexByte(f.Layout, '.'); dot == len(f.Layout)-1 {
		minDecimals = f.Amount.dec
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
				log.Printf("INFO: locale: unsupported amount format: %v\n", f.Layout)
				break
			}
		}
		f.Layout = f.Layout[:dot]
	}

	amount := f.Amount.RoundTo(maxDecimals)
	for minDecimals < amount.dec && amount.num%10 == 0 {
		amount.num /= 10
		amount.dec--
	}

	var b []byte
	pattern := locale.DecimalFormat
	for i := 0; i < len(pattern); {
		r, n := utf8.DecodeRuneInString(pattern[i:])
		switch r {
		// TODO: handle negative amounts
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
