package validator

import (
	"math"
	"strings"

	"github.com/validatorjs/validator.js/internal/jsre"
	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isCreditCard", isCreditCard)
	register("isLuhnNumber", func(a []js.Value) js.Value { return js.Bool(isLuhnNumber(assertString(arg(a, 0)))) })
	register("isIBAN", isIBAN)
	registerValue("ibanLocales", func() js.Value { return localeKeys(ibanPatterns) })
	register("isBIC", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		cc := js.ToUpperCase(js.SliceJS(s, 4, 6))
		if !iso31661Alpha2Codes[cc] && cc != "XK" {
			return js.Bool(false)
		}
		return js.Bool(test(reBIC, s))
	})
	register("isEAN", isEAN)
	register("isISIN", isISIN)
	register("isISBN", func(a []js.Value) js.Value { return js.Bool(isISBN(assertString(arg(a, 0)), arg(a, 1))) })
	register("isISSN", isISSN)
	register("isIMEI", isIMEI)
	register("isAbaRouting", isAbaRouting)
	register("isISO6346", isISO6346)
	register("isFreightContainerID", isISO6346)
	registerPattern("isEthereumAddress", re(`^(0x)[0-9a-f]{40}$`, "i"))
	register("isBtcAddress", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Bool(test(reBech32, s) || test(reBase58Address, s))
	})
}

var cardPatterns = []localePattern{
	{"amex", re(`^3[47][0-9]{13}$`, "")},
	{"dinersclub", re(`^3(?:0[0-5]|[68][0-9])[0-9]{11}$`, "")},
	{"discover", re(`^6(?:011|5[0-9][0-9])[0-9]{12,15}$`, "")},
	{"jcb", re(`^(?:2131|1800|35\d{3})\d{11}$`, "")},
	{"mastercard", re(`^5[1-5][0-9]{2}|(222[1-9]|22[3-9][0-9]|2[3-6][0-9]{2}|27[01][0-9]|2720)[0-9]{12}$`, "")},
	{"unionpay", re(`^(6[27][0-9]{14}|^(81[0-9]{14,17}))$`, "")},
	{"visa", re(`^(?:4[0-9]{12})(?:[0-9]{3,6})?$`, "")},
}

var reCardSeparators = re(`[- ]+`, "g")
var reBIC = re(`^[A-Za-z]{6}[A-Za-z0-9]{2}([A-Za-z0-9]{3})?$`, "")
var reBech32 = re(`^(bc1|tb1|bc1p|tb1p)[ac-hj-np-z02-9]{39,58}$`, "")
var reBase58Address = re(`^(1|2|3|m)[A-HJ-NP-Za-km-z1-9]{25,39}$`, "")

func isCreditCard(a []js.Value) js.Value {
	card := assertString(arg(a, 0))
	options := arg(a, 1)
	if options.IsUndefined() {
		options = js.NewObject()
	}
	if options.IsNull() {
		js.Throw("TypeError", "Cannot destructure property 'provider' of 'options' as it is null.")
	}
	provider := js.Get(options, "provider")
	sanitized := replace(card, reCardSeparators, "")
	if js.Truthy(provider) {
		key := js.ToLowerCase(stringMethodTarget(provider, "provider", "toLowerCase"))
		p, ok := lookupPattern(cardPatterns, key)
		switch {
		case ok:
			if !p.re.Test(sanitized) {
				return js.Bool(false)
			}
		case js.IsObjectProtoKey(key):
			js.Throw("TypeError", "cards[provider.toLowerCase(...)].test is not a function")
		default:
			js.Throw("Error", js.ToString(provider)+" is not a valid credit card provider.")
		}
	} else if !anyPattern(cardPatterns, sanitized) {
		return js.Bool(false)
	}
	return js.Bool(isLuhnNumber(card))
}

func isLuhnNumber(s string) bool {
	sanitized := js.ToU16(replace(s, reCardSeparators, ""))
	sum := 0.0
	double := false
	for i := len(sanitized) - 1; i >= 0; i-- {
		n := js.ParseInt(js.Str(js.FromU16(sanitized[i:i+1])), js.Num(10))
		if double {
			n *= 2
			if n >= 10 {
				sum += math.Mod(n, 10) + 1
			} else {
				sum += n
			}
		} else {
			sum += n
		}
		double = !double
	}
	return math.Mod(sum, 10) == 0 && len(sanitized) > 0
}

var reIBANSeparators = re(`[\s\-]+`, "gi")
var reIBANStrip = re(`[^A-Z0-9]+`, "gi")
var reCapitalLetter = re(`[A-Z]`, "g")
var reDigitRuns = re(`\d{1,7}`, "g")

