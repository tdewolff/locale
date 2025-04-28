package locale

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	parseStrconv "github.com/tdewolff/parse/v2/strconv"
)

// Available time layouts, otherwise falls back to time.Time.Format and translates the individual parts. The order and punctuation may not be in accordance with locale in that case. You can combine any date with time layout by concatenation: {date} + space + {time}
const (
	DateFull   string = "Monday, January 2, 2006"
	DateLong          = "January 2, 2006"
	DateMedium        = "Jan. 2, 2006"
	DateShort         = "1/2/06"
	TimeFull          = "15:04:05 Mountain Standard Time"
	TimeLong          = "15:04:05 MST"
	TimeMedium        = "15:04:05"
	TimeShort         = "15:04"
)

type TimeFormatter struct {
	time.Time
	Layout string
}

func (f TimeFormatter) Format(state fmt.State, verb rune) {
	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}

	idxSep := len(f.Layout)
	var datePattern string
	if strings.HasPrefix(f.Layout, DateFull) {
		datePattern = locale.DateFormat.Full
		idxSep = len(DateFull)
	} else if strings.HasPrefix(f.Layout, DateLong) {
		datePattern = locale.DateFormat.Long
		idxSep = len(DateLong)
	} else if strings.HasPrefix(f.Layout, DateMedium) {
		datePattern = locale.DateFormat.Medium
		idxSep = len(DateMedium)
	} else if strings.HasPrefix(f.Layout, DateShort) {
		datePattern = locale.DateFormat.Short
		idxSep = len(DateShort)
	}

	var timePattern string
	if idxSep < len(f.Layout) {
		idxTime := idxSep
		if strings.HasPrefix(f.Layout[idxSep:], " at ") {
			idxTime += 4
		} else if strings.HasPrefix(f.Layout[idxSep:], ", ") {
			idxTime += 2
		} else if strings.HasPrefix(f.Layout[idxSep:], " ") {
			idxTime += 1
		}

		if idxTime == idxSep {
			datePattern = layoutToPattern(f.Layout)
		} else {
			switch f.Layout[idxTime:] {
			case TimeFull:
				timePattern = locale.TimeFormat.Full
			case TimeLong:
				timePattern = locale.TimeFormat.Long
			case TimeMedium:
				timePattern = locale.TimeFormat.Medium
			case TimeShort:
				timePattern = locale.TimeFormat.Short
			default:
				timePattern = layoutToPattern(f.Layout[idxTime:])
			}
		}
	}

	var pattern string
	if datePattern != "" && timePattern != "" {
		switch f.Layout[:idxSep] {
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
	} else if timePattern != "" {
		pattern = timePattern
	} else {
		pattern = layoutToPattern(f.Layout)
	}

	var b []byte
	for i := 0; i < len(pattern); {
		r, n := utf8.DecodeRuneInString(pattern[i:])
		switch r {
		case '\'':
			j := i + 1
			for j < len(pattern) {
				if pattern[j] == '\'' {
					break
				}
				j++
			}
			b = append(b, pattern[i+1:j]...)
			i = j + 1
		default:
			var m int
			var ok bool
			if b, m, ok = formatDatetimeItem(b, pattern[i:], locale, f.Time); !ok {
				state.Write([]byte(f.Time.Format(f.Layout)))
				return
			} else if m != 0 {
				i += m
			} else {
				b = utf8.AppendRune(b, r)
				i += n
			}
		}
	}
	state.Write(b)
}

type IntervalFormatter struct {
	From, To time.Time
	Layout   string // Go format like TimeFormatter
}

