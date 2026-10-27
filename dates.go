package validator

import (
	"math"
	"strings"
	"time"

	"github.com/validatorjs/validator.js/internal/jsdate"
	"github.com/validatorjs/validator.js/internal/jsre"
	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isDate", func(a []js.Value) js.Value { return js.Bool(isDate(arg(a, 0), arg(a, 1))) })
	register("isAfter", func(a []js.Value) js.Value {
		return js.Bool(compareDates(arg(a, 0), arg(a, 1), func(x, y float64) bool { return x > y }))
	})
	register("isBefore", func(a []js.Value) js.Value {
		return js.Bool(compareDates(arg(a, 0), arg(a, 1), func(x, y float64) bool { return x < y }))
	})
	register("isTime", isTime)
	registerPattern("isRFC3339", reRFC3339)
	register("isISO8601", isISO8601)
}

// compareDates implements isAfter and isBefore: the comparison date defaults to the current time as Date()
// prints it, and both sides go through toDate.
func compareDates(date js.Value, options js.Value, cmp func(x, y float64) bool) bool {
	var cd js.Value
	if js.TypeOf(options) == "object" {
		cd = js.Get(options, "comparisonDate")
	} else {
		cd = options
	}
	if !js.Truthy(cd) {
		cd = js.Str(jsdate.ToDateString(jsdate.Now()))
	}
	comparison := toDate(cd)
	original := toDate(date)
	if original.IsNull() || comparison.IsNull() {
		return false
	}
	return cmp(original.Obj().Time, comparison.Obj().Time)
}

var reDateFormat = `(^(y{4}|y{2})[.\/-](m{1,2})[.\/-](d{1,2})$)|(^(m{1,2})[.\/-](d{1,2})[.\/-]((y{4}|y{2})$))|(^(d{1,2})[.\/-](m{1,2})[.\/-]((y{4}|y{2})$))`

// jsSplit implements String.prototype.split(separator) for any separator value.
func jsSplit(s string, sep js.Value) []string {
	switch {
	case sep.IsUndefined():
		return []string{s}
	case sep.IsRegExp():
		return splitRe(s, compiledOf(sep.Obj()))
	}
	return js.Split16(s, js.ToString(sep))
}

// stringMethodTarget returns v when it is a string and otherwise fails like calling a missing String method.
func stringMethodTarget(v js.Value, receiver, method string) string {
	if !v.IsString() {
		js.Get(v, method)
		js.Throw("TypeError", receiver+"."+method+" is not a function")
	}
	return v.String()
}

// findDelimiter implements delimiters.find(d => haystack.indexOf(d) !== -1).
func findDelimiter(delimiters js.Value, haystack js.Value) js.Value {
	if !delimiters.IsArray() {
		js.Get(delimiters, "find")
		js.Throw("TypeError", "options.delimiters.find is not a function")
	}
	for _, d := range delimiters.Obj().Elems {
		if indexOfNamed(haystack, d, "indexOf") != -1 {
			return d
		}
	}
	return js.Undefined
}

func dateDefaults() []option {
	return []option{{"format", js.Str("YYYY/MM/DD")}, {"delimiters", js.NewArray(js.Str("/"), js.Str("-"))},
		{"strictMode", js.Bool(false)}}
}

