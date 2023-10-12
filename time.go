package locale

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	parseStrconv "github.com/tdewolff/parse/v2/strconv"
)

var NullTime = time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)

type Date time.Time

func (t Date) String() string {
	return time.Time(t).Format("2006-01-02")
}

func (t *Date) Scan(isrc interface{}) error {
	if err := scanTime((*time.Time)(t), isrc); err != nil {
		return err
	}
	*t = Date((*time.Time)(t).Truncate(24 * time.Hour))
	return nil
}

func (t Date) Value() (driver.Value, error) {
	return t.String(), nil
}

type Time time.Time

func (t Time) String() string {
	format := "15:04"
	if time.Time(t).Nanosecond() != 0 {
		format = "15:04:05.999999999"
	} else if time.Time(t).Second() != 0 {
		format = "15:04:05"
	}
	return time.Time(t).Format(format)
}

func (t *Time) Scan(isrc interface{}) error {
	if err := scanTime((*time.Time)(t), isrc); err != nil {
		return err
	}
	d := (*time.Time)(t)
	*t = Time(time.Date(1, 1, 1, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location()))
	return nil
}

func (t Time) Value() (driver.Value, error) {
	return t.String(), nil
}

type Datetime time.Time

func (t Datetime) String() string {
	format := "2006-01-02 15:04"
	if time.Time(t).Nanosecond() != 0 {
		format = "2006-01-02 15:04:05.999999999"
	} else if time.Time(t).Second() != 0 {
		format = "2006-01-02 15:04:05"
	}
	return time.Time(t).Format(format)
}

func (t *Datetime) Scan(isrc interface{}) error {
	if err := scanTime((*time.Time)(t), isrc); err != nil {
		return err
	}
	return nil
}

func (t Datetime) Value() (driver.Value, error) {
	return t.String(), nil
}

type Duration time.Duration

func (d Duration) String() string {
	var format string
	if time.Duration(d) < 0 {
		d = -d
		format = "-"
	}
	if time.Duration(d) < 24*time.Hour {
		format += "2006-01-02 "
	}
	format += "15:04"
	if time.Duration(d).Nanoseconds() != 0 {
		format += "15:04:05.999999999"
	} else if time.Duration(d).Seconds() != 0 {
		format += "15:04:05"
	}
	return NullTime.Add(time.Duration(d)).Format(format)
}

func (d *Duration) Scan(isrc interface{}) error {
	return scanTime((*time.Duration)(d), isrc)
}

func (d Duration) Value() (driver.Value, error) {
	return d.String(), nil
}

// Available time layouts, otherwise falls back to time.Time.Format and translates the individual parts. The order and punctuation may not be in accordance with locale in that case. You can combine any date with time layout by concatenation: {date} + space + {time}
const (
	DateFull   string = "2006 January 2, Monday"
	DateLong          = "2006 January 2"
	DateMedium        = "2006 Jan 2"
	DateShort         = "2006-01-02"
	TimeFull          = "15:04:05 Mountain Standard Time"
	TimeLong          = "15:04:05 MST"
	TimeMedium        = "15:04:05"
	TimeShort         = "15:04"
)

type TimeFormatter struct {
	time.Time
	layout string
}

func translateGoTime(locale Locale, t time.Time, layout string) []byte {
	b := []byte(t.Format(layout))
	for j := 2; j < len(layout); j++ {
		// TODO: advance pos in b when layout is 1 or 2 without 0 prefix and b has two numbers
		i := len(layout) - j
		n := 0
		var replacement string
		if layout[i:i+2] == "PM" {
			dayPeriod := 0
			if t.Format("PM") == "PM" {
				dayPeriod = 1
			}
			replacement = locale.DayPeriodSymbol[dayPeriod].Abbreviated
			n = 2
		} else if i+3 <= len(layout) && layout[i:i+3] == "Jan" {
			replacement = locale.MonthSymbol[t.Month()-1].Abbreviated
			n = 3
		} else if i+3 <= len(layout) && layout[i:i+3] == "Mon" {
			replacement = locale.DaySymbol[t.Weekday()].Abbreviated
			n = 3
		} else if i+6 <= len(layout) && layout[i:i+6] == "Monday" {
			replacement = locale.DaySymbol[t.Weekday()].Wide
			n = 6
		} else if i+7 <= len(layout) && layout[i:i+7] == "January" {
			replacement = locale.MonthSymbol[t.Month()-1].Wide
			n = 7
		}

		if n != 0 {
			i := len(b) - j
			b = append(b[:i], append([]byte(replacement), b[i+n:]...)...)
		}
	}
	return b
}

