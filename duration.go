package locale

import (
	"database/sql/driver"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	parseStrconv "github.com/tdewolff/parse/v2/strconv"
)

type TimeUnitSymbol byte

const (
	Nanosecond TimeUnitSymbol = iota
	Microsecond
	Millisecond
	Second
	Minute
	Hour
	Day
	Week
	Fortnight
	Month
	Quarter
	Year
	Decade
	Century
)

func (u TimeUnitSymbol) Factor() float64 {
	switch u {
	case Nanosecond:
		return 1
	case Microsecond:
		return 1e3
	case Millisecond:
		return 1e6
	case Second:
		return 1e9
	case Minute:
		return 60e9
	case Hour:
		return 3600e9
	case Day:
		return 24 * 3600e9
	case Week:
		return 7 * 24 * 3600e9
	case Fortnight:
		return 14 * 24 * 3600e9
	case Month:
		return 30.437 * 24 * 3600e9
	case Quarter:
		return 3 * 30.437 * 24 * 3600e9
	case Year:
		return 365.2425 * 24 * 3600e9
	case Decade:
		return 10 * 365.2425 * 24 * 3600e9
	case Century:
		return 100 * 365.2425 * 24 * 3600e9
	}
	return 0
}

func (u TimeUnitSymbol) Symbol() byte {
	switch u {
	case Nanosecond:
		return 'n'
	case Microsecond:
		return 'u'
	case Millisecond:
		return 'f'
	case Second:
		return 's'
	case Minute:
		return 'm'
	case Hour:
		return 'h'
	case Day:
		return 'd'
	case Week:
		return 'w'
	case Fortnight:
		return 'F'
	case Month:
		return 'M'
	case Quarter:
		return 'q'
	case Year:
		return 'y'
	case Decade:
		return 'D'
	case Century:
		return 'c'
	}
	return 0
}

func (u TimeUnitSymbol) String() string {
	switch u {
	case Nanosecond:
		return "nanosecond"
	case Microsecond:
		return "microsecond"
	case Millisecond:
		return "millisecond"
	case Second:
		return "second"
	case Minute:
		return "minute"
	case Hour:
		return "hour"
	case Day:
		return "day"
	case Week:
		return "week"
	case Fortnight:
		return "fortnight"
	case Month:
		return "month"
	case Quarter:
		return "quarter"
	case Year:
		return "year"
	case Decade:
		return "decade"
	case Century:
		return "century"
	}
	return ""
}

var TimeUnitSymbols = map[byte]TimeUnitSymbol{
	'n': Nanosecond,
	'u': Microsecond,
	'f': Millisecond,
	's': Second,
	'm': Minute,
	'h': Hour,
	'd': Day,
	'w': Week,
	'F': Fortnight,
	'M': Month,
	'q': Quarter,
	'y': Year,
	'D': Decade,
	'c': Century,
}

var TimeUnitNames = map[string]TimeUnitSymbol{
	"nanosecond":  Nanosecond,
	"microsecond": Microsecond,
	"millisecond": Millisecond,
	"second":      Second,
	"minute":      Minute,
	"hour":        Hour,
	"day":         Day,
	"week":        Week,
	"fortnight":   Fortnight,
	"month":       Month,
	"quarter":     Quarter,
	"year":        Year,
	"decade":      Decade,
	"century":     Century,
}

type Period struct {
	N    int64
	Unit TimeUnitSymbol
}

func (period Period) Empty() bool {
	return period.Unit == 0
}

func (period Period) To(unit TimeUnitSymbol) float64 {
	return float64(period.N) * period.Unit.Factor() / unit.Factor()
}

func (period Period) Duration() time.Duration {
	return time.Duration(period.To(Nanosecond) + 0.5)
}

func (period *Period) Scan(isrc any) error {
	var s string
	switch src := isrc.(type) {
	case []byte:
		s = string(src)
	case string:
		s = src
	default:
		return fmt.Errorf("unexpected type for Period: %T", isrc)
	}

	*period = Period{}
	if 0 < len(s) {
		if n, err := strconv.Atoi(s[:len(s)-1]); err != nil {
			return fmt.Errorf("invalid Period: %s", s)
		} else if unit, ok := TimeUnitSymbols[s[len(s)-1]]; !ok {
			return fmt.Errorf("invalid Period: %s", s)
		} else {
			*period = Period{int64(n), unit}
		}
	}
	return nil
}

func (period Period) Value() (driver.Value, error) {
	return fmt.Sprintf("%d%c", period.N, period.Unit.Symbol()), nil
}

type Duration []Period