func isDate(input js.Value, options js.Value) bool {
	if options.IsString() {
		o := js.NewObject()
		o.Obj().SetProp("format", options)
		options = merge(o, dateDefaults())
	} else {
		options = merge(options, dateDefaults())
	}
	format := opt(options, "format")
	if input.IsString() && jsre.MustCompile(reDateFormat, "gi").Test(js.ToString(format)) {
		in := input.String()
		strict := optTrue(options, "strictMode")
		if strict && !js.StrictEquals(js.Num(float64(js.Len16(in))), js.Get(format, "length")) {
			return false
		}
		formatDelimiter := findDelimiter(opt(options, "delimiters"), format)
		dateDelimiter := formatDelimiter
		if !strict {
			dateDelimiter = findDelimiter(opt(options, "delimiters"), input)
		}
		dateWords := jsSplit(in, dateDelimiter)
		formatWords := jsSplit(js.ToLowerCase(stringMethodTarget(format, "options.format", "toLowerCase")), formatDelimiter)
		dateObj := map[string]string{}
		n := len(dateWords)
		if len(formatWords) > n {
			n = len(formatWords)
		}
		for i := 0; i < n; i++ {
			if i >= len(dateWords) || i >= len(formatWords) || dateWords[i] == "" || formatWords[i] == "" ||
				js.Len16(dateWords[i]) != js.Len16(formatWords[i]) {
				return false
			}
			dateObj[js.CharAt(formatWords[i], 0)] = dateWords[i]
		}
		field := func(k string) string {
			v, ok := dateObj[k]
			if !ok {
				js.Throw("TypeError", "Cannot read properties of undefined (reading 'length')")
			}
			return v
		}
		y, ok := dateObj["y"]
		if !ok {
			js.Throw("TypeError", "Cannot read properties of undefined (reading 'startsWith')")
		}
		if strings.HasPrefix(y, "-") {
			return false
		}
		fullYear := y
		if js.Len16(y) == 2 {
			parsed := js.ParseInt(js.Str(y), js.Num(10))
			if math.IsNaN(parsed) {
				return false
			}
			if parsed < float64(time.Now().UTC().Year()%100) {
				fullYear = "20" + y
			} else {
				fullYear = "19" + y
			}
		}
		month := field("m")
		if js.Len16(month) == 1 {
			month = "0" + month
		}
		day := field("d")
		if js.Len16(day) == 1 {
			day = "0" + day
		}
		t := dateParse(fullYear + "-" + month + "-" + day + "T00:00:00.000Z")
		got := math.NaN()
		if !math.IsNaN(t) {
			got = float64(jsdate.Split(t).Day)
		}
		return got == js.StringToNumber(dateObj["d"])
	}
	if !optTrue(options, "strictMode") {
		return input.IsDate() && !math.IsNaN(input.Obj().Time) && !math.IsInf(input.Obj().Time, 0)
	}
	return false
}

var timeFormats = map[string]map[string]*jsre.Regexp{
	"hour24": {
		"default":             re(`^([01]?[0-9]|2[0-3]):([0-5][0-9])$`, ""),
		"withSeconds":         re(`^([01]?[0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])$`, ""),
		"withOptionalSeconds": re(`^([01]?[0-9]|2[0-3]):([0-5][0-9])(?::([0-5][0-9]))?$`, ""),
	},
	"hour12": {
		"default":             re(`^(0?[1-9]|1[0-2]):([0-5][0-9]) (A|P)M$`, ""),
		"withSeconds":         re(`^(0?[1-9]|1[0-2]):([0-5][0-9]):([0-5][0-9]) (A|P)M$`, ""),
		"withOptionalSeconds": re(`^(0?[1-9]|1[0-2]):([0-5][0-9])(?::([0-5][0-9]))? (A|P)M$`, ""),
	},
}

func isTime(a []js.Value) js.Value {
	options := merge(arg(a, 1), []option{{"hourFormat", js.Str("hour24")}, {"mode", js.Str("default")}})
	input := arg(a, 0)
	if !input.IsString() {
		return js.Bool(false)
	}
	hourKey := js.ToPropertyKey(opt(options, "hourFormat"))
	modes, ok := timeFormats[hourKey]
	modeKey := js.ToPropertyKey(opt(options, "mode"))
	if !ok {
		if js.IsObjectProtoKey(hourKey) {
			js.Throw("TypeError", "Cannot read properties of undefined (reading 'test')")
		}
		js.Throw("TypeError", "Cannot read properties of undefined (reading '"+modeKey+"')")
	}
	r, ok := modes[modeKey]
	if !ok {
		if js.IsObjectProtoKey(modeKey) {
			js.Throw("TypeError", "formats[options.hourFormat][options.mode].test is not a function")
		}
		js.Throw("TypeError", "Cannot read properties of undefined (reading 'test')")
	}
	return js.Bool(test(r, input.String()))
}

