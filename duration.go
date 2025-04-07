package locale

import (
	"database/sql/driver"
	"fmt"
	"log"
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
	return d.String(), nil
}

// Available duration layouts
const (
	DurationLong   string = "seconds"
	DurationShort         = "sec"
	DurationNarrow        = "s"
)

type DurationFormatter struct {
	Duration
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

	num := int64(f.Duration)
	unitType := []string{"week", "day", "hour", "minute", "second", "millisecond", "microsecond", "nanosecond"}
	unitSize := []int64{7 * 24 * 3600 * 1e9, 24 * 3600 * 1e9, 3600 * 1e9, 60 * 1e9, 1e9, 1e6, 1e3, 1}
	for i := 0; num != 0 && i < len(unitType); i++ {
		if v := num / unitSize[i]; v != 0 {
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
			}

			pattern := count.Other
			if v == 1 {
				pattern = count.One
			}
			pattern = strings.ReplaceAll(pattern, "{0}", fmt.Sprintf("%d", v))
			if 1 < len(b) {
				b = append(b, ' ')
			}
			b = append(b, []byte(pattern)...)
		}
		num %= unitSize[i]
	}
	state.Write(b)
	return
}
