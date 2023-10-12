package locale

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

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
	datePattern := ""
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

	timePattern := ""
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

	pattern := ""
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
