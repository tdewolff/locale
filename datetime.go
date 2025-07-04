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

	pattern, datePattern, timePattern := layoutToPatterns(locale, f.Layout)
	pattern = strings.ReplaceAll(pattern, "{0}", timePattern)
	pattern = strings.ReplaceAll(pattern, "{1}", datePattern)

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

	pattern, datePattern, timePattern := layoutToPatterns(locale, f.Layout)

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
		if strings.IndexByte(timePattern, 'H') != -1 {
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

	fullPattern := pattern
	fullPattern = strings.ReplaceAll(fullPattern, "{0}", timePattern)
	fullPattern = strings.ReplaceAll(fullPattern, "{1}", datePattern)
	intervalPattern, ok := getIntervalPattern(locale, fullPattern, greatestDifference)
	if !ok {
		if greatestDifference == "a" || greatestDifference == "H" || greatestDifference == "h" || greatestDifference == "m" || greatestDifference == "s" {
			// date pattern displayed once with interval in time
			if greatestDifference == "H" || greatestDifference == "h" {
				if strings.IndexByte(timePattern, 'H') != -1 {
					greatestDifference = "H"
				} else {
					greatestDifference = "h"
				}
			}

			intervalPattern = pattern
			intervalPattern = strings.ReplaceAll(intervalPattern, "{1}", datePattern)

			timeIntervalPattern, ok := getIntervalPattern(locale, timePattern, greatestDifference)
			if ok {
				intervalPattern = strings.ReplaceAll(intervalPattern, "{0}", timeIntervalPattern)
			} else {
				intervalPattern = strings.ReplaceAll(intervalPattern, "{0}", locale.DatetimeIntervalFormat[""][""])
				intervalPattern = strings.ReplaceAll(intervalPattern, "{0}", timePattern)
				intervalPattern = strings.ReplaceAll(intervalPattern, "{1}", timePattern)
			}
		} else {
			intervalPattern = locale.DatetimeIntervalFormat[""][""]
			intervalPattern = strings.ReplaceAll(intervalPattern, "{0}", fullPattern)
			intervalPattern = strings.ReplaceAll(intervalPattern, "{1}", fullPattern)
		}
	}
	state.Write(formatInterval([]byte{}, intervalPattern, locale, f.From, f.To))
}