func (f TimeFormatter) Format(state fmt.State, verb rune) {
	localeName := "root"
	if languager, ok := state.(Languager); ok {
		localeName = ToLocaleName(languager.Language())
	}
	locale := GetLocale(localeName)

	idxSep := -1
	var datePattern string
	if strings.HasPrefix(f.layout, DateFull) {
		datePattern = locale.DateFormat.Full
		idxSep = len(DateFull)
	} else if strings.HasPrefix(f.layout, DateLong) {
		datePattern = locale.DateFormat.Long
		idxSep = len(DateLong)
	} else if strings.HasPrefix(f.layout, DateMedium) {
		datePattern = locale.DateFormat.Medium
		idxSep = len(DateMedium)
	} else if strings.HasPrefix(f.layout, DateShort) {
		datePattern = locale.DateFormat.Short
		idxSep = len(DateShort)
	}

	var timePattern string
	if idxSep < len(f.layout) {
		if idxSep != -1 {
			if f.layout[idxSep] != ' ' {
				state.Write(translateGoTime(locale, f.Time, f.layout))
				return
			}
		}
		switch f.layout[idxSep+1:] {
		case TimeFull:
			timePattern = locale.TimeFormat.Full
		case TimeLong:
			timePattern = locale.TimeFormat.Long
		case TimeMedium:
			timePattern = locale.TimeFormat.Medium
		case TimeShort:
			timePattern = locale.TimeFormat.Short
		default:
			state.Write(translateGoTime(locale, f.Time, f.layout))
			return
		}
	}

	var pattern string
	if datePattern != "" && timePattern != "" {
		switch f.layout[:idxSep] {
		case DateFull:
			pattern = locale.DatetimeFormat.Full
		case DateLong:
			pattern = locale.DatetimeFormat.Long
		case DateMedium:
			pattern = locale.DatetimeFormat.Medium
		case DateShort:
			pattern = locale.DatetimeFormat.Short
		}
		pattern = strings.ReplaceAll(pattern, "{0}", timePattern)
		pattern = strings.ReplaceAll(pattern, "{1}", datePattern)
	} else if datePattern != "" {
		pattern = datePattern
	} else {
		pattern = timePattern // can be empty
	}

	var b []byte
	for i := 0; i < len(pattern); {
		r, n := utf8.DecodeRuneInString(pattern[i:])
		switch r {
		case 'G', 'y', 'M', 'L', 'E', 'c', 'd', 'h', 'H', 'K', 'k', 'm', 's', 'a', 'b', 'B', 'z', 'v', 'Q':
			j := i + 1
			for j < len(pattern) && pattern[j] == pattern[i] {
				j++
			}
			// TODO: does not support all patterns
			dayPeriod := 0
			if f.Time.Format("PM") == "PM" {
				dayPeriod = 1
			}
			switch pattern[i:j] {
			case "y":
				b = strconv.AppendInt(b, int64(f.Year()), 10)
			case "yy":
				b = f.AppendFormat(b, "06")
			case "yyyy":
				b = f.AppendFormat(b, "2006")
			case "M":
				b = f.AppendFormat(b, "1")
			case "MM":
				b = f.AppendFormat(b, "01")
			case "MMM":
				b = append(b, []byte(locale.MonthSymbol[f.Month()-1].Abbreviated)...)
			case "MMMM":
				b = append(b, []byte(locale.MonthSymbol[f.Month()-1].Wide)...)
			case "MMMMM":
				b = append(b, []byte(locale.MonthSymbol[f.Month()-1].Narrow)...)
			case "d":
				b = f.AppendFormat(b, "2")
			case "dd":
				b = f.AppendFormat(b, "02")
			case "E", "EE", "EEE":
				b = append(b, []byte(locale.DaySymbol[f.Weekday()].Abbreviated)...)
			case "EEEE":
				b = append(b, []byte(locale.DaySymbol[f.Weekday()].Wide)...)
			case "EEEEE":
				b = append(b, []byte(locale.DaySymbol[f.Weekday()].Narrow)...)
			case "a", "aa", "aaa":
				b = append(b, []byte(locale.DayPeriodSymbol[dayPeriod].Abbreviated)...)
			case "aaaa":
				b = append(b, []byte(locale.DayPeriodSymbol[dayPeriod].Wide)...)
			case "aaaaa":
				b = append(b, []byte(locale.DayPeriodSymbol[dayPeriod].Narrow)...)
			case "h":
				b = f.AppendFormat(b, "3")
			case "hh":
				b = f.AppendFormat(b, "03")
			case "H":
				b = strconv.AppendInt(b, int64(f.Hour()), 10)
			case "HH":
				b = f.AppendFormat(b, "15")
			case "m":
				b = f.AppendFormat(b, "4")
			case "mm":
				b = f.AppendFormat(b, "04")
			case "s":
				b = f.AppendFormat(b, "5")
			case "ss":
				b = f.AppendFormat(b, "05")
			case "z", "zz", "zzz":
				b = f.AppendFormat(b, "MST")
			case "Z", "ZZ", "ZZZ":
				b = f.AppendFormat(b, "-0700")
			case "ZZZZ":
				if f.Location() == time.UTC {
					b = append(b, []byte("GMT")...)
				} else {
					b = f.AppendFormat(b, "MST")
				}
				b = f.AppendFormat(b, "-07:00")
			case "ZZZZZ":
				b = f.AppendFormat(b, "-07:00:00")
				if bytes.Equal(b[len(b)-3:], []byte(":00")) {
					b = b[:len(b)-3]
				}
			default:
				log.Printf("INFO: unsupported CLDR date/time format: %v\n", pattern[i:j])
			}
			i += n - 1
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
			b = append(b, pattern[i])
		}
		i += n
	}
	state.Write(b)
}