func (f IntervalFormatter) Format(state fmt.State, verb rune) {
	locale := locales["root"]
	if languager, ok := state.(Languager); ok {
		locale = GetLocale(languager.Language())
	}

	layoutPattern := layoutToPattern(f.Layout)

	var greatestDifference string
	if f.From.Year() != f.To.Year() {
		greatestDifference = "y"
	} else if f.From.Month() != f.To.Month() {
		greatestDifference = "M"
	} else if f.From.Day() != f.To.Day() {
		greatestDifference = "d"
	} else if f.From.Hour()/12 != f.To.Hour()/12 {
		greatestDifference = "a"
	} else if f.From.Hour() != f.To.Hour() {
		if strings.IndexByte(layoutPattern, 'H') != -1 {
			greatestDifference = "H"
		} else {
			greatestDifference = "h"
		}
	} else if f.From.Minute() != f.To.Minute() {
		greatestDifference = "m"
	} else if f.From.Second() != f.To.Second() {
		greatestDifference = "s"
	}

	// if pattern exists as interval format, use that
	// otherwise, if it differs only in time, format for datetime where time is an interval format of time or the default
	// otherwise, use the default interval format

	pattern, ok := getIntervalPattern(locale, layoutPattern, greatestDifference)
	if !ok {
		if greatestDifference == "a" || greatestDifference == "H" || greatestDifference == "h" || greatestDifference == "m" || greatestDifference == "s" {
			idxSep := len(f.Layout)
			var datePattern string
			if strings.HasPrefix(f.Layout, DateFull) {
				datePattern = locale.DateFormat.Full
				idxSep = len(DateFull)
			} else if strings.HasPrefix(f.Layout, DateLong) {
				datePattern = locale.DateFormat.Long
				idxSep = len(DateLong)
			} else if strings.HasPrefix(f.Layout, DateMedium) {
				datePattern = locale.DateFormat.Medium
				idxSep = len(DateMedium)
			} else if strings.HasPrefix(f.Layout, DateShort) {
				datePattern = locale.DateFormat.Short
				idxSep = len(DateShort)
			}

			var timePattern string
			if idxSep < len(f.Layout) {
				idxTime := idxSep
				if strings.HasPrefix(f.Layout[idxSep:], " at ") {
					idxTime += 4
				} else if strings.HasPrefix(f.Layout[idxSep:], ", ") {
					idxTime += 2
				} else if strings.HasPrefix(f.Layout[idxSep:], " ") {
					idxTime += 1
				}

				if idxTime == idxSep {
					datePattern = layoutToPattern(f.Layout)
				} else {
					switch f.Layout[idxTime:] {
					case TimeFull:
						timePattern = locale.TimeFormat.Full
					case TimeLong:
						timePattern = locale.TimeFormat.Long
					case TimeMedium:
						timePattern = locale.TimeFormat.Medium
					case TimeShort:
						timePattern = locale.TimeFormat.Short
					default:
						timePattern = layoutToPattern(f.Layout[idxTime:])
					}
				}
			}

			if datePattern != "" && timePattern != "" {
				switch f.Layout[:idxSep] {
				case DateFull:
					pattern = locale.DatetimeFormat.Full
				case DateLong:
					pattern = locale.DatetimeFormat.Long
				case DateMedium:
					pattern = locale.DatetimeFormat.Medium
				case DateShort:
					pattern = locale.DatetimeFormat.Short
				}
				pattern = strings.ReplaceAll(pattern, "{1}", datePattern)
			} else if datePattern != "" {
				pattern = datePattern
			} else if timePattern != "" {
				pattern = "{0}"
			} else {
				pattern = layoutToPattern(f.Layout)
			}

			if greatestDifference == "H" || greatestDifference == "h" {
				if strings.IndexByte(timePattern, 'H') != -1 {
					greatestDifference = "H"
				} else {
					greatestDifference = "h"
				}
			}

			intervalPattern, ok := getIntervalPattern(locale, timePattern, greatestDifference)
			if ok {
				pattern = strings.ReplaceAll(pattern, "{0}", intervalPattern)
			} else {
				pattern = strings.ReplaceAll(pattern, "{0}", locale.DatetimeIntervalFormat[""][""])
				pattern = strings.ReplaceAll(pattern, "{0}", timePattern)
				pattern = strings.ReplaceAll(pattern, "{1}", timePattern)
			}
		} else {
			pattern = locale.DatetimeIntervalFormat[""][""]
			pattern = strings.ReplaceAll(pattern, "{0}", layoutPattern)
			pattern = strings.ReplaceAll(pattern, "{1}", layoutPattern)
		}
	}
	state.Write(formatInterval([]byte{}, pattern, locale, f.From, f.To))
}