// arrayIncludes implements includes(arr, val) = arr.some(v => v === val), failing for a non-array.
func arrayIncludes(arr js.Value, val js.Value, expr string) bool {
	if !arr.IsArray() {
		js.Get(arr, "some")
		js.Throw("TypeError", expr+" is not a function")
	}
	return includesArray(arr, val)
}

func hasValidIBANFormat(s string, options js.Value) bool {
	stripped := js.ToUpperCase(replace(s, reIBANSeparators, ""))
	code := js.ToUpperCase(js.SliceJS(stripped, 0, 2))
	p, known := lookupPattern(ibanPatterns, code)
	if wl := js.Get(options, "whitelist"); js.Truthy(wl) {
		if !wl.IsArray() {
			js.Get(wl, "filter")
			js.Throw("TypeError", "countryCodeArray.filter is not a function")
		}
		for _, c := range wl.Obj().Elems {
			k := js.ToPropertyKey(c)
			if _, ok := lookupPattern(ibanPatterns, k); !ok && !js.IsObjectProtoKey(k) {
				return false
			}
		}
		if !arrayIncludes(wl, js.Str(code), "arr.some") {
			return false
		}
	}
	if bl := js.Get(options, "blacklist"); js.Truthy(bl) && arrayIncludes(bl, js.Str(code), "arr.some") {
		return false
	}
	return known && p.re.Test(stripped)
}

func hasValidIBANChecksum(s string) bool {
	stripped := js.ToUpperCase(replace(s, reIBANStrip, ""))
	rearranged := js.SliceJS(stripped, 4, js.Len16(stripped)) + js.SliceJS(stripped, 0, 4)
	digits := replaceFunc(rearranged, reCapitalLetter, func(m match) string {
		return js.NumberToString(float64(js.CharCodeAt(m.text, 0) - 55))
	})
	runs := matchTexts(digits, reDigitRuns)
	if runs == nil {
		js.Throw("TypeError", "Cannot read properties of null (reading 'reduce')")
	}
	acc := js.Str("")
	for _, r := range runs {
		acc = js.Num(math.Mod(js.ToNumber(js.Add(acc, js.Str(r))), 97))
	}
	return js.StrictEquals(acc, js.Num(1))
}

func isIBAN(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	if options.IsUndefined() {
		options = js.NewObject()
	}
	return js.Bool(hasValidIBANFormat(s, options) && hasValidIBANChecksum(s))
}

var reEAN = re(`^(\d{8}|\d{13}|\d{14})$`, "")

func isEAN(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	u := js.ToU16(s)
	actual := js.StringToNumber(js.SliceJS(s, -1, len(u)))
	if !test(reEAN, s) {
		return js.Bool(false)
	}
	checksum := 0.0
	for i := 0; i < len(u)-1; i++ {
		weight := 1.0
		if (len(u) == 8 || len(u) == 14) == (i%2 == 0) {
			weight = 3
		}
		checksum += js.StringToNumber(js.FromU16(u[i:i+1])) * weight
	}
	remainder := 10 - math.Mod(checksum, 10)
	if remainder >= 10 {
		remainder = 0
	}
	return js.Bool(actual == remainder)
}

var reISIN = re(`^[A-Z]{2}[0-9A-Z]{9}[0-9]$`, "")

func isISIN(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	if !test(reISIN, s) {
		return js.Bool(false)
	}
	double := true
	sum := 0
	add := func(digit int) {
		if double {
			if digit >= 5 {
				sum += 1 + (digit-5)*2
			} else {
				sum += digit * 2
			}
		} else {
			sum += digit
		}
		double = !double
	}
	for i := len(s) - 2; i >= 0; i-- {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			v := int(c) - 55
			add(v % 10)
			add(v / 10)
		} else {
			add(int(c - '0'))
		}
	}
	check := (sum+9)/10*10 - sum
	return js.Bool(int(s[len(s)-1]-'0') == check)
}

var reISBN10 = re(`^(?:[0-9]{9}X|[0-9]{10})$`, "")
var reISBN13 = re(`^(?:[0-9]{13})$`, "")
var reISBNSeparators = re(`[\s-]+`, "g")

