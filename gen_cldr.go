//go:build ignore

package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/language"
)

const BasePath = "cldr/"
const BaseURL = "https://raw.githubusercontent.com/unicode-org/cldr/main/common/"

var LocaleNames = []string{
	"root",
	"en",
	"es",
	"es_419",
	"es_CL",
}

var CurrencyNames = []string{
	"USD",
	"CLP",
	"EUR",
}

type CurrencyFormat struct {
	Standard string
	Amount   string
	ISO      string
}

type CalendarFormat struct {
	Full   string
	Long   string
	Medium string
	Short  string
}

type CurrencySymbol struct {
	Standard string
	Narrow   string
}

type CalendarSymbol struct {
	Wide        string
	Abbreviated string
	Narrow      string
}

type Locale struct {
	CurrencyFormat CurrencyFormat
	DateFormat     CalendarFormat
	TimeFormat     CalendarFormat
	DatetimeFormat CalendarFormat

	CurrencySymbol        map[string]CurrencySymbol
	DecimalSymbol         rune
	GroupSymbol           rune
	CurrencyDecimalSymbol rune
	CurrencyGroupSymbol   rune
	PlusSymbol            rune
	MinusSymbol           rune
	TimeSeparatorSymbol   rune
	MonthSymbol           [12]CalendarSymbol
	DaySymbol             [7]CalendarSymbol
	DayPeriodSymbol       [2]CalendarSymbol
}

type Currency struct {
	Digits       int
	Rounding     int
	CashDigits   int
	CashRounding int
}

var dayMap = map[string]int{
	"sun": 0,
	"mon": 1,
	"tue": 2,
	"wed": 3,
	"thu": 4,
	"fri": 5,
	"sat": 6,
}