func getIntervalPattern(locale Locale, pattern, greatestDifference string) (string, bool) {
	id := ""
	substitutions := map[string]string{}
	symbolLists := [][]string{{"G"}, {"y"}, {"MMMM", "MMM", "M"}, {"E"}, {"d"}, {"B"}, {"h"}, {"H"}, {"m"}, {"s"}, {"v", "z"}}
	for _, symbolList := range symbolLists {
		for _, symbol := range symbolList {
			if idx := strings.Index(pattern, symbol); idx != -1 {
				if symbol == "z" {
					start, end := idx, idx+1
					for end < len(pattern) && pattern[end] == 'z' {
						end++
					}
					substitutions["v"] = pattern[start:end]
					symbol = "v"
				}
				id += symbol
				break
			}
		}
	}

	if intervalPatterns, ok := locale.DatetimeIntervalFormat[id]; ok {
		if id == "" {
			intervalPattern, ok := intervalPatterns[""]
			return intervalPattern, ok
		}

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

		for from, to := range substitutions {
			intervalPattern = strings.ReplaceAll(intervalPattern, from, to)
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
				log.Printf("INFO: locale: unsupported date/time format: %v\n", pattern[i:i+m])
				i += m
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

func getTimezone(locale Locale, t time.Time) string {
	timezone := t.Location().String()
	if alias, ok := timezoneAliases[timezone]; ok {
		timezone = alias
	}
	return timezone
}

func is12Hours(locale Locale) bool {
	return strings.IndexByte(locale.TimeFormat.Full, 'h') != -1
}

func formatDatetimeItem(b []byte, pattern string, locale Locale, t time.Time) ([]byte, int, bool) {
	// TODO: handle literal characters (in single quotes)
	switch pattern[0] {
	case 'G', 'y', 'M', 'L', 'E', 'c', 'd', 'h', 'H', 'K', 'k', 'm', 's', 'a', 'b', 'B', 'z', 'Z', 'v', 'V', 'O', 'x', 'X', 'Q':
		n := 1
		for n < len(pattern) && pattern[n] == pattern[0] {
			n++
		}
		dayPeriod := 0
		if t.Format("PM") == "PM" {
			dayPeriod = 1
		}

		// TODO: does not support all patterns
		symbol := pattern[:n]
	TrySymbol:
		switch symbol {
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
		case "v":
			if metazone, ok := locale.Metazones[metazones[getTimezone(locale, t)]]; ok && metazone.Generic.Short != "" {
				b = append(b, metazone.Generic.Short...)
			} else {
				symbol = "O" // should try VVVV and otherwise O
				goto TrySymbol
			}
		case "vvvv":
			if metazone, ok := locale.Metazones[metazones[getTimezone(locale, t)]]; ok && metazone.Generic.Long != "" {
				b = append(b, metazone.Generic.Long...)
			} else {
				symbol = "OOOO" // should try VVVV and otherwise OOOO
				goto TrySymbol
			}
		case "V":
			zone, _ := t.Zone()
			b = append(b, zone...)
		case "VV":
			b = append(b, getTimezone(locale, t)...)
		case "VVV":
			if city, ok := locale.TimezoneCity[getTimezone(locale, t)]; ok {
				b = append(b, city...)
			} else {
				b = append(b, locale.TimezoneCity["Etc/Unknown"]...)
			}
		case "VVVV":
			symbol = "OOOO" // TODO: VVVV values don't exist in CLDR database?
			goto TrySymbol
		case "z", "zz", "zzz":
			if metazone, ok := locale.Metazones[metazones[getTimezone(locale, t)]]; ok && (t.IsDST() && metazone.Daylight.Short != "" || !t.IsDST() && metazone.Standard.Short != "") {
				if t.IsDST() {
					b = append(b, metazone.Daylight.Short...)
				} else {
					b = append(b, metazone.Standard.Short...)
				}
			} else {
				symbol = "O"
				goto TrySymbol
			}
		case "zzzz":
			if metazone, ok := locale.Metazones[metazones[getTimezone(locale, t)]]; ok && (t.IsDST() && metazone.Daylight.Long != "" || !t.IsDST() && metazone.Standard.Long != "") {
				if t.IsDST() {
					b = append(b, metazone.Daylight.Long...)
				} else {
					b = append(b, metazone.Standard.Long...)
				}
			} else {
				symbol = "O" // should try specific location format first, but doensn't exist in CLDR database?
				goto TrySymbol
			}
		case "Z", "ZZ", "ZZZ", "xxxx":
			b = t.AppendFormat(b, "-0700")
		case "O":
			if t.Location() == time.UTC {
				b = append(b, []byte("GMT")...)
			} else {
				b = t.AppendFormat(b, "MST")
			}
			b = t.AppendFormat(b, "-7")
		case "ZZZZ", "OOOO":
			if t.Location() == time.UTC {
				b = append(b, []byte("GMT")...)
			} else {
				b = t.AppendFormat(b, "MST")
			}
			b = t.AppendFormat(b, "-07:00")
		case "ZZZZZ", "XXXXX":
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

func layoutToPattern(locale Locale, layout string) string {
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
			if is12Hours(locale) {
				sb.WriteString("hh")
			} else {
				sb.WriteString("HH")
			}
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
		} else if strings.HasPrefix(layout[i:], "MT") {
			sb.WriteString("v")
			i += 2
		} else if strings.HasPrefix(layout[i:], "Mountain Time") {
			sb.WriteString("vvvv")
			i += 13
		} else if strings.HasPrefix(layout[i:], "America/Phoenix") {
			sb.WriteString("VV")
			i += 15
		} else if strings.HasPrefix(layout[i:], "Phoenix Time") {
			sb.WriteString("VVVV")
			i += 12
		} else if strings.HasPrefix(layout[i:], "Phoenix") {
			sb.WriteString("VVV")
			i += 7
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
			if is12Hours(locale) {
				sb.WriteString("H")
			} else {
				sb.WriteString("h")
			}
			i += 1
		} else if strings.HasPrefix(layout[i:], "03") {
			if is12Hours(locale) {
				sb.WriteString("HH")
			} else {
				sb.WriteString("hh")
			}
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
		} else if strings.HasPrefix(layout[i:], "GMT-7:00") {
			sb.WriteString("ZZZZ")
			i += 8
		} else if strings.HasPrefix(layout[i:], "GMT-7") {
			sb.WriteString("O")
			i += 5
		} else if strings.HasPrefix(layout[i:], "GMT-07:00") {
			sb.WriteString("OOOO")
			i += 9
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

func layoutToPatterns(locale Locale, layout string) (string, string, string) {
	idxSep := len(layout)
	var datePattern string
	if strings.HasPrefix(layout, DateFull) {
		datePattern = locale.DateFormat.Full
		idxSep = len(DateFull)
	} else if strings.HasPrefix(layout, DateLong) {
		datePattern = locale.DateFormat.Long
		idxSep = len(DateLong)
	} else if strings.HasPrefix(layout, DateMedium) {
		datePattern = locale.DateFormat.Medium
		idxSep = len(DateMedium)
	} else if strings.HasPrefix(layout, DateShort) {
		datePattern = locale.DateFormat.Short
		idxSep = len(DateShort)
	}

	var timePattern string
	if idxSep < len(layout) {
		idxTime := idxSep
		if strings.HasPrefix(layout[idxSep:], " at ") {
			idxTime += 4
		} else if strings.HasPrefix(layout[idxSep:], ", ") {
			idxTime += 2
		} else if strings.HasPrefix(layout[idxSep:], " ") {
			idxTime += 1
		}

		if idxTime == idxSep {
			datePattern = layoutToPattern(locale, layout)
		} else {
			switch layout[idxTime:] {
			case TimeFull:
				timePattern = locale.TimeFormat.Full
			case TimeLong:
				timePattern = locale.TimeFormat.Long
			case TimeMedium:
				timePattern = locale.TimeFormat.Medium
			case TimeShort:
				timePattern = locale.TimeFormat.Short
			default:
				timePattern = layoutToPattern(locale, layout[idxTime:])
			}
		}
	}

	var datetimePattern string
	if datePattern != "" && timePattern != "" {
		switch layout[:idxSep] {
		case DateFull:
			datetimePattern = locale.DatetimeFormat.Full
		case DateLong:
			datetimePattern = locale.DatetimeFormat.Long
		case DateMedium:
			datetimePattern = locale.DatetimeFormat.Medium
		case DateShort:
			datetimePattern = locale.DatetimeFormat.Short
		}
	} else if datePattern != "" {
		datetimePattern = "{1}"
	} else if timePattern != "" {
		datetimePattern = "{0}"
	} else {
		datetimePattern = layoutToPattern(locale, layout)
	}
	return datetimePattern, datePattern, timePattern
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

type TimezoneFormatter struct {
	*time.Location
}

func (f TimezoneFormatter) Format(state fmt.State, verb rune) {
	//locale := locales["root"]
	//if languager, ok := state.(Languager); ok {
	//	locale = GetLocale(languager.Language())
	//}
	state.Write([]byte(f.Location.String()))
}

// from https://github.com/arp242/tz/blob/3c7bf612261228ea207792aef3a725c2fec518c6/alias.go
var timezoneAliases = map[string]string{
	// Not in the tzdb and "deprecated", but some browsers send this.
	"CET": "Europe/Paris",
	"EET": "Europe/Sofia",
	"EST": "America/Cancun",
	"HST": "Pacific/Honolulu",
	"MET": "Europe/Paris",
	"MST": "America/Phoenix",
	"WET": "Europe/Lisbon",
	"PST": "America/Los_Angeles",

	// TODO
	// Etc/GMT-14
	// Etc/GMT-13
	// Etc/GMT-12
	// Etc/GMT-11
	// Etc/GMT-10
	// Etc/GMT-9
	// Etc/GMT-8
	// Etc/GMT-7
	// Etc/GMT-6
	// Etc/GMT-5
	// Etc/GMT-4
	// Etc/GMT-3
	// Etc/GMT-2
	// Etc/GMT-1
	// Etc/GMT+1
	// Etc/GMT+2
	// Etc/GMT+3
	// Etc/GMT+4
	// Etc/GMT+5
	// Etc/GMT+6
	// Etc/GMT+7
	// Etc/GMT+8
	// Etc/GMT+9
	// Etc/GMT+10
	// Etc/GMT+11
	// Etc/GMT+12

	// Extracted from tzdb with:
	// grep -h '^Link' *.zi | sed -E 's/\s+#.*//; s/\s+/ /g' | sort -u | sed -E 's/Link (.*?) (.*?)/"\2": "\1",/' |
	"Europe/Kiev": "Europe/Kyiv",

	"Africa/Bamako":                    "Africa/Abidjan",
	"Africa/Banjul":                    "Africa/Abidjan",
	"Africa/Conakry":                   "Africa/Abidjan",
	"Africa/Dakar":                     "Africa/Abidjan",
	"Africa/Freetown":                  "Africa/Abidjan",
	"Africa/Lome":                      "Africa/Abidjan",
	"Africa/Nouakchott":                "Africa/Abidjan",
	"Africa/Ouagadougou":               "Africa/Abidjan",
	"Africa/Timbuktu":                  "Africa/Abidjan",
	"Atlantic/St_Helena":               "Africa/Abidjan",
	"Egypt":                            "Africa/Cairo",
	"Africa/Maseru":                    "Africa/Johannesburg",
	"Africa/Mbabane":                   "Africa/Johannesburg",
	"Africa/Bangui":                    "Africa/Lagos",
	"Africa/Brazzaville":               "Africa/Lagos",
	"Africa/Douala":                    "Africa/Lagos",
	"Africa/Kinshasa":                  "Africa/Lagos",
	"Africa/Libreville":                "Africa/Lagos",
	"Africa/Luanda":                    "Africa/Lagos",
	"Africa/Malabo":                    "Africa/Lagos",
	"Africa/Niamey":                    "Africa/Lagos",
	"Africa/Porto-Novo":                "Africa/Lagos",
	"Africa/Blantyre":                  "Africa/Maputo",
	"Africa/Bujumbura":                 "Africa/Maputo",
	"Africa/Gaborone":                  "Africa/Maputo",
	"Africa/Harare":                    "Africa/Maputo",
	"Africa/Kigali":                    "Africa/Maputo",
	"Africa/Lubumbashi":                "Africa/Maputo",
	"Africa/Lusaka":                    "Africa/Maputo",
	"Africa/Addis_Ababa":               "Africa/Nairobi",
	"Africa/Asmara":                    "Africa/Nairobi",
	"Africa/Asmera":                    "Africa/Nairobi",
	"Africa/Dar_es_Salaam":             "Africa/Nairobi",
	"Africa/Djibouti":                  "Africa/Nairobi",
	"Africa/Kampala":                   "Africa/Nairobi",
	"Africa/Mogadishu":                 "Africa/Nairobi",
	"Indian/Antananarivo":              "Africa/Nairobi",
	"Indian/Comoro":                    "Africa/Nairobi",
	"Indian/Mayotte":                   "Africa/Nairobi",
	"Libya":                            "Africa/Tripoli",
	"America/Atka":                     "America/Adak",
	"US/Aleutian":                      "America/Adak",
	"US/Alaska":                        "America/Anchorage",
	"America/Buenos_Aires":             "America/Argentina/Buenos_Aires",
	"America/Argentina/ComodRivadavia": "America/Argentina/Catamarca",
	"America/Catamarca":                "America/Argentina/Catamarca",
	"America/Cordoba":                  "America/Argentina/Cordoba",
	"America/Rosario":                  "America/Argentina/Cordoba",
	"America/Jujuy":                    "America/Argentina/Jujuy",
	"America/Mendoza":                  "America/Argentina/Mendoza",
	"America/Coral_Harbour":            "America/Atikokan",
	"US/Central":                       "America/Chicago",
	"America/Aruba":                    "America/Curacao",
	"America/Kralendijk":               "America/Curacao",
	"America/Lower_Princes":            "America/Curacao",
	"America/Shiprock":                 "America/Denver",
	"Navajo":                           "America/Denver",
	"US/Mountain":                      "America/Denver",
	"US/Michigan":                      "America/Detroit",
	"Canada/Mountain":                  "America/Edmonton",
	"Canada/Atlantic":                  "America/Halifax",
	"Cuba":                             "America/Havana",
	"America/Fort_Wayne":               "America/Indiana/Indianapolis",
	"America/Indianapolis":             "America/Indiana/Indianapolis",
	"US/East-Indiana":                  "America/Indiana/Indianapolis",
	"America/Knox_IN":                  "America/Indiana/Knox",
	"US/Indiana-Starke":                "America/Indiana/Knox",
	"Jamaica":                          "America/Jamaica",
	"America/Louisville":               "America/Kentucky/Louisville",
	"US/Pacific":                       "America/Los_Angeles",
	"Brazil/West":                      "America/Manaus",
	"Mexico/BajaSur":                   "America/Mazatlan",
	"Mexico/General":                   "America/Mexico_City",
	"US/Eastern":                       "America/New_York",
	"Brazil/DeNoronha":                 "America/Noronha",
	"America/Cayman":                   "America/Panama",
	"US/Arizona":                       "America/Phoenix",
	"America/Anguilla":                 "America/Port_of_Spain",
	"America/Antigua":                  "America/Port_of_Spain",
	"America/Dominica":                 "America/Port_of_Spain",
	"America/Grenada":                  "America/Port_of_Spain",
	"America/Guadeloupe":               "America/Port_of_Spain",
	"America/Marigot":                  "America/Port_of_Spain",
	"America/Montserrat":               "America/Port_of_Spain",
	"America/St_Barthelemy":            "America/Port_of_Spain",
	"America/St_Kitts":                 "America/Port_of_Spain",
	"America/St_Lucia":                 "America/Port_of_Spain",
	"America/St_Thomas":                "America/Port_of_Spain",
	"America/St_Vincent":               "America/Port_of_Spain",
	"America/Tortola":                  "America/Port_of_Spain",
	"America/Virgin":                   "America/Port_of_Spain",
	"Canada/Saskatchewan":              "America/Regina",
	"America/Porto_Acre":               "America/Rio_Branco",
	"Brazil/Acre":                      "America/Rio_Branco",
	"Chile/Continental":                "America/Santiago",
	"Brazil/East":                      "America/Sao_Paulo",
	"Canada/Newfoundland":              "America/St_Johns",
	"America/Ensenada":                 "America/Tijuana",
	"America/Santa_Isabel":             "America/Tijuana",
	"Mexico/BajaNorte":                 "America/Tijuana",
	"America/Montreal":                 "America/Toronto",
	"Canada/Eastern":                   "America/Toronto",
	"Canada/Pacific":                   "America/Vancouver",
	"Canada/Yukon":                     "America/Whitehorse",
	"Canada/Central":                   "America/Winnipeg",
	"Asia/Ashkhabad":                   "Asia/Ashgabat",
	"Asia/Phnom_Penh":                  "Asia/Bangkok",
	"Asia/Vientiane":                   "Asia/Bangkok",
	"Asia/Dacca":                       "Asia/Dhaka",
	"Asia/Muscat":                      "Asia/Dubai",
	"Asia/Saigon":                      "Asia/Ho_Chi_Minh",
	"Hongkong":                         "Asia/Hong_Kong",
	"Asia/Tel_Aviv":                    "Asia/Jerusalem",
	"Israel":                           "Asia/Jerusalem",
	"Asia/Katmandu":                    "Asia/Kathmandu",
	"Asia/Calcutta":                    "Asia/Kolkata",
	"Asia/Macao":                       "Asia/Macau",
	"Asia/Ujung_Pandang":               "Asia/Makassar",
	"Europe/Nicosia":                   "Asia/Nicosia",
	"Asia/Bahrain":                     "Asia/Qatar",
	"Asia/Aden":                        "Asia/Riyadh",
	"Asia/Kuwait":                      "Asia/Riyadh",
	"ROK":                              "Asia/Seoul",
	"Asia/Chongqing":                   "Asia/Shanghai",
	"Asia/Chungking":                   "Asia/Shanghai",
	"Asia/Harbin":                      "Asia/Shanghai",
	"PRC":                              "Asia/Shanghai",
	"Singapore":                        "Asia/Singapore",
	"ROC":                              "Asia/Taipei",
	"Iran":                             "Asia/Tehran",
	"Asia/Thimbu":                      "Asia/Thimphu",
	"Japan":                            "Asia/Tokyo",
	"Asia/Ulan_Bator":                  "Asia/Ulaanbaatar",
	"Asia/Kashgar":                     "Asia/Urumqi",
	"Asia/Rangoon":                     "Asia/Yangon",
	"Atlantic/Faeroe":                  "Atlantic/Faroe",
	"Iceland":                          "Atlantic/Reykjavik",
	"Australia/South":                  "Australia/Adelaide",
	"Australia/Queensland":             "Australia/Brisbane",
	"Australia/Yancowinna":             "Australia/Broken_Hill",
	"Australia/North":                  "Australia/Darwin",
	"Australia/Tasmania":               "Australia/Hobart",
	"Australia/LHI":                    "Australia/Lord_Howe",
	"Australia/Victoria":               "Australia/Melbourne",
	"Australia/West":                   "Australia/Perth",
	"Australia/ACT":                    "Australia/Sydney",
	"Australia/Canberra":               "Australia/Sydney",
	"Australia/NSW":                    "Australia/Sydney",
	"Etc/GMT+0":                        "Etc/GMT",
	"Etc/GMT-0":                        "Etc/GMT",
	"Etc/GMT0":                         "Etc/GMT",
	"Etc/Greenwich":                    "Etc/GMT",
	"GMT":                              "Etc/GMT",
	"GMT+0":                            "Etc/GMT",
	"GMT-0":                            "Etc/GMT",
	"GMT0":                             "Etc/GMT",
	"Greenwich":                        "Etc/GMT",
	"Etc/UCT":                          "Etc/UTC",
	"Etc/Universal":                    "Etc/UTC",
	"Etc/Zulu":                         "Etc/UTC",
	"UCT":                              "Etc/UTC",
	"UTC":                              "Etc/UTC",
	"Universal":                        "Etc/UTC",
	"Zulu":                             "Etc/UTC",
	"Europe/Ljubljana":                 "Europe/Belgrade",
	"Europe/Podgorica":                 "Europe/Belgrade",
	"Europe/Sarajevo":                  "Europe/Belgrade",
	"Europe/Skopje":                    "Europe/Belgrade",
	"Europe/Zagreb":                    "Europe/Belgrade",
	"Europe/Tiraspol":                  "Europe/Chisinau",
	"Eire":                             "Europe/Dublin",
	"Europe/Mariehamn":                 "Europe/Helsinki",
	"Asia/Istanbul":                    "Europe/Istanbul",
	"Turkey":                           "Europe/Istanbul",
	"Portugal":                         "Europe/Lisbon",
	"Europe/Belfast":                   "Europe/London",
	"Europe/Guernsey":                  "Europe/London",
	"Europe/Isle_of_Man":               "Europe/London",
	"Europe/Jersey":                    "Europe/London",
	"GB":                               "Europe/London",
	"GB-Eire":                          "Europe/London",
	"W-SU":                             "Europe/Moscow",
	"Arctic/Longyearbyen":              "Europe/Oslo",
	"Atlantic/Jan_Mayen":               "Europe/Oslo",
	"Europe/Bratislava":                "Europe/Prague",
	"Europe/San_Marino":                "Europe/Rome",
	"Europe/Vatican":                   "Europe/Rome",
	"Poland":                           "Europe/Warsaw",
	"Europe/Busingen":                  "Europe/Zurich",
	"Europe/Vaduz":                     "Europe/Zurich",
	"Antarctica/McMurdo":               "Pacific/Auckland",
	"Antarctica/South_Pole":            "Pacific/Auckland",
	"NZ":                               "Pacific/Auckland",
	"NZ-CHAT":                          "Pacific/Chatham",
	"Pacific/Truk":                     "Pacific/Chuuk",
	"Pacific/Yap":                      "Pacific/Chuuk",
	"Chile/EasterIsland":               "Pacific/Easter",
	"Pacific/Saipan":                   "Pacific/Guam",
	"Pacific/Johnston":                 "Pacific/Honolulu",
	"US/Hawaii":                        "Pacific/Honolulu",
	"Kwajalein":                        "Pacific/Kwajalein",
	"Pacific/Midway":                   "Pacific/Pago_Pago",
	"Pacific/Samoa":                    "Pacific/Pago_Pago",
	"US/Samoa":                         "Pacific/Pago_Pago",
	"Pacific/Ponape":                   "Pacific/Pohnpei",
}
