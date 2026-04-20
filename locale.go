package locale

//go:generate go run gen_cldr.go

import (
	"reflect"
	"strings"
	"time"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type Printer struct {
	*message.Printer

	LanguageTag language.Tag
	Location    *time.Location
}

var _ = reflect.TypeOf(Printer{}) // no garble

type Translator interface {
	Translate(*Printer) string
}

func NewPrinter(t language.Tag, loc *time.Location) *Printer {
	return &Printer{
		Printer:     message.NewPrinter(t),
		LanguageTag: t,
		Location:    loc,
	}
}

func (p *Printer) T(a ...any) string {
	if len(a) == 0 {
		return ""
	} else if s, ok := a[0].(string); ok {
		if len(a) == 1 {
			return p.Sprintf(strings.ReplaceAll(s, "%", "%%")) // TODO: why?
		} else if len(a) == 2 {
			if A, ok := a[1].([]any); ok {
				// allow passing array of arguments instead of variadic arguments
				a = append([]any{a[0]}, A...)
			}
		}
	} else if len(a) == 3 {
		if layout, ok := a[2].(string); ok {
			if from, ok := a[0].(time.Time); ok {
				switch v := a[1].(type) {
				case time.Time:
					return p.Sprintf("%v", IntervalFormatter{from.In(p.Location), v.In(p.Location), layout})
				case time.Duration:
					return p.Sprintf("%v", DurationIntervalFormatter{from.In(p.Location), v, layout})
				case Duration:
					return p.Sprintf("%v", DurationIntervalFormatter{from.In(p.Location), time.Duration(v), layout})
				}
			}
		}
	} else if len(a) == 2 {
		if layout, ok := a[1].(string); ok {
			switch v := a[0].(type) {
			case time.Time:
				v = v.In(p.Location)
				return p.Sprintf("%v", TimeFormatter{v, layout})
			case *time.Location:
				return p.Sprintf("%v", TimezoneFormatter{v, layout})
			case time.Duration:
				return p.Sprintf("%v", DurationFormatter{v, layout})
			case Duration:
				return p.Sprintf("%v", DurationFormatter{time.Duration(v), layout})
			case Amount:
				return p.Sprintf("%v", AmountFormatter{v, layout})
			case currency.Unit:
				return p.Sprintf("%v", CurrencyFormatter{v, layout})
			}
		}
	}
	for i, arg := range a {
		switch v := arg.(type) {
		case int:
			a[i] = DecimalFormatter{float64(v)}
		case int16:
			a[i] = DecimalFormatter{float64(v)}
		case int32:
			a[i] = DecimalFormatter{float64(v)}
		case int64:
			a[i] = DecimalFormatter{float64(v)}
		case uint:
			a[i] = DecimalFormatter{float64(v)}
		case uint16:
			a[i] = DecimalFormatter{float64(v)}
		case uint32:
			a[i] = DecimalFormatter{float64(v)}
		case uint64:
			a[i] = DecimalFormatter{float64(v)}
		case float32:
			a[i] = DecimalFormatter{float64(v)}
		case float64:
			a[i] = DecimalFormatter{v}
		case language.Region:
			a[i] = RegionFormatter{v}
		default:
			if translator, ok := arg.(Translator); ok {
				a[i] = translator.Translate(p)
			}
		}
	}
	if s, ok := a[0].(string); ok {
		return p.Sprintf(s, a[1:]...)
	}
	return p.Sprint(a...)
}