func main() {
	locales := map[string]Locale{}
	for _, localeName := range LocaleNames {
		tag := language.MustParse(localeName)
		locale := Locale{
			CurrencySymbol: map[string]CurrencySymbol{},
		}
		if !tag.IsRoot() {
			parent := tag.Parent().String()
			parent = strings.Replace(parent, "-", "_", 1)
			if tag.Parent().IsRoot() {
				parent = "root"
			}
			if parentLocale, ok := locales[parent]; !ok {
				panic(fmt.Sprintf("%v: parent locale %v not found", tag.String(), parent))
			} else {
				locale = parentLocale
			}
		}

		err := readXMLLeafs("main/"+localeName+".xml", func(tags []string, attrs []map[string]string, content string) {
			if content != "↑↑↑" {
				if isTag(tags, attrs, "ldml/numbers/currencyFormats/currencyFormatLength[!type]/currencyFormat[type=standard]/pattern") {

					if alt := attrs[len(attrs)-1]["alt"]; alt == "" {
						locale.CurrencyFormat.Standard = content
					} else if alt == "noCurrency" {
						locale.CurrencyFormat.Amount = content
					} else if alt == "alphaNextToNumber" {
						locale.CurrencyFormat.ISO = content
					}
				} else if isTag(tags, attrs, "ldml/numbers/symbols[numberSystem=latn]/*") {
					if r, _ := utf8.DecodeRuneInString(content); r != utf8.RuneError {
						switch tags[len(tags)-1] {
						case "decimal":
							locale.DecimalSymbol = r
						case "group":
							locale.GroupSymbol = r
						case "currencyDecimal":
							locale.CurrencyDecimalSymbol = r
						case "currencyGroup":
							locale.CurrencyGroupSymbol = r
						case "plusSign":
							locale.PlusSymbol = r
						case "minusSign":
							locale.MinusSymbol = r
						case "timeSeparator":
							locale.TimeSeparatorSymbol = r
						}
					}
				} else if isTag(tags, attrs, "ldml/numbers/currencies/currency/symbol") {
					currency := attrs[len(attrs)-2]["type"]
					currencySymbol := locale.CurrencySymbol[currency]
					if attrs[len(attrs)-1]["alt"] == "narrow" {
						currencySymbol.Narrow = content
					} else {
						currencySymbol.Standard = content
					}
					locale.CurrencySymbol[currency] = currencySymbol
				} else if isTag(tags, attrs, "ldml/dates/calendars/calendar[type=gregorian]/months/monthContext/monthWidth/month") {
					if month, _ := strconv.Atoi(attrs[len(attrs)-1]["type"]); 1 <= month && month <= 12 {
						context := attrs[len(attrs)-3]["type"]
						width := attrs[len(attrs)-2]["type"]
						if context == "format" && width == "wide" {
							locale.MonthSymbol[month-1].Wide = content
						} else if context == "format" && width == "abbreviated" {
							locale.MonthSymbol[month-1].Abbreviated = content
						} else if context == "stand-alone" && width == "narrow" {
							locale.MonthSymbol[month-1].Narrow = content
						}
					}
				} else if isTag(tags, attrs, "ldml/dates/calendars/calendar[type=gregorian]/days/dayContext/dayWidth/day") {
					if day, ok := dayMap[attrs[len(attrs)-1]["type"]]; ok {
						context := attrs[len(attrs)-3]["type"]
						width := attrs[len(attrs)-2]["type"]
						if context == "format" && width == "wide" {
							locale.DaySymbol[day].Wide = content
						} else if context == "format" && width == "abbreviated" {
							locale.DaySymbol[day].Abbreviated = content
						} else if context == "stand-alone" && width == "narrow" {
							locale.DaySymbol[day].Narrow = content
						}
					}
				} else if isTag(tags, attrs, "ldml/dates/calendars/calendar[type=gregorian]/dayPeriods/dayPeriodContext/dayPeriodWidth/dayPeriod") {
					if period := attrs[len(attrs)-1]["type"]; period == "am" || period == "pm" {
						i := 0
						if period == "pm" {
							i = 1
						}
						context := attrs[len(attrs)-3]["type"]
						width := attrs[len(attrs)-2]["type"]
						if context == "format" && width == "wide" {
							locale.DayPeriodSymbol[i].Wide = content
						} else if context == "format" && width == "abbreviated" {
							locale.DayPeriodSymbol[i].Abbreviated = content
						} else if context == "stand-alone" && width == "narrow" {
							locale.DayPeriodSymbol[i].Narrow = content
						}
					}
				} else if isTag(tags, attrs, "ldml/dates/calendars/calendar[type=gregorian]/dateFormats/dateFormatLength/dateFormat/pattern") {
					length := attrs[len(attrs)-3]["type"]
					if length == "full" {
						locale.DateFormat.Full = content
					} else if length == "long" {
						locale.DateFormat.Long = content
					} else if length == "medium" {
						locale.DateFormat.Medium = content
					} else if length == "short" {
						locale.DateFormat.Short = content
					}
				} else if isTag(tags, attrs, "ldml/dates/calendars/calendar[type=gregorian]/timeFormats/timeFormatLength/timeFormat/pattern") {
					length := attrs[len(attrs)-3]["type"]
					if length == "full" {
						locale.TimeFormat.Full = content
					} else if length == "long" {
						locale.TimeFormat.Long = content
					} else if length == "medium" {
						locale.TimeFormat.Medium = content
					} else if length == "short" {
						locale.TimeFormat.Short = content
					}
				} else if isTag(tags, attrs, "ldml/dates/calendars/calendar[type=gregorian]/dateTimeFormats/dateTimeFormatLength/dateTimeFormat/pattern") {
					length := attrs[len(attrs)-3]["type"]
					if length == "full" {
						locale.DatetimeFormat.Full = content
					} else if length == "long" {
						locale.DatetimeFormat.Long = content
					} else if length == "medium" {
						locale.DatetimeFormat.Medium = content
					} else if length == "short" {
						locale.DatetimeFormat.Short = content
					}
				}
			}
		})
		if err != nil {
			panic(err)
		}
		locales[localeName] = locale
	}
	for localeName, locale := range locales {
		if locale.CurrencyFormat.Amount == "" {
			locale.CurrencyFormat.Amount = locale.CurrencyFormat.Standard
		}
		if locale.CurrencyFormat.ISO == "" {
			locale.CurrencyFormat.ISO = locale.CurrencyFormat.Standard
		}
		if locale.CurrencyDecimalSymbol == 0 {
			locale.CurrencyDecimalSymbol = locale.DecimalSymbol
		}
		if locale.CurrencyGroupSymbol == 0 {
			locale.CurrencyGroupSymbol = locale.GroupSymbol
		}
		locales[localeName] = locale
	}

	currencies := map[string]Currency{}
	err := readXMLLeafs("supplemental/supplementalData.xml", func(tags []string, attrs []map[string]string, content string) {
		if isTag(tags, attrs, "supplementalData/currencyData/fractions/info") {
			currency := Currency{
				Digits:       -1,
				Rounding:     -1,
				CashDigits:   -1,
				CashRounding: -1,
			}
			iso4217 := ""
			for key, val := range attrs[len(attrs)-1] {
				if key == "iso4217" {
					iso4217 = val
					continue
				}

				i, err := strconv.Atoi(val)
				if err != nil {
					continue
				}
				switch key {
				case "digits":
					currency.Digits = i
				case "rounding":
					currency.Rounding = i
				case "cashDigits":
					currency.CashDigits = i
				case "cashRounding":
					currency.CashRounding = i
				}
			}
			if iso4217 != "" {
				if currency.CashDigits == -1 {
					currency.CashDigits = currency.Digits
				}
				if currency.CashRounding == -1 {
					currency.CashRounding = currency.Rounding
				}
				currencies[iso4217] = currency
			}
		}
	})
	if err != nil {
		panic(err)
	}

	f, err := os.Create("cldr.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	w.Write([]byte("// Automatically generated by gen_cldr.go\n"))
	w.Write([]byte("package locale\n"))

	types := []interface{}{CurrencyFormat{}, CalendarFormat{}, CurrencySymbol{}, CalendarSymbol{}, Locale{}, Currency{}}
	for _, v := range types {
		t := reflect.TypeOf(v)
		fmt.Fprintf(w, "\ntype %v ", t.Name())
		if err := printRootType(w, t); err != nil {
			panic(err)
		}
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "\nvar locales = map[string]Locale")
	if err := printValue(w, reflect.ValueOf(locales)); err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "\nvar currencies = map[string]Currency")
	if err := printValue(w, reflect.ValueOf(currencies)); err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "\n")
}