const (
	rfcTimeHour    = `([01][0-9]|2[0-3])`
	rfcTimeMinute  = `[0-5][0-9]`
	rfcPartialTime = rfcTimeHour + `:` + rfcTimeMinute + `:([0-5][0-9]|60)(\.[0-9]+)?`
	rfcTimeOffset  = `([zZ]|[-+]` + rfcTimeHour + `:` + rfcTimeMinute + `)`
	rfcFullDate    = `[0-9]{4}-(0[1-9]|1[0-2])-([12]\d|0[1-9]|3[01])`
)

var reRFC3339 = re(`^`+rfcFullDate+`[ tT]`+rfcPartialTime+rfcTimeOffset+`$`, "")

var reISO8601 = re(`^([\+-]?\d{4}(?!\d{2}\b))((-?)((0[1-9]|1[0-2])(\3([12]\d|0[1-9]|3[01]))?|W(0[1-9]|[1-4]\d|5[0-3])(-?[1-7])?|(00[1-9]|0[1-9]\d|[12]\d{2}|3([0-5]\d|6[0-6])))([T\s]((([01]\d|2[0-3])((:?)[0-5]\d)?|24:?00)([\.,]\d+(?!:))?)?(\17[0-5]\d([\.,]\d+)?)?([zZ]|([\+-])([01]\d|2[0-3]):?([0-5]\d)?)?)?)?$`, "")
var reISO8601Strict = re(`^([\+-]?\d{4}(?!\d{2}\b))((-?)((0[1-9]|1[0-2])(\3([12]\d|0[1-9]|3[01]))?|W(0[1-9]|[1-4]\d|5[0-3])(-?[1-7])?|(00[1-9]|0[1-9]\d|[12]\d{2}|3([0-5]\d|6[0-6])))([T]((([01]\d|2[0-3])((:?)[0-5]\d)?|24:?00)([\.,]\d+(?!:))?)?(\17[0-5]\d([\.,]\d+)?)?([zZ]|([\+-])([01]\d|2[0-3]):?([0-5]\d)?)?)?)?$`, "")
var reOrdinalDate = re(`^(\d{4})-?(\d{3})([ T]{1}\.*|$)`, "")
var reCalendarDate = re(`(\d{4})-?(\d{0,2})-?(\d*)`, "")

// lastTwo implements s.slice(-2).
func lastTwo(s string) string { return js.SliceJS(s, -2, js.Len16(s)) }

func isValidISODate(s string) bool {
	if m, ok := firstMatch(s, reOrdinalDate); ok {
		year := js.StringToNumber(*m.caps[0])
		day := js.StringToNumber(*m.caps[1])
		if (math.Mod(year, 4) == 0 && math.Mod(year, 100) != 0) || math.Mod(year, 400) == 0 {
			return day <= 366
		}
		return day <= 365
	}
	m, _ := firstMatch(s, reCalendarDate)
	year := js.StringToNumber(*m.caps[0])
	month := js.StringToNumber(*m.caps[1])
	day := js.StringToNumber(*m.caps[2])
	monthString := js.NumberToString(month)
	if month != 0 {
		monthString = lastTwo("0" + monthString)
	} else {
		monthString = "01"
	}
	dayString := js.NumberToString(day)
	if day != 0 {
		dayString = lastTwo("0" + dayString)
	} else {
		dayString = "01"
	}
	t := dateParse(js.NumberToString(year) + "-" + monthString + "-" + dayString)
	if month != 0 && day != 0 {
		if math.IsNaN(t) {
			return false
		}
		f := jsdate.Split(t)
		return float64(f.Year) == year && float64(f.Month+1) == month && float64(f.Day) == day
	}
	return true
}

func isISO8601(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	if options.IsUndefined() {
		options = js.NewObject()
	}
	var check bool
	if optTrue(options, "strictSeparator") {
		check = test(reISO8601Strict, s)
	} else {
		check = test(reISO8601, s)
	}
	if check && optTrue(options, "strict") {
		return js.Bool(isValidISODate(s))
	}
	return js.Bool(check)
}
