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
	} else if len(a) == 2 {
		if layout, ok := a[1].(string); ok {
			switch v := a[0].(type) {
			// TODO: handle numbers
			case time.Time:
				return p.Sprintf("%v", TimeFormatter{v, layout})
			case Date:
				return p.Sprintf("%v", TimeFormatter{time.Time(v), layout})
			case Time:
				return p.Sprintf("%v", TimeFormatter{time.Time(v), layout})
			case Datetime:
				return p.Sprintf("%v", TimeFormatter{time.Time(v), layout})
			case time.Duration:
				return p.Sprintf("%v", TimeFormatter{NullTime.Add(v), layout})
			case Duration:
				return p.Sprintf("%v", TimeFormatter{NullTime.Add(time.Duration(v)), layout})
			case Amount:
				return p.Sprintf("%v", AmountFormatter{v, layout})
			}
		}
	}
	return p.Sprint(a...)
}