func readXMLLeafs(filename string, cb func([]string, []map[string]string, string)) error {
	if _, err := os.Stat(BasePath + filename); err != nil && err != os.ErrNotExist {
		return err
	} else if err == os.ErrNotExist {
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return err
		}
		f, err := os.Create(BasePath + filename)
		if err != nil {
			return err
		}

		resp, err := http.Get(BaseURL + filename)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if _, err := io.Copy(f, resp.Body); err != nil {
			return err
		}
	}

	f, err := os.Open(BasePath + filename)
	if err != nil {
		return err
	}

	state := 0
	tags := []string{}
	attrs := []map[string]string{}
	content := ""
	decoder := xml.NewDecoder(f)
	for {
		t, err := decoder.Token()
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}

		if elem, ok := t.(xml.StartElement); ok {
			tags = append(tags, elem.Name.Local)
			attr := map[string]string{}
			for _, a := range elem.Attr {
				attr[a.Name.Local] = a.Value
			}
			attrs = append(attrs, attr)
			content = ""
			state = 1
		} else if char, ok := t.(xml.CharData); ok && state == 1 {
			content = string(char)
		} else if _, ok = t.(xml.EndElement); ok {
			if state == 1 {
				cb(tags, attrs, content)
			}
			attrs = attrs[:len(attrs)-1]
			tags = tags[:len(tags)-1]
			state = 0
		} else {
			state = 0
		}
	}
}

func isTag(tags []string, attrs []map[string]string, s string) bool {
	elems := strings.Split(s, "/")
	if len(tags) != len(elems) {
		return false
	}
	for i, elem := range elems {
		if elem == "*" {
			return true
		}

		idx := strings.IndexByte(elem, '[')
		if idx == -1 {
			idx = len(elem)
		}

		tag := elem[:idx]
		if tag != tags[i] {
			return false
		}
		for idx < len(elem) {
			if elem[idx] != '[' {
				panic("wrong tag attr syntax")
			}
			end := strings.IndexByte(elem[idx+1:], ']')
			if end == -1 {
				panic("wrong tag attr syntax")
			}
			is := strings.IndexByte(elem[idx+1:], '=')
			if is == -1 {
				is = end
			}

			key := elem[idx+1 : idx+1+is]
			if key[0] == '!' {
				if _, ok := attrs[i][key[1:]]; ok {
					return false
				}
			} else if attrVal, ok := attrs[i][key]; !ok {
				return false
			} else if is != end {
				val := elem[idx+1+is+1 : idx+1+end]
				if val != attrVal {
					return false
				}
			}
			idx += 1 + end + 1
		}
	}
	return true
}

type Prefixer struct {
	io.Writer
	prefix []byte
}

func NewPrefixer(w io.Writer, prefix string) *Prefixer {
	return &Prefixer{w, []byte(prefix)}
}

func (w Prefixer) Write(b []byte) (int, error) {
	for i := len(b) - 1; 0 <= i; i-- {
		if b[i] == '\n' {
			b = append(b[:i+1], append(w.prefix, b[i+1:]...)...)
		}
	}
	return w.Writer.Write(b)
}

func printRootType(w io.Writer, t reflect.Type) error {
	if t.Kind() == reflect.Struct {
		fmt.Fprintf(w, "struct {")
		n := t.NumField()
		wi := NewPrefixer(w, "    ")
		fieldLen := 0
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if fieldLen < len(field.Name) {
				fieldLen = len(field.Name)
			}
		}
		for i := 0; i < n; i++ {
			field := t.Field(i)
			fmt.Fprintf(wi, "\n%s%s ", field.Name, strings.Repeat(" ", fieldLen-len(field.Name)))
			if err := printType(wi, field.Type); err != nil {
				return fmt.Errorf("struct field %v: %v", field.Name, err)
			}
		}
		if 0 < n {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "}")
		return nil
	}
	return printType(w, t)
}