func FromDuration(dur time.Duration) Duration {
	var d Duration
	if time.Hour <= dur {
		d = append(d, Period{int64(dur / time.Hour), Hour})
		dur %= time.Hour
	}
	if time.Minute <= dur {
		d = append(d, Period{int64(dur / time.Minute), Minute})
		dur %= time.Minute
	}
	if time.Second <= dur {
		d = append(d, Period{int64(dur / time.Second), Second})
		dur %= time.Second
	}
	if time.Millisecond <= dur {
		d = append(d, Period{int64(dur / time.Millisecond), Millisecond})
		dur %= time.Millisecond
	}
	if time.Microsecond <= dur {
		d = append(d, Period{int64(dur / time.Microsecond), Microsecond})
		dur %= time.Microsecond
	}
	if dur != 0 {
		d = append(d, Period{int64(dur), Nanosecond})
	}
	return d
}

func FromTimeDuration(start time.Time, dur time.Duration) Duration {
	var neg bool
	var d Duration
	end := start.Add(dur)
	if end.Before(start) {
		start, end = end, start
		neg = true
	}

	var centuries, decades, years, months, weeks, days int64
	for !end.Before(start.AddDate(100, 0, 0)) {
		start = start.AddDate(100, 0, 0)
		centuries++
	}
	if 0 < centuries {
		if neg {
			centuries = -centuries
		} else {
		}
		d = append(d, Period{centuries, Century})
	}

	for !end.Before(start.AddDate(10, 0, 0)) {
		start = start.AddDate(10, 0, 0)
		decades++
	}
	if 0 < decades {
		if neg {
			decades = -decades
		}
		d = append(d, Period{decades, Decade})
	}

	for !end.Before(start.AddDate(1, 0, 0)) {
		start = start.AddDate(1, 0, 0)
		years++
	}
	if 0 < years {
		if neg {
			years = -years
		}
		d = append(d, Period{years, Year})
	}

	for !end.Before(start.AddDate(0, 1, 0)) {
		start = start.AddDate(0, 1, 0)
		months++
	}
	if 0 < months {
		if neg {
			months = -months
		}
		d = append(d, Period{months, Month})
	}

	for !end.Before(start.AddDate(0, 0, 7)) {
		start = start.AddDate(0, 0, 7)
		weeks++
	}
	if 0 < weeks {
		if neg {
			weeks = -weeks
		}
		d = append(d, Period{weeks, Week})
	}

	for !end.Before(start.AddDate(0, 0, 1)) {
		start = start.AddDate(0, 0, 1)
		days++
	}
	if 0 < days {
		if neg {
			days = -days
		}
		d = append(d, Period{days, Day})
	}
	if start.Before(end) {
		dt := FromDuration(end.Sub(start))
		if neg {
			for i := range dt {
				dt[i].N = -dt[i].N
			}
		}
		d = append(d, dt...)
	}
	return d
}

func (d Duration) Duration() time.Duration {
	var num float64
	for _, period := range d {
		num += period.To(Nanosecond)
	}
	return time.Duration(num + 0.5)
}

func (d Duration) String() string {
	if len(d) == 0 {
		return ""
	}
	var sb strings.Builder
	if d[0].N < 0 {
		sb.WriteByte('-')
	}
	for _, period := range d {
		if period.N < 0 {
			fmt.Fprintf(&sb, "%d", -period.N)
		} else {
			fmt.Fprintf(&sb, "%d", period.N)
		}
		sb.WriteByte(period.Unit.Symbol())
	}
	return sb.String()
}

func (d *Duration) Scan(isrc interface{}) error {
	var b []byte
	switch src := isrc.(type) {
	case Duration:
		*d = src
		return nil
	case time.Duration:
		*d = FromDuration(src)
		return nil
	case int64:
		*d = FromDuration(time.Duration(src * 1e9))
		return nil
	case []byte:
		b = src
	case string:
		b = []byte(src)
	default:
		return fmt.Errorf("incompatible type for Duration: %T", isrc)
	}

	neg := false
	if 0 < len(b) && b[0] == '-' {
		neg = true
		b = b[1:]
	}

	*d = Duration{}
	for 0 < len(b) {
		num, n := parseStrconv.ParseInt(b)
		if n == 0 {
			return fmt.Errorf("invalid duration")
		} else if neg && 0 < num {
			num = -num
		}

		unit, ok := TimeUnitSymbols[b[n]]
		if !ok {
			return fmt.Errorf("invalid duration")
		}
		*d = append(*d, Period{num, unit})
		b = b[n:]
	}
	return nil
}

func (d Duration) Value() (driver.Value, error) {
	return d.String(), nil
}

// Available duration layouts
const (
	DurationLong    string = "second"
	DurationShort          = "sec"
	DurationNarrow         = "s"
	DurationTime           = "15:04"
	DurationDigital        = "15:04:05"
)

