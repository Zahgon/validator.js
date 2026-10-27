package validator

import (
	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isLength", isLength)
	register("isByteLength", isByteLength)
	registerPattern("isAscii", reASCII)
	registerPattern("isFullWidth", reFullWidth)
	registerPattern("isHalfWidth", reHalfWidth)
	register("isVariableWidth", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Bool(test(reFullWidth, s) && test(reHalfWidth, s))
	})
	registerPattern("isMultibyte", re(`[^\x00-\x7F]`, ""))
	registerPattern("isSurrogatePair", re(`[\uD800-\uDBFF][\uDC00-\uDFFF]`, ""))
	register("isLowercase", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Bool(s == js.ToLowerCase(s))
	})
	register("isUppercase", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Bool(s == js.ToUpperCase(s))
	})
	register("isEmpty", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		options := merge(arg(a, 1), []option{{"ignore_whitespace", js.Bool(false)}})
		if optTrue(options, "ignore_whitespace") {
			s = js.Trim(s)
		}
		return js.Bool(js.Len16(s) == 0)
	})
	register("isIn", isIn)
	register("isBoolean", isBoolean)
	register("isISO31661Alpha2", func(a []js.Value) js.Value {
		return js.Bool(isCountryCode(a, reAlpha2Code, iso31661Alpha2Codes))
	})
	register("isISO31661Alpha3", func(a []js.Value) js.Value {
		return js.Bool(isCountryCode(a, re(`^[a-zA-Z]{3}$`, ""), iso31661Alpha3Codes))
	})
	register("isISO31661Numeric", func(a []js.Value) js.Value {
		return js.Bool(iso31661NumericCodes[assertString(arg(a, 0))])
	})
	register("isISO6391", func(a []js.Value) js.Value { return js.Bool(iso6391Codes[assertString(arg(a, 0))]) })
	register("isISO15924", func(a []js.Value) js.Value { return js.Bool(iso15924Codes[assertString(arg(a, 0))]) })
	register("isISO4217", func(a []js.Value) js.Value {
		return js.Bool(iso4217Codes[js.ToUpperCase(assertString(arg(a, 0)))])
	})
}

var reASCII = re(`^[\x00-\x7F]+$`, "")
var reFullWidth = re(`[^\u0020-\u007E\uFF61-\uFF9F\uFFA0-\uFFDC\uFFE8-\uFFEE0-9a-zA-Z]`, "")
var reHalfWidth = re(`[\u0020-\u007E\uFF61-\uFF9F\uFFA0-\uFFDC\uFFE8-\uFFEE0-9a-zA-Z]`, "")
var reAlpha2Code = re(`^[a-zA-Z]{2}$`, "")

func stringSet(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, s := range items {
		m[s] = true
	}
	return m
}

// registerPattern registers a validator that asserts a string and tests one pattern.
func registerPattern(name string, r interface{ Test(string) bool }) {
	register(name, func(a []js.Value) js.Value { return js.Bool(r.Test(assertString(arg(a, 0)))) })
}

var rePresentation = re(`[^\uFE0F\uFE0E][\uFE0F\uFE0E]`, "g")
var reSurrogatePairs = re(`[\uD800-\uDBFF][\uDC00-\uDFFF]`, "g")

func isLength(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	var min, max js.Value
	if js.TypeOf(options) == "object" {
		min = js.Get(options, "min")
		max = js.Get(options, "max")
	} else {
		min = arg(a, 1)
		max = arg(a, 2)
	}
	if !js.Truthy(min) {
		min = js.Num(0)
	}
	n := js.Len16(s) - len(matchTexts(s, rePresentation)) - len(matchTexts(s, reSurrogatePairs))
	ln := js.Num(float64(n))
	inside := js.GreaterEq(ln, min) && (max.IsUndefined() || js.LessEq(ln, max))
	if inside && !options.IsNullish() {
		if d := js.Get(options, "discreteLengths"); d.IsArray() {
			for _, e := range d.Obj().Elems {
				if js.StrictEquals(e, ln) {
					return js.Bool(true)
				}
			}
			return js.Bool(false)
		}
	}
	return js.Bool(inside)
}

func utf8ByteLength(s string) int {
	u := js.ToU16(s)
	n := 0
	for i := 0; i < len(u); i++ {
		c := u[i]
		switch {
		case c <= 0x7F:
			n++
		case c <= 0x7FF:
			n += 2
		case c >= 0xD800 && c <= 0xDBFF && i+1 < len(u) && u[i+1] >= 0xDC00 && u[i+1] <= 0xDFFF:
			n += 4
			i++
		default:
			n += 3
		}
	}
	return n
}

func isByteLength(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	var min, max js.Value
	if js.TypeOf(options) == "object" {
		min = js.Get(options, "min")
		if !js.Truthy(min) {
			min = js.Num(0)
		}
		max = js.Get(options, "max")
	} else {
		min = arg(a, 1)
		max = arg(a, 2)
	}
	ln := js.Num(float64(utf8ByteLength(s)))
	return js.Bool(js.GreaterEq(ln, min) && (max.IsUndefined() || js.LessEq(ln, max)))
}

func isIn(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	switch {
	case options.IsArray():
		items := make([]string, len(options.Obj().Elems))
		for i, e := range options.Obj().Elems {
			items[i] = toStringValue(e)
		}
		for _, item := range items {
			if item == s {
				return js.Bool(true)
			}
		}
		return js.Bool(false)
	case js.TypeOf(options) == "object":
		if f := js.Get(options, "hasOwnProperty"); !f.IsFunction() {
			js.Throw("TypeError", "options.hasOwnProperty is not a function")
		}
		return js.Bool(js.HasOwnProperty(options, s))
	case js.Truthy(options) && js.Get(options, "indexOf").IsFunction():
		return js.Bool(js.IndexOf16(options.String(), s, 0) >= 0)
	}
	return js.Bool(false)
}

func isBoolean(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	if options.IsUndefined() {
		options = js.NewObject()
	}
	if optTrue(options, "loose") {
		switch js.ToLowerCase(s) {
		case "true", "false", "1", "0", "yes", "no":
			return js.Bool(true)
		}
		return js.Bool(false)
	}
	return js.Bool(s == "true" || s == "false" || s == "1" || s == "0")
}

// isCountryCode implements the shared logic of isISO31661Alpha2 and isISO31661Alpha3.
func isCountryCode(a []js.Value, shape interface{ Test(string) bool }, codes map[string]bool) bool {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	if options.IsUndefined() {
		options = js.NewObject()
	}
	if options.IsNull() {
		js.Throw("TypeError", "Cannot destructure property 'userAssignedCodes' of 'options' as it is null.")
	}
	userCodes := js.Get(options, "userAssignedCodes")
	upper := js.ToUpperCase(s)
	if js.Truthy(userCodes) {
		if !userCodes.IsArray() {
			js.Throw("TypeError", "(userAssignedCodes || []).reduce is not a function")
		}
		var valid []string
		for _, c := range userCodes.Obj().Elems {
			if shape.Test(js.ToString(c)) {
				if !c.IsString() {
					js.Get(c, "toUpperCase")
					js.Throw("TypeError", "userAssignedCode.toUpperCase is not a function")
				}
				valid = append(valid, js.ToUpperCase(c.String()))
			}
		}
		for _, v := range valid {
			if v == upper {
				return true
			}
		}
	}
	return codes[upper]
}
