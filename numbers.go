package validator

import (
	"math"
	"strings"

	"github.com/validatorjs/validator.js/internal/jsre"
	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isInt", func(a []js.Value) js.Value { return js.Bool(isInt(arg(a, 0), arg(a, 1))) })
	register("isFloat", func(a []js.Value) js.Value { return js.Bool(isFloat(arg(a, 0), arg(a, 1))) })
	register("toFloat", func(a []js.Value) js.Value { return js.Num(toFloat(arg(a, 0))) })
	register("isDecimal", isDecimal)
	register("isNumeric", isNumeric)
	register("isDivisibleBy", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Bool(math.Mod(toFloat(js.Str(s)), js.ParseInt(arg(a, 1), js.Num(10))) == 0)
	})
	register("isPort", func(a []js.Value) js.Value {
		options := js.NewObject()
		options.Obj().SetProp("allow_leading_zeroes", js.Bool(false))
		options.Obj().SetProp("min", js.Num(0))
		options.Obj().SetProp("max", js.Num(65535))
		return js.Bool(isInt(arg(a, 0), options))
	})
	registerPattern("isHexadecimal", re(`^(0x|0h)?[0-9A-F]+$`, "i"))
	registerPattern("isOctal", re(`^(0o)?[0-7]+$`, "i"))
	registerValue("isFloatLocales", func() js.Value { return localeKeys(decimalMarks) })
}

type localePattern struct {
	locale string
	re     *jsre.Regexp
}

type localeString struct {
	locale string
	value  string
}

func localeKeys[T localePattern | localeString](table []T) js.Value {
	keys := make([]js.Value, len(table))
	for i, e := range table {
		switch x := any(e).(type) {
		case localePattern:
			keys[i] = js.Str(x.locale)
		case localeString:
			keys[i] = js.Str(x.locale)
		}
	}
	return js.NewArray(keys...)
}

// decimalOf reads decimal[key] with property semantics: a known locale gives its mark, an Object.prototype member
// gives a function and anything else undefined.
func decimalOf(key js.Value) js.Value {
	k := js.ToPropertyKey(key)
	for _, e := range decimalMarks {
		if e.locale == k {
			return js.Str(e.value)
		}
	}
	if js.IsObjectProtoKey(k) {
		return js.NewFunction(k)
	}
	return js.Undefined
}

// inDecimal implements `key in decimal`.
func inDecimal(key js.Value) bool {
	k := js.ToPropertyKey(key)
	for _, e := range decimalMarks {
		if e.locale == k {
			return true
		}
	}
	return js.IsObjectProtoKey(k)
}

// hasOwn calls options.hasOwnProperty(key), failing like JavaScript when that is not a function.
func hasOwn(options js.Value, key string) bool {
	if f := js.Get(options, "hasOwnProperty"); !f.IsFunction() {
		js.Throw("TypeError", "options.hasOwnProperty is not a function")
	}
	return js.HasOwnProperty(options, key)
}

// boundOK evaluates `!options.hasOwnProperty(key) || isNullOrUndefined(options[key]) || cmp(value, options[key])`.
func boundOK(options js.Value, key string, value js.Value, cmp func(a, b js.Value) bool) bool {
	if !hasOwn(options, key) {
		return true
	}
	b := js.Get(options, key)
	return b.IsNullish() || cmp(value, b)
}

var reInt = re(`^(?:[-+]?(?:0|[1-9][0-9]*))$`, "")
var reIntLeadingZeroes = re(`^[-+]?[0-9]+$`, "")

func isInt(str js.Value, options js.Value) bool {
	s := assertString(str)
	if !js.Truthy(options) {
		options = js.NewObject()
	}
	r := reIntLeadingZeroes
	if v := js.Get(options, "allow_leading_zeroes"); v.IsBoolean() && !v.Boolean() {
		r = reInt
	}
	sv := js.Str(s)
	minOK := boundOK(options, "min", sv, js.GreaterEq)
	maxOK := boundOK(options, "max", sv, js.LessEq)
	ltOK := boundOK(options, "lt", sv, js.Less)
	gtOK := boundOK(options, "gt", sv, js.Greater)
	return test(r, s) && minOK && maxOK && ltOK && gtOK
}

func isFloat(str js.Value, options js.Value) bool {
	s := assertString(str)
	if !js.Truthy(options) {
		options = js.NewObject()
	}
	mark := js.Str(".")
	if locale := js.Get(options, "locale"); js.Truthy(locale) {
		mark = decimalOf(locale)
	}
	float := newRegExp(js.Str(`^(?:[-+])?(?:[0-9]+)?(?:\`+js.ToString(mark)+`[0-9]*)?(?:[eE][\+\-]?(?:[0-9]+))?$`),
		js.Undefined)
	switch s {
	case "", ".", ",", "-", "+":
		return false
	}
	value := js.Num(js.ParseFloat(js.Str(js.ReplaceFirst(s, ",", "."))))
	return regexpTest(float, js.Str(s)) &&
		boundOK(options, "min", value, js.GreaterEq) &&
		boundOK(options, "max", value, js.LessEq) &&
		boundOK(options, "lt", value, js.Less) &&
		boundOK(options, "gt", value, js.Greater)
}

func toFloat(str js.Value) float64 {
	if !isFloat(str, js.Undefined) {
		return math.NaN()
	}
	return js.ParseFloat(str)
}

func isDecimal(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := merge(arg(a, 1), []option{{"force_decimal", js.Bool(false)}, {"decimal_digits", js.Str("1,")},
		{"locale", js.Str("en-US")}})
	locale := opt(options, "locale")
	if !inDecimal(locale) {
		js.Throw("Error", "Invalid locale '"+js.ToString(locale)+"'")
	}
	switch strings.ReplaceAll(s, " ", "") {
	case "", "-", "+":
		return js.Bool(false)
	}
	optional := "?"
	if optTrue(options, "force_decimal") {
		optional = ""
	}
	pattern := `^[-+]?([0-9]+)?(\` + js.ToString(decimalOf(locale)) + `[0-9]{` +
		js.ToString(opt(options, "decimal_digits")) + `})` + optional + `$`
	return js.Bool(regexpTest(newRegExp(js.Str(pattern), js.Undefined), js.Str(s)))
}

var reNumericNoSymbols = re(`^[0-9]+$`, "")

func isNumeric(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	if js.Truthy(options) && optTrue(options, "no_symbols") {
		return js.Bool(test(reNumericNoSymbols, s))
	}
	mark := js.Str(".")
	base := options
	if !js.Truthy(base) {
		base = js.NewObject()
	}
	if js.Truthy(js.Get(base, "locale")) {
		mark = decimalOf(js.Get(options, "locale"))
	}
	pattern := `^[+-]?([0-9]*[` + js.ToString(mark) + `])?[0-9]+$`
	return js.Bool(regexpTest(newRegExp(js.Str(pattern), js.Undefined), js.Str(s)))
}