type DurationFormatter struct {
	Duration
	Layout string
}

func (f DurationFormatter) Format(state fmt.State, verb rune) {
	if len(f.Duration) == 0 {
		return
	}

	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}

	neg := f.Duration[0].N < 0
	switch f.Layout {
	case DurationTime:
		dur := f.Duration.Duration()
		hours := int64(dur.Hours())
		minutes := int64(dur.Minutes()) - hours*60
		if neg {
			fmt.Fprintf(state, "-%02d:%02d", hours, minutes)
		} else {
			fmt.Fprintf(state, "%02d:%02d", hours, minutes)
		}
		return
	case DurationDigital:
		dur := f.Duration.Duration()
		hours := int64(dur.Hours())
		minutes := int64(dur.Minutes()) - hours*60
		seconds := int64(dur.Seconds()) - hours*3600 - minutes*60
		if 0 < hours {
			if neg {
				fmt.Fprintf(state, "-%d:%02d:%02d", hours, minutes, seconds)
			} else {
				fmt.Fprintf(state, "%d:%02d:%02d", hours, minutes, seconds)
			}
		} else {
			if neg {
				fmt.Fprintf(state, "-%d:%02d", minutes, seconds)
			} else {
				fmt.Fprintf(state, "%d:%02d", minutes, seconds)
			}
		}
		return
	}

	rounded := strings.HasPrefix(f.Layout, "≈")
	if rounded {
		f.Layout = strings.TrimPrefix(f.Layout, "≈")
	}
	minUnit, maxUnit := Nanosecond, Century
	if bracket := strings.IndexByte(f.Layout, '['); bracket != -1 && strings.HasSuffix(f.Layout, "]") {
		fields := strings.Split(f.Layout[bracket+1:len(f.Layout)-1], ",")
		if unit, ok := TimeUnitNames[fields[0]]; ok {
			minUnit = unit
		} else if len(fields[0]) == 1 {
			if unit, ok := TimeUnitSymbols[fields[0][0]]; ok {
				minUnit = unit
			}
		}
		if unit, ok := TimeUnitNames[fields[1]]; ok {
			maxUnit = unit
		} else if len(fields[1]) == 1 {
			if unit, ok := TimeUnitSymbols[fields[1][0]]; ok {
				maxUnit = unit
			}
		}
		f.Layout = f.Layout[:bracket]
	}

	// filter min/max unit
	var d Duration
	var num float64
	i, j := 0, len(f.Duration)
	for ; i < len(f.Duration); i++ {
		if f.Duration[i].Unit < maxUnit {
			break
		}
		num += f.Duration[i].To(maxUnit) // always integer
	}
	if 0.0 < num {
		d = append(d, Period{int64(num), maxUnit})
	}
	num = 0.0
	for ; i <= j-1; j-- {
		if minUnit < f.Duration[j-1].Unit {
			break
		}
		num += f.Duration[j-1].To(minUnit) // always fraction
	}
	d = append(d, f.Duration[i:j]...)
	if 0.0 < num {
		if minUnit == maxUnit && len(d) == 1 {
			d[0].N += int64(math.Round(num))
		} else {
			d = append(d, Period{int64(math.Round(num)), minUnit})
		}
	}
	if len(d) == 0 {
		d = append(d, Period{0, minUnit})
	}

	// write periods
	var b []byte
	if neg {
		b = append(b, '-')
	}
	if rounded {
		num := float64(d[0].N)
		for i := 1; i < len(d); i++ {
			num += d[i].To(d[0].Unit)
		}
		unit := d[0].Unit
		for unit != Century {
			if factor := d[0].Unit.Factor() / (unit + 1).Factor(); num*factor < 1.0 {
				break
			}
			unit++
		}
		num *= d[0].Unit.Factor() / unit.Factor()
		d[0].Unit = unit
		d[0].N = int64(math.Round(num))
		d = d[:1]
	}
	for _, period := range d {
		unit := period.Unit.String()
		var count Count
		switch f.Layout {
		case DurationLong:
			count = locale.Unit["duration-"+unit].Long
		case DurationShort:
			count = locale.Unit["duration-"+unit].Short
		case DurationNarrow:
			count = locale.Unit["duration-"+unit].Narrow
		default:
			log.Printf("INFO: locale: unsupported duration format: %v\n", f.Layout)
			return
		}

		pattern := count.Other
		if period.N == 1 {
			pattern = count.One
		}
		pattern = strings.ReplaceAll(pattern, "{0}", fmt.Sprintf("%d", period.N))
		if 1 < len(b) {
			b = append(b, ' ')
		}
		b = append(b, []byte(pattern)...)
	}
	state.Write(b)
}