func getIntervalPattern(locale Locale, pattern, greatestDifference string) (string, bool) {
	if intervalPatterns, ok := locale.DatetimeIntervalFormat[pattern]; ok {
		intervalPattern, ok := intervalPatterns[greatestDifference]
		if !ok {
			greatestDifference = "s"
			for diff := range intervalPatterns {
				if diff == "y" {
					greatestDifference = diff
					break
				} else if diff == "M" {
					greatestDifference = diff
				} else if greatestDifference != "M" && diff < greatestDifference {
					greatestDifference = diff
				}
			}
			intervalPattern, ok = intervalPatterns[greatestDifference]
		}
		return intervalPattern, ok
	}
	return "", false
}

func formatInterval(b []byte, pattern string, locale Locale, from, to time.Time) []byte {
	handled := map[byte]bool{}
	for i := 0; i < len(pattern); {
		r, n := utf8.DecodeRuneInString(pattern[i:])
		switch r {
		case '\'':
			j := i + 1
			for j < len(pattern) {
				if pattern[j] == '\'' {
					break
				}
				j++
			}
			b = append(b, pattern[i+1:j]...)
			i = j + 1
		default:
			// TODO: doesnt handle repeating mixed format and standalone fields
			t := from
			if handled[pattern[i]] {
				t = to
			}
			handled[pattern[i]] = true

			var m int
			var ok bool
			if b, m, ok = formatDatetimeItem(b, pattern[i:], locale, t); !ok {
				log.Printf("INFO: locale: unsupported date/time format: %v\n", pattern[:m])
			} else if m != 0 {
				i += m
			} else {
				b = utf8.AppendRune(b, r)
				i += n
			}
		}
	}
	return b
}

