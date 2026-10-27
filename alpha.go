package validator

import (
	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isAlpha", func(a []js.Value) js.Value { return js.Bool(isAlphaLike(a, alphaPatterns, "alpha")) })
	register("isAlphanumeric", func(a []js.Value) js.Value {
		return js.Bool(isAlphaLike(a, alphanumericPatterns, "alphanumeric"))
	})
	registerValue("isAlphaLocales", func() js.Value { return localeKeys(alphaPatterns) })
	registerValue("isAlphanumericLocales", func() js.Value { return localeKeys(alphanumericPatterns) })
	registerPattern("isLocale", reLanguageTag)
}

var reIgnoreEscape = re(`[-[\]{}()*+?.,\\^$|#\\s]`, "g")

// isAlphaLike implements isAlpha and isAlphanumeric over their locale table.
func isAlphaLike(a []js.Value, table []localePattern, tableName string) bool {
	str := assertString(arg(a, 0))
	locale := arg(a, 1)
	if locale.IsUndefined() {
		locale = js.Str("en-US")
	}
	options := arg(a, 2)
	if options.IsUndefined() {
		options = js.NewObject()
	}
	if options.IsNullish() {
		js.Throw("TypeError", "Cannot destructure property 'ignore' of 'options' as it is null.")
	}
	ignore := js.Get(options, "ignore")
	if js.Truthy(ignore) {
		switch {
		case ignore.IsRegExp():
			str = replaceRegExp(str, ignore, "")
		case ignore.IsString():
			class := "[" + replace(ignore.String(), reIgnoreEscape, `\$&`) + "]"
			str = replaceRegExp(str, newRegExp(js.Str(class), js.Str("g")), "")
		default:
			js.Throw("Error", "ignore should be instance of a String or RegExp")
		}
	}
	key := js.ToPropertyKey(locale)
	for _, e := range table {
		if e.locale == key {
			return e.re.Test(str)
		}
	}
	if js.IsObjectProtoKey(key) {
		js.Throw("TypeError", tableName+"[locale].test is not a function")
	}
	js.Throw("Error", "Invalid locale '"+js.ToString(locale)+"'")
	return false
}

const (
	localeExtlang    = `([A-Za-z]{3}(-[A-Za-z]{3}){0,2})`
	localeLanguage   = `(([a-zA-Z]{2,3}(-` + localeExtlang + `)?)|([a-zA-Z]{5,8}))`
	localeScript     = `([A-Za-z]{4})`
	localeRegion     = `([A-Za-z]{2}|\d{3})`
	localeVariant    = `([A-Za-z0-9]{5,8}|(\d[A-Z-a-z0-9]{3}))`
	localeSingleton  = `(\d|[A-W]|[Y-Z]|[a-w]|[y-z])`
	localeExtension  = `(` + localeSingleton + `(-[A-Za-z0-9]{2,8})+)`
	localePrivateuse = `(x(-[A-Za-z0-9]{1,8})+)`
	localeIrregular  = `((en-GB-oed)|(i-ami)|(i-bnn)|(i-default)|(i-enochian)|` +
		`(i-hak)|(i-klingon)|(i-lux)|(i-mingo)|(i-navajo)|(i-pwn)|(i-tao)|` +
		`(i-tay)|(i-tsu)|(sgn-BE-FR)|(sgn-BE-NL)|(sgn-CH-DE))`
	localeRegular = `((art-lojban)|(cel-gaulish)|(no-bok)|(no-nyn)|(zh-guoyu)|` +
		`(zh-hakka)|(zh-min)|(zh-min-nan)|(zh-xiang))`
	localeGrandfathered = `(` + localeIrregular + `|` + localeRegular + `)`
	localeDelimiter     = `(-|_)`
	localeLangtag       = localeLanguage + `(` + localeDelimiter + localeScript + `)?(` + localeDelimiter +
		localeRegion + `)?(` + localeDelimiter + localeVariant + `)*(` + localeDelimiter + localeExtension +
		`)*(` + localeDelimiter + localePrivateuse + `)?`
)

var reLanguageTag = re(`(^`+localePrivateuse+`$)|(^`+localeGrandfathered+`$)|(^`+localeLangtag+`$)`, "")
