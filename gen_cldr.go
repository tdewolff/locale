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
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/language"
)

const BasePath = "cldr/"
const BaseURL = "https://raw.githubusercontent.com/unicode-org/cldr/main/common/"
const CacheDuration = 7 * 24 * time.Hour

var LocaleNames = []string{
	"root",
	"en",
	"es",
	"es_419",
	"es_CL",
	"nl",
	"nl_NL",
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

type CalendarSymbol struct {
	Wide        string
	Abbreviated string
	Narrow      string
}

type Count struct {
	One   string
	Other string
}

type Currency struct {
	Name     string
	Standard string
	Narrow   string
}

type Unit struct {
	Long   Count
	Short  Count
	Narrow Count
}

type MetazoneSymbol struct {
	Long  string
	Short string
}

type Metazone struct {
	Generic  MetazoneSymbol
	Standard MetazoneSymbol
	Daylight MetazoneSymbol
}

type Locale struct {
	DecimalFormat          string
	CurrencyFormat         CurrencyFormat
	DateFormat             CalendarFormat
	TimeFormat             CalendarFormat
	DatetimeFormat         CalendarFormat
	DatetimeIntervalFormat map[string]map[string]string
	TimezoneFormat         string

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
	TimezoneCity          map[string]string
	Metazones             map[string]Metazone

	Currency map[string]Currency
	Unit     map[string]Unit

	Territory map[string]string
}

type CurrencyInfo struct {
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
	localeXMLs := map[string]*XMLNode{}
	for _, localeName := range LocaleNames {
		tag := language.MustParse(localeName)
		locale := Locale{
			Currency:               map[string]Currency{},
			Unit:                   map[string]Unit{},
			Territory:              map[string]string{},
			TimezoneCity:           map[string]string{},
			Metazones:              map[string]Metazone{},
			DatetimeIntervalFormat: map[string]map[string]string{},
		}

		var parentXML *XMLNode
		if !tag.IsRoot() {
			parent := tag.Parent()
			name := parent.String()
			name = strings.Replace(name, "-", "_", 1)
			if parent.IsRoot() {
				name = "root"
			}
			var ok bool
			if parentXML, ok = localeXMLs[name]; !ok {
				panic(fmt.Sprintf("%v: parent locale %v not found", tag.String(), name))
			}
		}

		datetimeAvailableFormat := map[string]string{}
		if xmlLocale, err := ParseXML("main/" + localeName + ".xml"); err != nil {
			panic(err)
		} else {
			xmlLocale.ResolveAliases()
			if parentXML != nil {
				xmlLocale.InheritFrom(parentXML)
			}
			localeXMLs[localeName] = xmlLocale

			if n, ok := xmlLocale.Find("ldml/numbers/decimalFormats[numberSystem=latn]/decimalFormatLength[!type]/decimalFormat/pattern"); ok {
				locale.DecimalFormat = n.Text
			}
			for _, n := range xmlLocale.FindAll("ldml/numbers/currencyFormats[numberSystem=latn]/currencyFormatLength[!type]/currencyFormat[type=standard]/pattern") {
				if alt := n.Attr("alt"); alt == "" {
					locale.CurrencyFormat.Standard = n.Text
				} else if alt == "noCurrency" {
					locale.CurrencyFormat.Amount = n.Text
				} else if alt == "alphaNextToNumber" {
					locale.CurrencyFormat.ISO = n.Text
				}
			}
			for _, n := range xmlLocale.FindAll("ldml/numbers/symbols[numberSystem=latn]/*") {
				if r, _ := utf8.DecodeRuneInString(n.Text); r != utf8.RuneError {
					switch n.Tag {
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
			}
			for _, n := range xmlLocale.FindAll("ldml/numbers/currencies/currency[type]/*") {
				cur := n.Parent.Attr("type")
				currency := locale.Currency[cur]
				if n.Tag == "displayName" {
					if _, ok := n.Attr2("count"); !ok {
						currency.Name = n.Text
					}
				} else if n.Tag == "symbol" {
					if n.Attr("alt") == "narrow" {
						currency.Narrow = n.Text
					} else {
						currency.Standard = n.Text
					}
				}
				locale.Currency[cur] = currency
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/calendars/calendar[type=gregorian]/months/monthContext[type]/monthWidth[type]/month[type]") {
				if month, _ := strconv.Atoi(n.Attr("type")); 1 <= month && month <= 12 {
					width := n.Parent.Attr("type")
					context := n.Parent.Parent.Attr("type")
					if context == "format" && width == "wide" {
						locale.MonthSymbol[month-1].Wide = n.Text
					} else if context == "format" && width == "abbreviated" {
						locale.MonthSymbol[month-1].Abbreviated = n.Text
					} else if context == "stand-alone" && width == "narrow" {
						locale.MonthSymbol[month-1].Narrow = n.Text
					}
				}
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/calendars/calendar[type=gregorian]/days/dayContext[type]/dayWidth[type]/day[type]") {
				if day, ok := dayMap[n.Attr("type")]; ok {
					width := n.Parent.Attr("type")
					context := n.Parent.Parent.Attr("type")
					if context == "format" && width == "wide" {
						locale.DaySymbol[day].Wide = n.Text
					} else if context == "format" && width == "abbreviated" {
						locale.DaySymbol[day].Abbreviated = n.Text
					} else if context == "stand-alone" && width == "narrow" {
						locale.DaySymbol[day].Narrow = n.Text
					}
				}
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/calendars/calendar[type=gregorian]/dayPeriods/dayPeriodContext[type]/dayPeriodWidth[type]/dayPeriod[type]") {
				if period := n.Attr("type"); period == "am" || period == "pm" {
					i := 0
					if period == "pm" {
						i = 1
					}
					width := n.Parent.Attr("type")
					context := n.Parent.Parent.Attr("type")
					if context == "format" && width == "wide" {
						locale.DayPeriodSymbol[i].Wide = n.Text
					} else if context == "format" && width == "abbreviated" {
						locale.DayPeriodSymbol[i].Abbreviated = n.Text
					} else if context == "stand-alone" && width == "narrow" {
						locale.DayPeriodSymbol[i].Narrow = n.Text
					}
				}
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/calendars/calendar[type=gregorian]/dateFormats/dateFormatLength[type]/dateFormat/pattern") {
				if length := n.Parent.Parent.Attr("type"); length == "full" {
					locale.DateFormat.Full = n.Text
				} else if length == "long" {
					locale.DateFormat.Long = n.Text
				} else if length == "medium" {
					locale.DateFormat.Medium = n.Text
				} else if length == "short" {
					locale.DateFormat.Short = n.Text
				}
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/calendars/calendar[type=gregorian]/timeFormats/timeFormatLength[type]/timeFormat/pattern[!alt]") {
				if length := n.Parent.Parent.Attr("type"); length == "full" {
					locale.TimeFormat.Full = n.Text
				} else if length == "long" {
					locale.TimeFormat.Long = n.Text
				} else if length == "medium" {
					locale.TimeFormat.Medium = n.Text
				} else if length == "short" {
					locale.TimeFormat.Short = n.Text
				}
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/calendars/calendar[type=gregorian]/dateTimeFormats/dateTimeFormatLength[type]/dateTimeFormat/pattern") {
				if length := n.Parent.Parent.Attr("type"); length == "full" {
					locale.DatetimeFormat.Full = n.Text
				} else if length == "long" {
					locale.DatetimeFormat.Long = n.Text
				} else if length == "medium" {
					locale.DatetimeFormat.Medium = n.Text
				} else if length == "short" {
					locale.DatetimeFormat.Short = n.Text
				}
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/calendars/calendar[type=gregorian]/dateTimeFormats/availableFormats/dateFormatItem[id]") {
				datetimeAvailableFormat[n.Attr("id")] = n.Text
			}
			if n, ok := xmlLocale.Find("ldml/dates/calendars/calendar[type=gregorian]/dateTimeFormats/intervalFormats/intervalFormatFallback"); ok {
				if _, ok := locale.DatetimeIntervalFormat[""]; !ok {
					locale.DatetimeIntervalFormat[""] = map[string]string{}
				}
				locale.DatetimeIntervalFormat[""][""] = n.Text
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/calendars/calendar[type=gregorian]/dateTimeFormats/intervalFormats/intervalFormatItem[id]/greatestDifference[id]") {
				id := n.Parent.Attr("id")
				if _, ok := locale.DatetimeIntervalFormat[id]; !ok {
					locale.DatetimeIntervalFormat[id] = map[string]string{}
				}
				greatestDifference := n.Attr("id")
				locale.DatetimeIntervalFormat[id][greatestDifference] = n.Text
			}
			if n, ok := xmlLocale.Find("ldml/dates/calendars/calendar[type=gregorian]/dateTimeFormats/appendItems/appendItem[request=Timezone]"); ok {
				locale.TimezoneFormat = n.Text
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/timeZoneNames/zone[type]/exemplarCity") {
				locale.TimezoneCity[n.Parent.Attr("type")] = n.Text
			}
			for _, n := range xmlLocale.FindAll("ldml/dates/timeZoneNames/metazone[type]/*/*") {
				typ := n.Parent.Parent.Attr("type")
				metazone := locale.Metazones[typ]
				switch n.Parent.Tag {
				case "long":
					switch n.Tag {
					case "generic":
						metazone.Generic.Long = n.Text
					case "standard":
						metazone.Standard.Long = n.Text
					case "daylight":
						metazone.Daylight.Long = n.Text
					}
				case "short":
					switch n.Tag {
					case "generic":
						metazone.Generic.Short = n.Text
					case "standard":
						metazone.Standard.Short = n.Text
					case "daylight":
						metazone.Daylight.Short = n.Text
					}
				}
				locale.Metazones[typ] = metazone
			}
			for _, n := range xmlLocale.FindAll("ldml/units/unitLength[type]/unit[type]/unitPattern[count]") {
				if unitName := n.Parent.Attr("type"); strings.HasPrefix(unitName, "duration-") {
					var count *Count
					unit := locale.Unit[unitName]
					switch n.Parent.Parent.Attr("type") {
					case "long":
						count = &unit.Long
					case "short":
						count = &unit.Short
					case "narrow":
						count = &unit.Narrow
					default:
						return
					}
					switch n.Attr("count") {
					case "one":
						count.One = n.Text
					case "other":
						count.Other = n.Text
					default:
						return
					}
					locale.Unit[unitName] = unit
				}
			}
			for _, n := range xmlLocale.FindAll("ldml/localeDisplayNames/territories/territory[type]") {
				locale.Territory[n.Attr("type")] = n.Text
			}
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

	currencyInfos := map[string]CurrencyInfo{}
	if xmlSupplementalData, err := ParseXML("supplemental/supplementalData.xml"); err != nil {
		panic(err)
	} else {
		for _, n := range xmlSupplementalData.FindAll("supplementalData/currencyData/fractions/info") {
			currencyInfo := CurrencyInfo{
				Digits:       -1,
				Rounding:     -1,
				CashDigits:   -1,
				CashRounding: -1,
			}
			iso4217 := ""
			for _, attr := range n.Attrs {
				if attr[0] == "iso4217" {
					iso4217 = attr[1]
					continue
				}

				i, err := strconv.Atoi(attr[1])
				if err != nil {
					continue
				}
				switch attr[0] {
				case "digits":
					currencyInfo.Digits = i
				case "rounding":
					currencyInfo.Rounding = i
				case "cashDigits":
					currencyInfo.CashDigits = i
				case "cashRounding":
					currencyInfo.CashRounding = i
				}
			}
			if iso4217 != "" {
				if currencyInfo.CashDigits == -1 {
					currencyInfo.CashDigits = currencyInfo.Digits
				}
				if currencyInfo.CashRounding == -1 {
					currencyInfo.CashRounding = currencyInfo.Rounding
				}
				currencyInfos[iso4217] = currencyInfo
			}
		}
	}

	metazones := map[string]string{}
	if xmlMetaZones, err := ParseXML("supplemental/metaZones.xml"); err != nil {
		panic(err)
	} else {
		for _, n := range xmlMetaZones.FindAll("supplementalData/metaZones/metazoneInfo/timezone[type]/usesMetazone[mzone][!to]") {
			timezone := n.Parent.Attr("type")
			metazone := n.Attr("mzone")
			metazones[timezone] = metazone
		}
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

	types := []interface{}{CurrencyFormat{}, CalendarFormat{}, CalendarSymbol{}, Count{}, Currency{}, Unit{}, Locale{}, CurrencyInfo{}, MetazoneSymbol{}, Metazone{}}
	for _, v := range types {
		t := reflect.TypeOf(v)
		fmt.Fprintf(w, "\ntype %v ", t.Name())
		if err := printType(w, t, 0); err != nil {
			panic(err)
		}
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "\nvar locales = map[string]Locale")
	if err := printValue(w, reflect.ValueOf(locales), 0); err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "\nvar currencies = map[string]CurrencyInfo")
	if err := printValue(w, reflect.ValueOf(currencyInfos), 0); err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "\nvar metazones = map[string]string")
	if err := printValue(w, reflect.ValueOf(metazones), 0); err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "\n")
}

type XMLNode struct {
	Parent *XMLNode
	Nodes  []*XMLNode

	Tag   string
	Attrs [][2]string
	Text  string
}

func ParseXML(filename string) (*XMLNode, error) {
	if info, err := os.Stat(BasePath + filename); err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if err != nil || CacheDuration < time.Now().Sub(info.ModTime()) {
		if err := os.MkdirAll(BasePath+filepath.Dir(filename), 0755); err != nil {
			return nil, err
		}
		f, err := os.Create(BasePath + filename)
		if err != nil {
			return nil, err
		}

		fmt.Println("Updating", filename)
		resp, err := http.Get(BaseURL + filename)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if _, err := io.Copy(f, resp.Body); err != nil {
			return nil, err
		}
	}

	f, err := os.Open(BasePath + filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	root := &XMLNode{}
	stack := []*XMLNode{root}
	decoder := xml.NewDecoder(f)
	for {
		t, err := decoder.Token()
		if err != nil {
			if err != io.EOF {
				return nil, err
			} else if len(root.Nodes) == 0 {
				return nil, nil
			}
			return root, nil
		}

		cur := stack[len(stack)-1]
		if elem, ok := t.(xml.StartElement); ok {
			attrs := [][2]string{}
			for _, attr := range elem.Attr {
				if attr.Name.Local != "draft" {
					attrs = append(attrs, [2]string{attr.Name.Local, attr.Value})
				}
			}
			slices.SortFunc(attrs, func(a, b [2]string) int {
				return strings.Compare(a[0], b[0])
			})
			n := &XMLNode{
				Parent: cur,
				Tag:    elem.Name.Local,
				Attrs:  attrs,
			}
			cur.Nodes = append(cur.Nodes, n)
			stack = append(stack, cur.Nodes[len(cur.Nodes)-1])
		} else if char, ok := t.(xml.CharData); ok {
			cur.Text += string(char)
		} else if _, ok = t.(xml.EndElement); ok {
			cur.Text = strings.TrimSpace(cur.Text)
			slices.SortFunc(cur.Nodes, func(a, b *XMLNode) int {
				return a.Compare(b)
			})
			stack = stack[:len(stack)-1]
		}
	}
}

func (a *XMLNode) Compare(b *XMLNode) int {
	if cmp := strings.Compare(a.Tag, b.Tag); cmp != 0 {
		return cmp
	}
	for i := 0; i < len(a.Attrs) && i < len(b.Attrs); i++ {
		if cmp := strings.Compare(a.Attrs[i][0], b.Attrs[i][0]); cmp != 0 {
			return cmp
		} else if cmp := strings.Compare(a.Attrs[i][1], b.Attrs[i][1]); cmp != 0 {
			return cmp
		}
	}
	if len(a.Attrs) < len(b.Attrs) {
		return -1
	} else if len(b.Attrs) < len(a.Attrs) {
		return 1
	}
	return 0
}

func (n *XMLNode) Attr(key string) string {
	for _, attr := range n.Attrs {
		if attr[0] == key {
			return attr[1]
		}
	}
	return ""
}

func (n *XMLNode) Attr2(key string) (string, bool) {
	for _, attr := range n.Attrs {
		if attr[0] == key {
			return attr[1], true
		}
	}
	return "", false
}

type XMLCondType int

const (
	XMLCondExists XMLCondType = iota
	XMLCondNotExists
	XMLCondValue
	XMLCondNotValue
)

type XMLCond struct {
	Type  XMLCondType
	Attr  string
	Value string
}

func (c XMLCond) Match(n *XMLNode) bool {
	switch c.Type {
	case XMLCondExists:
		_, ok := n.Attr2(c.Attr)
		return ok
	case XMLCondNotExists:
		_, ok := n.Attr2(c.Attr)
		return !ok
	case XMLCondValue:
		v, ok := n.Attr2(c.Attr)
		return ok && v == c.Value
	case XMLCondNotValue:
		v, ok := n.Attr2(c.Attr)
		return !ok || v != c.Value
	}
	return true
}

func (n *XMLNode) Path() string {
	query := ""
	for cur := n; cur != nil; cur = cur.Parent {
		elem := cur.Tag
		for _, attr := range cur.Attrs {
			elem += fmt.Sprintf("[%v=%v]", attr[0], attr[1])
		}
		query = elem + "/" + query
	}
	if 0 < len(query) {
		query = query[:len(query)-1]
	}
	return query
}

func (n *XMLNode) Find(xpath string) (*XMLNode, bool) {
	matches := n.FindAll(xpath)
	if len(matches) == 0 {
		return nil, false
	}
	return matches[0], true
}

func (n *XMLNode) FindAll(xpath string) []*XMLNode {
	elems := strings.Split(xpath, "/")
	if 0 < len(elems) && elems[0] == "" {
		for n.Parent != nil {
			n = n.Parent
		}
		elems = elems[1:]
	}
	if 0 < len(elems) && elems[len(elems)-1] == "" {
		elems = elems[:len(elems)-1]
	}

	matches, nodes := []*XMLNode{n}, []*XMLNode{}
	for _, elem := range elems {
		nodes = nodes[:0]
		for _, match := range matches {
			for _, n := range match.Nodes {
				nodes = append(nodes, n)
			}
		}

		conditions := []XMLCond{}
		for {
			if bracket := strings.LastIndexByte(elem, '['); bracket != -1 {
				cond := elem[bracket+1 : len(elem)-1]
				if notIs := strings.Index(cond, "!="); notIs != -1 {
					conditions = append(conditions, XMLCond{
						Type:  XMLCondNotValue,
						Attr:  cond[:notIs],
						Value: cond[notIs+2:],
					})
				} else if is := strings.IndexByte(cond, '='); is != -1 {
					conditions = append(conditions, XMLCond{
						Type:  XMLCondValue,
						Attr:  cond[:is],
						Value: cond[is+1:],
					})
				} else if strings.HasPrefix(cond, "!") {
					conditions = append(conditions, XMLCond{
						Type: XMLCondNotExists,
						Attr: cond[1:],
					})
				} else {
					conditions = append(conditions, XMLCond{
						Type: XMLCondExists,
						Attr: cond,
					})
				}
				elem = elem[:bracket]
			} else {
				break
			}
		}

		matches = matches[:0]
	NodesLoop:
		for _, n := range nodes {
			if elem != "*" && n.Tag != elem {
				continue
			}
			for _, cond := range conditions {
				if !cond.Match(n) {
					continue NodesLoop
				}
			}
			matches = append(matches, n)
		}
		if len(matches) == 0 {
			return []*XMLNode{}
		}
	}
	return matches
}

func (n *XMLNode) ResolveAliases() {
	for i := 0; i < len(n.Nodes); i++ {
		child := n.Nodes[i]
		if child.Text == "↑↑↑" {
			n.Nodes = append(n.Nodes[:i], n.Nodes[i+1:]...)
			i--
		} else if child.Tag == "alias" && child.Attr("source") == "locale" {
			src := n
			path := child.Attr("path")
			for 0 < len(path) {
				slash, inVal := -1, false
				for i := 0; i < len(path); i++ {
					if !inVal && path[i] == '/' {
						slash = i
						break
					} else if path[i] == '\'' {
						inVal = !inVal
					}
				}
				elem := path
				if slash != -1 {
					elem = path[:slash]
					path = path[slash+1:]
				}
				if elem == ".." {
					src = src.Parent
				} else {
					query := elem
					if bracket := strings.IndexByte(elem, '['); bracket != -1 {
						query = elem[:bracket]
						for bracket < len(elem) {
							if bracket+1 == len(elem) || elem[bracket+1] != '@' {
								fmt.Println("WARN: unsupported XPath:", elem)
								break
							} else if is := strings.IndexByte(elem[bracket+2:], '='); is == -1 || is == 0 || bracket+2+is+1 == len(elem) || elem[bracket+2+is+1] != '\'' {
								fmt.Println("WARN: unsupported XPath:", elem)
								break
							} else if quote := strings.IndexByte(elem[bracket+2+is+2:], '\''); quote == -1 || quote == 0 || bracket+2+is+2+quote+1 == len(elem) || elem[bracket+2+is+2+quote+1] != ']' {
								fmt.Println("WARN: unsupported XPath:", elem)
								break
							} else {
								query += fmt.Sprintf("[%v=%v]", elem[bracket+2:bracket+2+is], elem[bracket+2+is+2:bracket+2+is+2+quote])
								bracket = bracket + 2 + is + 2 + quote + 2
							}
						}
					}

					if src2, ok := src.Find(query); !ok {
						if query == "dateTimeFormat[type=standard]" {
							query = "dateTimeFormat[!type]"
							if src2, ok := src.Find(query); !ok {
								fmt.Printf("WARN: alias not found in %v: %v\n", n.Path(), child.Attr("path"))
								return
							} else {
								src = src2
							}
						} else {
							fmt.Printf("WARN: alias not found in %v: %v\n", n.Path(), child.Attr("path"))
							return
						}
					} else {
						src = src2
					}
				}
				if slash == -1 {
					break
				}
			}
			if src == n {
				fmt.Printf("WARN: alias not found in %v: %v\n", n.Path(), child.Attr("path"))
			} else {
				nodes := make([]*XMLNode, len(src.Nodes))
				for i, node := range src.Nodes {
					node2 := *node
					node2.Parent = n
					nodes[i] = &node2
				}
				n.Nodes = append(n.Nodes[:i], append(nodes, n.Nodes[i+1:]...)...)
			}
		} else {
			child.ResolveAliases()
		}
	}
}

func (n *XMLNode) InheritFrom(parent *XMLNode) {
	for i, j := 0, 0; j < len(parent.Nodes); j++ {
		var cmp int
		for i < len(n.Nodes) {
			if cmp = parent.Nodes[j].Compare(n.Nodes[i]); cmp != 1 {
				break
			}
			i++
		}
		if i == len(n.Nodes) || cmp == -1 {
			n.Nodes = append(n.Nodes[:i], append([]*XMLNode{parent.Nodes[j]}, n.Nodes[i:]...)...)
			i++
		} else if cmp == 0 {
			n.Nodes[i].InheritFrom(parent.Nodes[j])
			i++
		}
	}
}

func (n *XMLNode) StringTo(w io.Writer) {
	w.Write([]byte("<" + n.Tag))
	for _, attr := range n.Attrs {
		w.Write([]byte(" " + attr[0] + "=\"" + attr[1] + "\""))
	}
	if len(n.Text) == 0 && len(n.Nodes) == 0 {
		w.Write([]byte("/>"))
	} else {
		w.Write([]byte(">" + n.Text))
		if 0 < len(n.Nodes) {
			wi := NewPrefixer(w, "  ")
			for _, child := range n.Nodes {
				wi.Write([]byte("\n"))
				child.StringTo(wi)
			}
			w.Write([]byte("\n"))
		}
		w.Write([]byte("</" + n.Tag + ">"))
	}
}

func (n *XMLNode) String() string {
	sb := strings.Builder{}
	n.StringTo(&sb)
	return sb.String()
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

func printType(w io.Writer, t reflect.Type, level int) error {
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
		if err := printType(w, t.Elem(), level+1); err != nil {
			return fmt.Errorf("array: %v", err)
		}
	case reflect.Slice:
		fmt.Fprintf(w, "[]")
		if err := printType(w, t.Elem(), level+1); err != nil {
			return fmt.Errorf("slice: %v", err)
		}
	case reflect.Map:
		fmt.Fprintf(w, "map[")
		if err := printType(w, t.Key(), level+1); err != nil {
			return fmt.Errorf("map key: %v", err)
		}
		fmt.Fprintf(w, "]")
		if err := printType(w, t.Elem(), level+1); err != nil {
			return fmt.Errorf("map value: %v", err)
		}
	case reflect.String:
		fmt.Fprintf(w, "string")
	case reflect.Struct:
		if 0 < level {
			fmt.Fprintf(w, t.Name())
		} else {
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
				if err := printType(wi, field.Type, level+1); err != nil {
					return fmt.Errorf("struct field %v: %v", field.Name, err)
				}
			}
			if 0 < n {
				fmt.Fprintf(w, "\n")
			}
			fmt.Fprintf(w, "}")
		}
	default:
		return fmt.Errorf("unsupported type: %v", t)
	}
	return nil
}

func printValue(w io.Writer, v reflect.Value, level int) error {
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
			if err := printValue(wi, v.Index(i), level+1); err != nil {
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
			if err := printValue(wi, keys[i], level+1); err != nil {
				return fmt.Errorf("map key %v: %v", keys[i], err)
			}
			fmt.Fprintf(wi, ": ")
			if err := printValue(wi, v.MapIndex(keys[i]), level+1); err != nil {
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
				if err := printType(wi, field.Type(), level+1); err != nil {
					return fmt.Errorf("struct field %v: %v", v.Type().Field(i), err)
				}
			}
			if err := printValue(wi, field, level+1); err != nil {
				return fmt.Errorf("struct field %v: %v", v.Type().Field(i), err)
			}
		}
		fmt.Fprintf(w, "}")
	default:
		return fmt.Errorf("unsupported value: %v", v)
	}
	return nil
}