func formatDatetimeItem(b []byte, pattern string, locale Locale, t time.Time) ([]byte, int, bool) {
	// TODO: handle literal characters (in single quotes)
	switch pattern[0] {
	case 'G', 'y', 'M', 'L', 'E', 'c', 'd', 'h', 'H', 'K', 'k', 'm', 's', 'a', 'b', 'B', 'z', 'v', 'Q':
		n := 1
		for n < len(pattern) && pattern[n] == pattern[0] {
			n++
		}
		// TODO: does not support all patterns
		dayPeriod := 0
		if t.Format("PM") == "PM" {
			dayPeriod = 1
		}
		switch pattern[:n] {
		case "y":
			b = strconv.AppendInt(b, int64(t.Year()), 10)
		case "yy":
			b = t.AppendFormat(b, "06")
		case "yyyy":
			b = t.AppendFormat(b, "2006")
		case "M":
			b = t.AppendFormat(b, "1")
		case "MM":
			b = t.AppendFormat(b, "01")
		case "MMM":
			b = append(b, []byte(locale.MonthSymbol[t.Month()-1].Abbreviated)...)
		case "MMMM":
			b = append(b, []byte(locale.MonthSymbol[t.Month()-1].Wide)...)
		case "MMMMM":
			b = append(b, []byte(locale.MonthSymbol[t.Month()-1].Narrow)...)
		case "d":
			b = t.AppendFormat(b, "2")
		case "dd":
			b = t.AppendFormat(b, "02")
		case "E", "EE", "EEE":
			b = append(b, []byte(locale.DaySymbol[t.Weekday()].Abbreviated)...)
		case "EEEE":
			b = append(b, []byte(locale.DaySymbol[t.Weekday()].Wide)...)
		case "EEEEE":
			b = append(b, []byte(locale.DaySymbol[t.Weekday()].Narrow)...)
		case "a", "aa", "aaa":
			b = append(b, []byte(locale.DayPeriodSymbol[dayPeriod].Abbreviated)...)
		case "aaaa":
			b = append(b, []byte(locale.DayPeriodSymbol[dayPeriod].Wide)...)
		case "aaaaa":
			b = append(b, []byte(locale.DayPeriodSymbol[dayPeriod].Narrow)...)
		case "h":
			b = t.AppendFormat(b, "3")
		case "hh":
			b = t.AppendFormat(b, "03")
		case "H":
			b = strconv.AppendInt(b, int64(t.Hour()), 10)
		case "HH":
			b = t.AppendFormat(b, "15")
		case "m":
			b = t.AppendFormat(b, "4")
		case "mm":
			b = t.AppendFormat(b, "04")
		case "s":
			b = t.AppendFormat(b, "5")
		case "ss":
			b = t.AppendFormat(b, "05")
		case "z", "zz", "zzz":
			b = t.AppendFormat(b, "MST")
		case "zzzz":
			b = t.AppendFormat(b, "MST") // TODO: should be longer: Mountain Standard Time
		case "Z", "ZZ", "ZZZ":
			b = t.AppendFormat(b, "-0700")
		case "ZZZZ":
			if t.Location() == time.UTC {
				b = append(b, []byte("GMT")...)
			} else {
				b = t.AppendFormat(b, "MST")
			}
			b = t.AppendFormat(b, "-07:00")
		case "ZZZZZ":
			b = t.AppendFormat(b, "-07:00:00")
			if bytes.Equal(b[len(b)-3:], []byte(":00")) {
				b = b[:len(b)-3]
			}
		default:
			return b, n, false
		}
		return b, n, true
	}
	return b, 0, true
}

func layoutToPattern(layout string) string {
	// TODO: write unknown character (literal) in single quotes
	sb := strings.Builder{}
	for i := 0; i < len(layout); {
		if strings.HasPrefix(layout[i:], "6") {
			sb.WriteString("y")
			i += 1
		} else if strings.HasPrefix(layout[i:], "06") {
			sb.WriteString("yy")
			i += 2
		} else if strings.HasPrefix(layout[i:], "2006") {
			sb.WriteString("yyyy")
			i += 4
		} else if strings.HasPrefix(layout[i:], "15") {
			sb.WriteString("HH")
			i += 2
		} else if strings.HasPrefix(layout[i:], "1") {
			sb.WriteString("M")
			i += 1
		} else if strings.HasPrefix(layout[i:], "01") {
			sb.WriteString("MM")
			i += 2
		} else if strings.HasPrefix(layout[i:], "January") {
			sb.WriteString("MMMM")
			i += 7
		} else if strings.HasPrefix(layout[i:], "Jan") {
			sb.WriteString("MMM")
			i += 3
		} else if strings.HasPrefix(layout[i:], "J") {
			sb.WriteString("MMMMM")
			i += 1
		} else if strings.HasPrefix(layout[i:], "2") {
			sb.WriteString("d")
			i += 1
		} else if strings.HasPrefix(layout[i:], "02") {
			sb.WriteString("dd")
			i += 2
		} else if strings.HasPrefix(layout[i:], "Monday") {
			sb.WriteString("EEEE")
			i += 6
		} else if strings.HasPrefix(layout[i:], "Mon") {
			sb.WriteString("E")
			i += 3
		} else if strings.HasPrefix(layout[i:], "MST-07:00") {
			sb.WriteString("ZZZZ")
			i += 9
		} else if strings.HasPrefix(layout[i:], "MST") {
			sb.WriteString("z")
			i += 3
		} else if strings.HasPrefix(layout[i:], "Mountain Standard Time") {
			sb.WriteString("zzzz")
			i += 22
		} else if strings.HasPrefix(layout[i:], "M") {
			sb.WriteString("EEEEE")
			i += 1
		} else if strings.HasPrefix(layout[i:], "PM") {
			sb.WriteString("a")
			i += 2
		} else if strings.HasPrefix(layout[i:], "p.m.") {
			sb.WriteString("aaaa")
			i += 4
		} else if strings.HasPrefix(layout[i:], "p. m.") {
			sb.WriteString("aaaaa")
			i += 5
		} else if strings.HasPrefix(layout[i:], "3") {
			// TODO: missing H
			sb.WriteString("h")
			i += 1
		} else if strings.HasPrefix(layout[i:], "03") {
			sb.WriteString("hh")
			i += 2
		} else if strings.HasPrefix(layout[i:], "4") {
			sb.WriteString("m")
			i += 1
		} else if strings.HasPrefix(layout[i:], "04") {
			sb.WriteString("mm")
			i += 2
		} else if strings.HasPrefix(layout[i:], "5") {
			sb.WriteString("s")
			i += 1
		} else if strings.HasPrefix(layout[i:], "05") {
			sb.WriteString("ss")
			i += 2
		} else if strings.HasPrefix(layout[i:], "-0700") {
			sb.WriteString("Z")
			i += 5
		} else if strings.HasPrefix(layout[i:], "-07:00:00") {
			sb.WriteString("ZZZZZ")
			i += 9
		} else if strings.HasPrefix(layout[i:], "-07:00") {
			sb.WriteString("ZZZZZ")
			i += 6
		} else {
			sb.WriteByte(layout[i])
			i += 1
		}
	}
	return sb.String()
}

