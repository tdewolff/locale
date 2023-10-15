package locale

import (
	"time"

	"golang.org/x/text/message"
)

type Printer message.Printer

func (printer *Printer) T(a ...interface{}) string {
	p := (*message.Printer)(printer)
	if len(a) == 0 {
		return ""
	} else if s, ok := a[0].(string); ok {
		return p.Sprintf(s, a[1:]...)
	} else if len(a) == 3 {
		from, ok0 := a[0].(time.Time)
		to, ok1 := a[1].(time.Time)
		layout, ok2 := a[2].(string)
		if ok0 && ok1 && ok2 {
			return p.Sprintf("%v", IntervalFormatter{from, to, layout})
		}
	} else if len(a) == 2 {
		if layout, ok := a[1].(string); ok {
			switch v := a[0].(type) {
			case time.Time:
				return p.Sprintf("%v", TimeFormatter{v, layout})
			case time.Duration:
				return p.Sprintf("%v", DurationFormatter{Duration(v), layout})
			case Duration:
				return p.Sprintf("%v", DurationFormatter{v, layout})
			case Amount:
				return p.Sprintf("%v", AmountFormatter{v, layout})
			}
		}
	} else if len(a) == 1 {
		switch v := a[0].(type) {
		case int:
			return p.Sprintf("%v", DecimalFormatter{float64(v)})
		case int16:
			return p.Sprintf("%v", DecimalFormatter{float64(v)})
		case int32:
			return p.Sprintf("%v", DecimalFormatter{float64(v)})
		case int64:
			return p.Sprintf("%v", DecimalFormatter{float64(v)})
		case uint:
			return p.Sprintf("%v", DecimalFormatter{float64(v)})
		case uint16:
			return p.Sprintf("%v", DecimalFormatter{float64(v)})
		case uint32:
			return p.Sprintf("%v", DecimalFormatter{float64(v)})
		case uint64:
			return p.Sprintf("%v", DecimalFormatter{float64(v)})
		case float32:
			return p.Sprintf("%v", DecimalFormatter{float64(v)})
		case float64:
			return p.Sprintf("%v", DecimalFormatter{v})
		}
	}
	return p.Sprint(a...)
}
