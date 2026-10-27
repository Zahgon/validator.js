package validator

import (
	"math"

	"github.com/validatorjs/validator.js/internal/jsre"
	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isLicensePlate", isLicensePlate)
	register("isVAT", isVAT)
	register("isPassportNumber", isPassportNumber)
	registerValue("passportNumberLocales", func() js.Value { return localeKeys(passportPatterns) })
}

// regexEntry is one member of a table of `str => /re/.test(str)` validators: a pattern tested on the input (or
// on its trimmed form), or a function for the members that compute more than one pattern.
type regexEntry struct {
	key  string
	re   *jsre.Regexp
	trim bool
	fn   func(str string) bool
}

func (e regexEntry) check(s string) bool {
	if e.fn != nil {
		return e.fn(s)
	}
	if e.trim {
		s = js.Trim(s)
	}
	return e.re.Test(s)
}

func lookupEntry(table []regexEntry, key string) (regexEntry, bool) {
	for _, e := range table {
		if e.key == key {
			return e, true
		}
	}
	return regexEntry{}, false
}

// callEntry implements `table[key](str)` after `key in table` was found true.
func callEntry(table []regexEntry, key, s string) (js.Value, bool) {
	if e, ok := lookupEntry(table, key); ok {
		return js.Bool(e.check(s)), true
	}
	if js.IsObjectProtoKey(key) {
		return callProtoMethod(key, s, func(k string) bool { _, ok := lookupEntry(table, k); return ok }), true
	}
	return js.Undefined, false
}

func isLicensePlate(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	locale := arg(a, 1)
	if v, ok := callEntry(licensePlateValidators, js.ToPropertyKey(locale), s); ok {
		return v
	}
	if locale.IsString() && locale.String() == "any" {
		for _, e := range licensePlateValidators {
			if e.check(s) {
				return js.Bool(true)
			}
		}
		return js.Bool(false)
	}
	js.Throw("Error", "Invalid locale '"+js.ToString(locale)+"'")
	return js.Undefined
}

func isVAT(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	code := assertString(arg(a, 1))
	if v, ok := callEntry(vatMatchers, code, s); ok {
		return v
	}
	js.Throw("Error", "Invalid country code: '"+code+"'")
	return js.Undefined
}

var reVATAU = re(`^(AU)?(\d{11})$`, "")
var reLeadingAU = re(`^AU`, "")

func vatAU(s string) bool {
	if !test(reVATAU, s) {
		return false
	}
	weights := []float64{10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	s = replace(s, reLeadingAU, "")
	abn := js.NumberToString(js.ParseInt(js.Str(js.SliceJS(s, 0, 1)), js.Num(10))-1) + js.SliceJS(s, 1, js.Len16(s))
	total := 0.0
	for i := 0; i < 11; i++ {
		total += weights[i] * js.StringToNumber(js.CharAt(abn, i))
	}
	return total != 0 && math.Mod(total, 89) == 0
}

var reVATCH = re(`^(CHE[- ]?)?(\d{9}|(\d{3}\.\d{3}\.\d{3})|(\d{3} \d{3} \d{3})) ?(TVA|MWST|IVA)?$`, "")
var reSingleDigit = re(`\d`, "g")

func vatCH(s string) bool {
	if !test(reVATCH, s) {
		return false
	}
	digits := matchTexts(s, reSingleDigit)
	last := js.StringToNumber(digits[len(digits)-1])
	digits = digits[:len(digits)-1]
	weights := []float64{5, 4, 3, 2, 7, 6, 5, 4}
	sum := 0.0
	for i, d := range digits {
		w := math.NaN()
		if i < len(weights) {
			w = weights[i]
		}
		sum += js.StringToNumber(d) * w
	}
	return last == math.Mod(11-math.Mod(sum, 11), 11)
}

var reVATPT = re(`^(PT)?(\d{9})$`, "")

func vatPT(s string) bool {
	m, ok := firstMatch(s, reVATPT)
	if !ok {
		return false
	}
	tin := *m.caps[1]
	digits := make([]float64, 8)
	for i := range digits {
		digits[i] = float64(tin[i] - '0')
	}
	checksum := 11 - math.Mod(reverseMultiplyAndSum(digits, 9), 11)
	last := float64(tin[8] - '0')
	if checksum > 9 {
		return last == 0
	}
	return checksum == last
}

var reWhitespaceAll = re(`\s`, "g")

func isPassportNumber(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	countryCode := arg(a, 1)
	normalized := js.ToUpperCase(replace(s, reWhitespaceAll, ""))
	upper := js.ToUpperCase(stringMethodTarget(countryCode, "toUpperCase"))
	if _, ok := lookupPattern(passportPatterns, upper); !ok {
		return js.Bool(false)
	}
	p, ok := lookupPattern(passportPatterns, countryCode.String())
	if !ok {
		js.Throw("TypeError", "Cannot read properties of undefined (reading 'test')")
	}
	return js.Bool(p.re.Test(normalized))
}