// scanTime from database
func scanTime(t *time.Time, isrc interface{}) error {
	var b []byte
	switch src := isrc.(type) {
	case time.Time:
		*t = src
		return nil
	case int64:
		*t = time.Unix(src, 0)
		return nil
	case string:
		b = []byte(src)
	case []byte:
		b = src
	default:
		return fmt.Errorf("incompatible type for time.Time: %T", isrc)
	}

	if bytes.Equal(b, []byte("now")) {
		*t = time.Now().UTC()
		return nil
	}

	var year, month, day, hours, minutes, seconds uint64
	var fseconds float64
	year, month, day = 1, 1, 1

	first, n := parseStrconv.ParseUint(b)
	if n == 0 {
		return fmt.Errorf("invalid time")
	}
	b = b[n:]

	if b[0] == '.' {
		seconds = first
		fseconds, n = parseStrconv.ParseFloat(b)
		if n != len(b) {
			return fmt.Errorf("invalid time")
		}
		*t = time.Unix(int64(seconds), int64(fseconds*1e9+0.5))
		return nil
	}

	if b[0] == '-' {
		year = first
		if n != 4 || year == 0 {
			return fmt.Errorf("invalid year")
		}

		if len(b) == 0 || b[0] != '-' {
			return fmt.Errorf("invalid time")
		}
		b = b[1:]
		month, n = parseStrconv.ParseUint(b)
		if n != 2 || month == 0 || 12 < month {
			return fmt.Errorf("invalid month")
		}
		b = b[n:]

		if len(b) == 0 || b[0] != '-' {
			return fmt.Errorf("invalid time")
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
			return fmt.Errorf("invalid time")
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
		return fmt.Errorf("invalid time")
	}
	b = b[1:]
	minutes, n = parseStrconv.ParseUint(b)
	if n != 2 || 59 < minutes {
		return fmt.Errorf("invalid minutes")
	}
	b = b[n:]

	if len(b) != 0 {
		if b[0] != ':' {
			return fmt.Errorf("invalid time")
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
		return fmt.Errorf("invalid time")
	}

	*t = time.Date(int(year), time.Month(month), int(day), int(hours), int(minutes), int(seconds), int(fseconds*1e9+0.5), time.UTC)
	return nil
}