func isISBN(isbn string, options js.Value) bool {
	v := optional(options, "version")
	if !js.Truthy(v) {
		v = options
	}
	version := js.ToString(v)
	if !js.Truthy(v) {
		ten, thirteen := js.NewObject(), js.NewObject()
		ten.Obj().SetProp("version", js.Num(10))
		thirteen.Obj().SetProp("version", js.Num(13))
		return isISBN(isbn, ten) || isISBN(isbn, thirteen)
	}
	s := replace(isbn, reISBNSeparators, "")
	digit := func(i int) float64 { return js.StringToNumber(js.CharAt(s, i)) }
	checksum := 0.0
	switch version {
	case "10":
		if !test(reISBN10, s) {
			return false
		}
		for i := 0; i < 9; i++ {
			checksum += float64(i+1) * digit(i)
		}
		if js.CharAt(s, 9) == "X" {
			checksum += 100
		} else {
			checksum += 10 * digit(9)
		}
		return math.Mod(checksum, 11) == 0
	case "13":
		if !test(reISBN13, s) {
			return false
		}
		for i := 0; i < 12; i++ {
			checksum += float64(1+2*(i%2)) * digit(i)
		}
		return digit(12)-math.Mod(10-math.Mod(checksum, 10), 10) == 0
	}
	return false
}

func isISSN(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	if options.IsUndefined() {
		options = js.NewObject()
	}
	pattern := `^\d{4}-?\d{3}[\dX]$`
	if optTrue(options, "require_hyphen") {
		pattern = strings.Replace(pattern, "?", "", 1)
	}
	flags := "i"
	if optTrue(options, "case_sensitive") {
		flags = ""
	}
	if !jsre.MustCompile(pattern, flags).Test(s) {
		return js.Bool(false)
	}
	digits := js.ToU16(js.ToUpperCase(js.ReplaceFirst(s, "-", "")))
	checksum := 0.0
	for i, d := range digits {
		v := 10.0
		if d != 'X' {
			v = js.StringToNumber(js.FromU16([]uint16{d}))
		}
		checksum += v * float64(8-i)
	}
	return js.Bool(math.Mod(checksum, 11) == 0)
}

var reIMEI = re(`^[0-9]{15}$`, "")
var reIMEIHyphens = re(`^\d{2}-\d{6}-\d{6}-\d{1}$`, "")
var reHyphens = re(`-`, "g")

func isIMEI(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	if !js.Truthy(options) {
		options = js.NewObject()
	}
	r := reIMEI
	if optTrue(options, "allow_hyphens") {
		r = reIMEIHyphens
	}
	if !test(r, s) {
		return js.Bool(false)
	}
	s = replace(s, reHyphens, "")
	sum, mul := 0, 2
	for i := 0; i < 14; i++ {
		tp := int(s[14-i-1]-'0') * mul
		if tp >= 10 {
			sum += tp%10 + 1
		} else {
			sum += tp
		}
		if mul == 1 {
			mul++
		} else {
			mul--
		}
	}
	chk := (10 - sum%10) % 10
	return js.Bool(chk == int(s[14]-'0'))
}

var reAbaRouting = re(`^(?!(1[3-9])|(20)|(3[3-9])|(4[0-9])|(5[0-9])|(60)|(7[3-9])|(8[1-9])|(9[0-2])|(9[3-9]))[0-9]{9}$`, "")

func isAbaRouting(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	if !test(reAbaRouting, s) {
		return js.Bool(false)
	}
	sum := 0
	for i := 0; i < len(s); i++ {
		sum += int(s[i]-'0') * [3]int{3, 7, 1}[i%3]
	}
	return js.Bool(sum%10 == 0)
}

var reISO6346 = re(`^[A-Z]{3}(U[0-9]{7})|([J,Z][0-9]{6,7})$`, "")
var reDigit = re(`^[0-9]$`, "")

func isISO6346(a []js.Value) js.Value {
	s := js.ToUpperCase(assertString(arg(a, 0)))
	if !test(reISO6346, s) {
		return js.Bool(false)
	}
	u := js.ToU16(s)
	if len(u) != 11 {
		return js.Bool(true)
	}
	sum := 0.0
	for i := 0; i < len(u)-1; i++ {
		ch := js.FromU16(u[i : i+1])
		if !test(reDigit, ch) {
			code := float64(u[i]) - 55
			var converted float64
			switch {
			case code < 11:
				converted = code
			case code >= 11 && code <= 20:
				converted = 12 + math.Mod(code, 11)
			case code >= 21 && code <= 30:
				converted = 23 + math.Mod(code, 21)
			default:
				converted = 34 + math.Mod(code, 31)
			}
			sum += converted * math.Pow(2, float64(i))
		} else {
			sum += js.StringToNumber(ch) * math.Pow(2, float64(i))
		}
	}
	check := math.Mod(sum, 11)
	if check == 10 {
		check = 0
	}
	return js.Bool(js.StringToNumber(js.FromU16(u[len(u)-1:])) == check)
}
