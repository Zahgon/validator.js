package validator

import (
	"strings"

	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isMobilePhone", isMobilePhone)
	registerValue("isMobilePhoneLocales", func() js.Value { return localeKeys(phonePatterns) })
	register("isPostalCode", isPostalCode)
	registerValue("isPostalCodeLocales", func() js.Value { return localeKeys(postalPatterns) })
	register("isCurrency", isCurrency)
}

// lookupPattern finds the pattern registered under a property key.
func lookupPattern(table []localePattern, key string) (localePattern, bool) {
	for _, e := range table {
		if e.locale == key {
			return e, true
		}
	}
	return localePattern{}, false
}

func anyPattern(table []localePattern, s string) bool {
	for _, e := range table {
		if e.re.Test(s) {
			return true
		}
	}
	return false
}

func isMobilePhone(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	locale, options := arg(a, 1), arg(a, 2)
	if js.Truthy(options) && optTrue(options, "strictMode") && !strings.HasPrefix(s, "+") {
		return js.Bool(false)
	}
	if locale.IsArray() {
		for _, key := range locale.Obj().Elems {
			if p, ok := lookupPattern(phonePatterns, js.ToPropertyKey(key)); ok && p.re.Test(s) {
				return js.Bool(true)
			}
		}
		return js.Bool(false)
	}
	key := js.ToPropertyKey(locale)
	if p, ok := lookupPattern(phonePatterns, key); ok {
		return js.Bool(p.re.Test(s))
	}
	if js.IsObjectProtoKey(key) {
		js.Throw("TypeError", "phones[locale].test is not a function")
	}
	if !js.Truthy(locale) || (locale.IsString() && locale.String() == "any") {
		return js.Bool(anyPattern(phonePatterns, s))
	}
	js.Throw("Error", "Invalid locale '"+js.ToString(locale)+"'")
	return js.Undefined
}

func isPostalCode(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	locale := arg(a, 1)
	key := js.ToPropertyKey(locale)
	if p, ok := lookupPattern(postalPatterns, key); ok {
		return js.Bool(p.re.Test(s))
	}
	if js.IsObjectProtoKey(key) {
		js.Throw("TypeError", "patterns[locale].test is not a function")
	}
	if locale.IsString() && locale.String() == "any" {
		return js.Bool(anyPattern(postalPatterns, s))
	}
	js.Throw("Error", "Invalid locale '"+js.ToString(locale)+"'")
	return js.Undefined
}

func currencyDefaults() []option {
	return []option{{"symbol", js.Str("$")}, {"require_symbol", js.Bool(false)},
		{"allow_space_after_symbol", js.Bool(false)}, {"symbol_after_digits", js.Bool(false)},
		{"allow_negatives", js.Bool(true)}, {"parens_for_negatives", js.Bool(false)},
		{"negative_sign_before_digits", js.Bool(false)}, {"negative_sign_after_digits", js.Bool(false)},
		{"allow_negative_sign_placeholder", js.Bool(false)}, {"thousands_separator", js.Str(",")},
		{"decimal_separator", js.Str(".")}, {"allow_decimal", js.Bool(true)}, {"require_decimal", js.Bool(false)},
		{"digits_after_decimal", js.NewArray(js.Num(2))}, {"allow_space_after_digits", js.Bool(false)}}
}

var reNonWord = re(`\W`, "")

// currencyPattern assembles the pattern source of currencyRegex from the merged options.
func currencyPattern(options js.Value) string {
	digits := opt(options, "digits_after_decimal")
	decimalDigits := `\d{` + js.ToString(index(digits, 0)) + `}`
	if !digits.IsArray() {
		js.Get(digits, "forEach")
		js.Throw("TypeError", "options.digits_after_decimal.forEach is not a function")
	}
	for i, d := range digits.Obj().Elems {
		if i != 0 {
			decimalDigits = decimalDigits + `|\d{` + js.ToString(d) + `}`
		}
	}
	symbolOpt := opt(options, "symbol")
	if !symbolOpt.IsString() {
		js.Get(symbolOpt, "replace")
		js.Throw("TypeError", "options.symbol.replace is not a function")
	}
	symbol := "(" + replaceFunc(symbolOpt.String(), reNonWord, func(m match) string { return `\` + m.text }) + ")"
	if !optTrue(options, "require_symbol") {
		symbol += "?"
	}
	const negative = "-?"
	withSep := `[1-9]\d{0,2}(\` + js.ToString(opt(options, "thousands_separator")) + `\d{3})*`
	whole := "(" + strings.Join([]string{"0", `[1-9]\d*`, withSep}, "|") + ")?"
	decimalAmount := `(\` + js.ToString(opt(options, "decimal_separator")) + "(" + decimalDigits + "))"
	if !optTrue(options, "require_decimal") {
		decimalAmount += "?"
	}
	pattern := whole
	if optTrue(options, "allow_decimal") || optTrue(options, "require_decimal") {
		pattern += decimalAmount
	}
	if optTrue(options, "allow_negatives") && !optTrue(options, "parens_for_negatives") {
		if optTrue(options, "negative_sign_after_digits") {
			pattern += negative
		} else if optTrue(options, "negative_sign_before_digits") {
			pattern = negative + pattern
		}
	}
	switch {
	case optTrue(options, "allow_negative_sign_placeholder"):
		pattern = `( (?!\-))?` + pattern
	case optTrue(options, "allow_space_after_symbol"):
		pattern = " ?" + pattern
	case optTrue(options, "allow_space_after_digits"):
		pattern += "( (?!$))?"
	}
	if optTrue(options, "symbol_after_digits") {
		pattern += symbol
	} else {
		pattern = symbol + pattern
	}
	if optTrue(options, "allow_negatives") {
		if optTrue(options, "parens_for_negatives") {
			pattern = `(\(` + pattern + `\)|` + pattern + ")"
		} else if !(optTrue(options, "negative_sign_before_digits") || optTrue(options, "negative_sign_after_digits")) {
			pattern = negative + pattern
		}
	}
	return `^(?!-? )(?=.*\d)` + pattern + "$"
}

func isCurrency(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := merge(arg(a, 1), currencyDefaults())
	rx := newRegExp(js.Str(currencyPattern(options)), js.Undefined)
	return js.Bool(regexpTest(rx, js.Str(s)))
}