func printType(w io.Writer, t reflect.Type) error {
	switch t.Kind() {
	case reflect.Bool:
		fmt.Fprintf(w, "bool")
	case reflect.Int:
		fmt.Fprintf(w, "int")
	case reflect.Int8:
		fmt.Fprintf(w, "int8")
	case reflect.Int16:
		fmt.Fprintf(w, "int16")
	case reflect.Int32:
		fmt.Fprintf(w, "int32")
	case reflect.Int64:
		fmt.Fprintf(w, "int64")
	case reflect.Uint:
		fmt.Fprintf(w, "uint")
	case reflect.Uint8:
		fmt.Fprintf(w, "uint8")
	case reflect.Uint16:
		fmt.Fprintf(w, "uint16")
	case reflect.Uint32:
		fmt.Fprintf(w, "uint32")
	case reflect.Uint64:
		fmt.Fprintf(w, "uint64")
	case reflect.Float32:
		fmt.Fprintf(w, "float32")
	case reflect.Float64:
		fmt.Fprintf(w, "float64")
	case reflect.Array:
		fmt.Fprintf(w, "[%d]", t.Len())
		if err := printType(w, t.Elem()); err != nil {
			return fmt.Errorf("array: %v", err)
		}
	case reflect.Slice:
		fmt.Fprintf(w, "[]")
		if err := printType(w, t.Elem()); err != nil {
			return fmt.Errorf("slice: %v", err)
		}
	case reflect.Map:
		fmt.Fprintf(w, "map[")
		if err := printType(w, t.Key()); err != nil {
			return fmt.Errorf("map key: %v", err)
		}
		fmt.Fprintf(w, "]")
		if err := printType(w, t.Elem()); err != nil {
			return fmt.Errorf("map value: %v", err)
		}
	case reflect.String:
		fmt.Fprintf(w, "string")
	case reflect.Struct:
		fmt.Fprintf(w, t.Name())
	default:
		return fmt.Errorf("unsupported type: %v", t)
	}
	return nil
}

func printValue(w io.Writer, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			fmt.Fprintf(w, "true")
		} else {
			fmt.Fprintf(w, "false")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Fprintf(w, "%v", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fmt.Fprintf(w, "%v", v.Uint())
	case reflect.Float32, reflect.Float64:
		fmt.Fprintf(w, "%v", v.Float())
	case reflect.Array, reflect.Slice:
		fmt.Fprintf(w, "{")
		n := v.Len()
		wi := NewPrefixer(w, "    ")
		for i := 0; i < n; i++ {
			fmt.Fprintf(wi, "\n")
			if err := printValue(wi, v.Index(i)); err != nil {
				return fmt.Errorf("array/slice index %v: %v", i, err)
			}
			fmt.Fprintf(wi, ",")
		}
		if 0 < n {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "}")
	case reflect.Map:
		fmt.Fprintf(w, "{")
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})
		wi := NewPrefixer(w, "    ")
		for i := 0; i < len(keys); i++ {
			fmt.Fprintf(wi, "\n")
			if err := printValue(wi, keys[i]); err != nil {
				return fmt.Errorf("map key %v: %v", keys[i], err)
			}
			fmt.Fprintf(wi, ": ")
			if err := printValue(wi, v.MapIndex(keys[i])); err != nil {
				return fmt.Errorf("map value for %v: %v", keys[i], err)
			}
			fmt.Fprintf(wi, ",")
		}
		if 0 < v.Len() {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "}")
	case reflect.String:
		fmt.Fprintf(w, "\"%v\"", v.String())
	case reflect.Struct:
		fmt.Fprintf(w, "{")
		n := v.NumField()
		wi := w
		for i := 0; i < n; i++ {
			if i != 0 {
				fmt.Fprintf(w, ", ")
			}
			field := v.Field(i)
			if k := field.Kind(); k == reflect.Array || k == reflect.Map || k == reflect.Slice || k == reflect.Struct {
				if err := printType(wi, field.Type()); err != nil {
					return fmt.Errorf("struct field %v: %v", v.Type().Field(i), err)
				}
			}
			if err := printValue(wi, field); err != nil {
				return fmt.Errorf("struct field %v: %v", v.Type().Field(i), err)
			}
		}
		fmt.Fprintf(w, "}")
	default:
		return fmt.Errorf("unsupported value: %v", v)
	}
	return nil
}