// scanTime from database
func scanTime(idst interface{}, isrc interface{}) error {
	var b []byte
	var name string
	var t *time.Time
	var d *time.Duration
	switch dst := idst.(type) {
	case *time.Time:
		switch src := isrc.(type) {
		case time.Time:
			*dst = src
			return nil
		case int64:
			*dst = time.Unix(src, 0)
			return nil
		case string:
			b = []byte(src)
		case []byte:
			b = src
		default:
			return fmt.Errorf("incompatible type for time.Time: %T", isrc)
		}
		name = "time"
		t = dst
	case *time.Duration:
		switch src := isrc.(type) {
		case time.Duration:
			*dst = src
			return nil
		case int64:
			*dst = time.Duration(src) * time.Second
			return nil
		case string:
			b = []byte(src)
		case []byte:
			b = src
		default:
			return fmt.Errorf("incompatible type for time.Duration: %T", isrc)
		}
		name = "duration"
		d = dst
	default:
		return fmt.Errorf("incompatible destination type: %T", idst)
	}

	neg := false
	if d == nil && bytes.Equal(b, []byte("now")) {
		*t = time.Now().UTC()
		return nil
	} else if d != nil && 0 < len(b) && (b[0] == '+' || b[0] == '-') {
		neg = b[0] == '-'
		b = b[1:]
	}

	var year, month, day, hours, minutes, seconds uint64
	var fseconds float64
	year, month, day = 1, 1, 1

	first, n := parseStrconv.ParseUint(b)
	if n == 0 {
		return fmt.Errorf("invalid %s", name)
	}
	b = b[n:]

	if b[0] == '.' {
		seconds = first
		fseconds, n = parseStrconv.ParseFloat(b)
		if n != len(b) {
			return fmt.Errorf("invalid %s", name)
		}
		if d == nil {
			*t = time.Unix(int64(seconds), int64(fseconds*1e9+0.5))
		} else {
			*d = time.Duration(seconds)*time.Second + time.Duration(fseconds*1e9+0.5)*time.Nanosecond
			if neg {
				*d = -*d
			}
		}
		return nil
	}

	if b[0] == '-' {
		year = first
		if n != 4 || year == 0 {
			return fmt.Errorf("invalid year")
		}

		if len(b) == 0 || b[0] != '-' {
			return fmt.Errorf("invalid %s", name)
		}
		b = b[1:]
		month, n = parseStrconv.ParseUint(b)
		if n != 2 || month == 0 || 12 < month {
			return fmt.Errorf("invalid month")
		}
		b = b[n:]

		if len(b) == 0 || b[0] != '-' {
			return fmt.Errorf("invalid %s", name)
		}
		b = b[1:]
		day, n = parseStrconv.ParseUint(b)
		if n != 2 || day == 0 || 31 < day {
			return fmt.Errorf("invalid day")
		}
		b = b[n:]

		if len(b) == 0 {
			*t = time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
			return nil
		} else if b[0] != ' ' && b[0] != 'T' {
			return fmt.Errorf("invalid %s", name)
		}
		b = b[1:]

		first, n = parseStrconv.ParseUint(b)
		b = b[n:]
	}

	hours = first
	if n != 2 || 23 < hours {
		return fmt.Errorf("invalid hours")
	}

	if len(b) == 0 || b[0] != ':' {
		return fmt.Errorf("invalid %s", name)
	}
	b = b[1:]
	minutes, n = parseStrconv.ParseUint(b)
	if n != 2 || 59 < minutes {
		return fmt.Errorf("invalid minutes")
	}
	b = b[n:]

	if len(b) != 0 {
		if b[0] != ':' {
			return fmt.Errorf("invalid %s", name)
		}
		b = b[1:]
		seconds, n = parseStrconv.ParseUint(b)
		if n != 2 || 59 < seconds {
			return fmt.Errorf("invalid seconds")
		}
		b = b[n:]

		if 0 < len(b) && b[0] == '.' {
			fseconds, n = parseStrconv.ParseFloat(b)
			b = b[n:]
		}
	}

	if len(b) != 0 {
		return fmt.Errorf("invalid %s", name)
	}

	date := time.Date(int(year), time.Month(month), int(day), int(hours), int(minutes), int(seconds), int(fseconds*1e9+0.5), time.UTC)
	if d == nil {
		*t = date
	} else {
		*d = date.Sub(time.Time{})
		if neg {
			*d = -*d
		}
	}
	return nil
}
