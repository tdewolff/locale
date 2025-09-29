package locale

import (
	"database/sql/driver"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	parseStrconv "github.com/tdewolff/parse/v2/strconv"
)

type Duration time.Duration

// String formats duration in seconds.
func (d Duration) String() string {
	sign := ""
	if int64(d) < 0 {
		sign = "-"
	}
	seconds := int64(d) / 1e9
	fseconds := int64(d) % 1e9
	if fseconds != 0 {
		n := 9
		for (fseconds % 10) == 0 {
			fseconds /= 10
			n--
		}
		return fmt.Sprintf("%s%d.%0*d", sign, seconds, n, fseconds)
	}
	return fmt.Sprintf("%s%d", sign, seconds)
}

func (d Duration) Format(layout string) string {
	return (time.Time{}).Add(time.Duration(d)).Format(layout)
}

func (d *Duration) Scan(isrc interface{}) error {
	var b []byte
	switch src := isrc.(type) {
	case Duration:
		*d = src
		return nil
	case time.Duration:
		*d = Duration(src)
		return nil
	case int64:
		*d = Duration(src * 1e9)
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

	first, n := parseStrconv.ParseUint(b)
	if n == 0 {
		return fmt.Errorf("invalid duration")
	} else if 0 < len(b) && (b[0] == 'h' || b[0] == 'm' || b[0] == 's' || b[0] == 'u' || b[0] == 'n') {
		duration, err := time.ParseDuration(string(b))
		if err != nil {
			return fmt.Errorf("invalid duration: %v", err)
		}
		if neg {
			duration = -duration
		}
		*d = Duration(duration)
		return nil
	}
	b = b[n:]

	var hours, minutes, seconds, fseconds uint64
	if len(b) == 0 {
		seconds = first
	} else if b[0] == '.' {
		seconds = first
		fseconds, n = parseStrconv.ParseUint(b)
		if n != len(b) || 9 < n {
			return fmt.Errorf("invalid duration")
		}
		fseconds *= uint64(int64Scales[9-n])
	} else if b[0] == ':' {
		hours = first
		if n != 2 || 23 < hours {
			return fmt.Errorf("invalid hours")
		}

		if len(b) == 0 || b[0] != ':' {
			return fmt.Errorf("invalid duration")
		}
		b = b[1:]
		minutes, n = parseStrconv.ParseUint(b)
		if n != 2 || 59 < minutes {
			return fmt.Errorf("invalid minutes")
		}
		b = b[n:]

		if len(b) != 0 {
			if b[0] != ':' {
				return fmt.Errorf("invalid duration")
			}
			b = b[1:]
			seconds, n = parseStrconv.ParseUint(b)
			if n != 2 || 59 < seconds {
				return fmt.Errorf("invalid seconds")
			}
			b = b[n:]

			if 0 < len(b) && b[0] == '.' {
				fseconds, n = parseStrconv.ParseUint(b)
				if 9 < n {
					return fmt.Errorf("invalid duration")
				}
				fseconds *= uint64(int64Scales[9-n])
				b = b[n:]
			}
		}
	}
	if len(b) != 0 {
		return fmt.Errorf("invalid duration")
	}
	*d = Duration(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second + time.Duration(fseconds)*time.Nanosecond)
	if neg {
		*d = -*d
	}
	return nil
}

func (d Duration) Value() (driver.Value, error) {
	return int64(math.Round(time.Duration(d).Seconds())), nil
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
	time.Duration
	Layout string
}

func (f DurationFormatter) Format(state fmt.State, verb rune) {
	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}

	var b []byte
	if f.Duration == 0 {
		log.Printf("INFO: locale: unsupported zero duration\n")
		return
	} else if f.Duration < 0 {
		b = append(b, '-')
		f.Duration = -f.Duration
	}

	switch f.Layout {
	case DurationTime:
		hours := int64(f.Duration.Hours())
		minutes := int64(f.Duration.Minutes()) - hours*60
		b = append(b, fmt.Sprintf("%02d:%02d", hours, minutes)...)
		state.Write(b)
		return
	case DurationDigital:
		hours := int64(f.Duration.Hours())
		minutes := int64(f.Duration.Minutes()) - hours*60
		seconds := int64(f.Duration.Seconds()) - hours*3600 - minutes*60
		if 0 < hours {
			b = append(b, fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)...)
		} else {
			b = append(b, fmt.Sprintf("%d:%02d", minutes, seconds)...)
		}
		state.Write(b)
		return
	}

	approximate := strings.HasPrefix(f.Layout, "≈")
	if approximate {
		f.Layout = strings.TrimPrefix(f.Layout, "≈")
	}

	num := int64(f.Duration)
	unitType := []string{"week", "day", "hour", "minute", "second", "millisecond", "microsecond", "nanosecond"}
	unitSize := []int64{7 * 24 * 3600 * 1e9, 24 * 3600 * 1e9, 3600 * 1e9, 60 * 1e9, 1e9, 1e6, 1e3, 1}
	for i := 0; num != 0 && i < len(unitType); i++ {
		if _, ok := locale.Unit["duration-"+unitType[i]]; ok {
			if n := num / unitSize[i]; n != 0 {
				var count Count
				switch f.Layout {
				case DurationLong:
					count = locale.Unit["duration-"+unitType[i]].Long
				case DurationShort:
					count = locale.Unit["duration-"+unitType[i]].Short
				case DurationNarrow:
					count = locale.Unit["duration-"+unitType[i]].Narrow
				default:
					log.Printf("INFO: locale: unsupported duration format: %v\n", f.Layout)
					return
				}

				pattern := count.Other
				if n == 1 {
					pattern = count.One
				}
				pattern = strings.ReplaceAll(pattern, "{0}", fmt.Sprintf("%d", n))
				if 1 < len(b) {
					b = append(b, ' ')
				}
				b = append(b, []byte(pattern)...)
				if approximate {
					break
				}
				num %= unitSize[i]
			}
		}
	}
	state.Write(b)
}

type DurationIntervalFormatter struct {
	Time time.Time
	time.Duration
	Layout string
}

type IntervalSymbol int

const (
	Nanosecond  IntervalSymbol = 0
	Microsecond IntervalSymbol = 1
	Millisecond IntervalSymbol = 2
	Second      IntervalSymbol = 3
	Minute      IntervalSymbol = 4
	Hour        IntervalSymbol = 5
	Day         IntervalSymbol = 6
	Week        IntervalSymbol = 7
	Month       IntervalSymbol = 8
	Year        IntervalSymbol = 9
	Decade      IntervalSymbol = 10
	Century     IntervalSymbol = 11
)

var IntervalSymbols = map[string]IntervalSymbol{
	"nanosecond":  Nanosecond,
	"microsecond": Microsecond,
	"millisecond": Millisecond,
	"second":      Second,
	"minute":      Minute,
	"hour":        Hour,
	"day":         Day,
	"week":        Week,
	"month":       Month,
	"year":        Year,
	"decade":      Decade,
	"century":     Century,
}

func (f DurationIntervalFormatter) Format(state fmt.State, verb rune) {
	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}

	var b []byte
	if f.Duration < 0 {
		b = append(b, '-')
		f.Duration = -f.Duration
	}

	approximate := strings.HasPrefix(f.Layout, "≈")
	if approximate {
		f.Layout = strings.TrimPrefix(f.Layout, "≈")
	}
	minUnit, maxUnit := Nanosecond, Century
	if bracket := strings.IndexByte(f.Layout, '['); bracket != -1 && strings.HasSuffix(f.Layout, "]") {
		fields := strings.Split(f.Layout[bracket+1:len(f.Layout)-1], ",")
		if unit, ok := IntervalSymbols[fields[0]]; ok {
			minUnit = unit
		}
		if unit, ok := IntervalSymbols[fields[1]]; ok {
			maxUnit = unit
		}
		f.Layout = f.Layout[:bracket]
	}

	written := false
	write := func(unit string, n int) {
		written = true
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
		if n == 1 {
			pattern = count.One
		}
		pattern = strings.ReplaceAll(pattern, "{0}", fmt.Sprintf("%d", n))
		if 1 < len(b) {
			b = append(b, ' ')
		}
		b = append(b, []byte(pattern)...)
	}

	start, end := f.Time, f.Time.Add(f.Duration)
	if _, ok := locale.Unit["duration-century"]; ok && minUnit <= Century && Century <= maxUnit {
		n := 0
		for !end.Before(start.AddDate(100, 0, 0)) {
			start = start.AddDate(100, 0, 0)
			n++
		}
		if (approximate || minUnit == Century) && !end.Before(start.AddDate(50, 0, 0)) {
			start = end
			n++
		}
		if 0 < n || minUnit == Century && !written {
			write("century", n)
		}
	}
	if _, ok := locale.Unit["duration-decade"]; ok && minUnit <= Decade && Decade <= maxUnit {
		n := 0
		for !end.Before(start.AddDate(10, 0, 0)) {
			start = start.AddDate(10, 0, 0)
			n++
		}
		if (approximate || minUnit == Decade) && !end.Before(start.AddDate(5, 0, 0)) {
			start = end
			n++
		}
		if 0 < n || minUnit == Decade && !written {
			write("decade", n)
		}
	}
	if _, ok := locale.Unit["duration-year"]; ok && minUnit <= Year && Year <= maxUnit {
		n := 0
		for !end.Before(start.AddDate(1, 0, 0)) {
			start = start.AddDate(1, 0, 0)
			n++
		}
		if (approximate || minUnit == Year) && !end.Before(start.AddDate(0, 6, 0)) {
			start = end
			n++
		}
		if 0 < n || minUnit == Year && !written {
			write("year", n)
		}
	}
	if _, ok := locale.Unit["duration-month"]; ok && minUnit <= Month && Month <= maxUnit {
		n := 0
		for !end.Before(start.AddDate(0, 1, 0)) {
			start = start.AddDate(0, 1, 0)
			n++
		}
		if halfMonth := start.AddDate(0, 1, 0).Sub(start) / 2; (approximate || minUnit == Month) && !end.Before(start.Add(halfMonth)) {
			start = end
			n++
		}
		if 0 < n || minUnit == Month && !written {
			write("month", n)
		}
	}
	if _, ok := locale.Unit["duration-week"]; ok && minUnit <= Week && Week <= maxUnit {
		n := 0
		for !end.Before(start.AddDate(0, 0, 7)) {
			start = start.AddDate(0, 0, 7)
			n++
		}
		if halfWeek := start.AddDate(0, 0, 7).Sub(start) / 2; (approximate || minUnit == Week) && !end.Before(start.Add(halfWeek)) {
			start = end
			n++
		}
		if 0 < n || minUnit == Week && !written {
			write("week", n)
		}
	}
	if _, ok := locale.Unit["duration-day"]; ok && minUnit <= Day && Day <= maxUnit {
		n := 0
		for !end.Before(start.AddDate(0, 0, 1)) {
			start = start.AddDate(0, 0, 1)
			n++
		}
		if (approximate || minUnit == Day) && !end.Before(start.Add(12*time.Hour)) {
			start = end
			n++
		}
		if 0 < n || minUnit == Day && !written {
			write("day", n)
		}
	}

	num := int64(end.Sub(start))
	unitSize := []int64{3600 * 1e9, 60 * 1e9, 1e9, 1e6, 1e3, 1}
	unitSymbol := []IntervalSymbol{Hour, Minute, Second, Millisecond, Microsecond, Nanosecond}
	unitType := []string{"hour", "minute", "second", "millisecond", "microsecond", "nanosecond"}
	for i := 0; num != 0 && i < len(unitSymbol) && minUnit <= unitSymbol[i]; i++ {
		if _, ok := locale.Unit["duration-"+unitType[i]]; ok && unitSymbol[i] <= maxUnit {
			n := num / unitSize[i]
			if (approximate || minUnit == unitSymbol[i]) && unitSize[i]/2 <= num%unitSize[i] {
				n++
			}
			if n != 0 || minUnit == unitSymbol[i] && !written {
				write(unitType[i], int(n))
				if approximate {
					break
				}
				num %= unitSize[i]
			}
		}
	}
	state.Write(b)
	return
}
