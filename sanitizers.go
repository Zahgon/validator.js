package validator

import (
	"math"
	"strings"

	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("toDate", func(a []js.Value) js.Value { return toDate(arg(a, 0)) })
	register("toInt", toInt)
	register("toBoolean", toBoolean)
	register("equals", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Bool(js.StrictEquals(js.Str(s), arg(a, 1)))
	})
	register("contains", contains)
	register("matches", matches)
	register("ltrim", func(a []js.Value) js.Value { return js.Str(ltrim(arg(a, 0), arg(a, 1))) })
	register("rtrim", func(a []js.Value) js.Value { return js.Str(rtrim(arg(a, 0), arg(a, 1))) })
	register("trim", func(a []js.Value) js.Value { return js.Str(rtrim(js.Str(ltrim(arg(a, 0), arg(a, 1))), arg(a, 1))) })
	register("escape", escapeHTML)
	register("unescape", unescapeHTML)
	register("stripLow", stripLow)
	register("whitelist", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Str(replaceRegExp(s, newRegExp(js.Str("[^"+js.ToString(arg(a, 1))+"]+"), js.Str("g")), ""))
	})
	register("blacklist", func(a []js.Value) js.Value {
		return js.Str(blacklist(assertString(arg(a, 0)), js.ToString(arg(a, 1))))
	})
	register("isWhitelisted", isWhitelisted)
	register("toString", func(a []js.Value) js.Value { return js.Str(toStringValue(arg(a, 0))) })
}

// toDate converts a date string with Date.parse and returns a Date, or null when it does not parse.
func toDate(date js.Value) js.Value {
	s := assertString(date)
	t := dateParse(s)
	if math.IsNaN(t) {
		return js.Null
	}
	return js.NewDate(t)
}

func toInt(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	radix := arg(a, 1)
	if !js.Truthy(radix) {
		radix = js.Num(10)
	}
	return js.Num(js.ParseInt(js.Str(s), radix))
}

var reTrue = re(`^true$`, "i")
var reFalse = re(`^false$`, "i")

func toBoolean(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	if js.Truthy(arg(a, 1)) {
		return js.Bool(s == "1" || test(reTrue, s))
	}
	return js.Bool(s != "0" && !test(reFalse, s) && s != "")
}

func contains(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := merge(arg(a, 2), []option{{"ignoreCase", js.Bool(false)}, {"minOccurrences", js.Num(1)}})
	elem := toStringValue(arg(a, 1))
	var parts []string
	if optTrue(options, "ignoreCase") {
		parts = js.Split16(js.ToLowerCase(s), js.ToLowerCase(elem))
	} else {
		parts = js.Split16(s, elem)
	}
	return js.Bool(js.Greater(js.Num(float64(len(parts))), opt(options, "minOccurrences")))
}

func matches(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	pattern := arg(a, 1)
	if !pattern.IsRegExp() {
		pattern = newRegExp(pattern, arg(a, 2))
	}
	return js.Bool(stringMatch(s, pattern))
}

// stringMatch reports whether String.prototype.match finds anything, running the RegExp object's exec protocol.
func stringMatch(s string, rx js.Value) bool {
	if compiledOf(rx.Obj()).Global() {
		rx.Obj().LastIndex = js.Num(0)
	}
	return regexpExec(rx, s) != nil
}

// replaceRegExp implements String.prototype.replace(rx, repl) for a RegExp object.
func replaceRegExp(s string, rx js.Value, repl string) string {
	return replace(s, compiledOf(rx.Obj()), repl)
}

var reTrimEscape = re(`[.*+?^${}()|[\]\\]`, "g")
var reLeadingSpace = re(`^\s+`, "g")
var reSpace = re(`\s`, "")

// trimChars returns chars.replace(...) for ltrim and rtrim, failing like a method call on a non-string.
func trimChars(chars js.Value) string {
	if !chars.IsString() {
		js.Get(chars, "replace")
		js.Throw("TypeError", "chars.replace is not a function")
	}
	return replace(chars.String(), reTrimEscape, `\$&`)
}

func ltrim(str js.Value, chars js.Value) string {
	s := assertString(str)
	if js.Truthy(chars) {
		rx := newRegExp(js.Str("^["+trimChars(chars)+"]+"), js.Str("g"))
		return replaceRegExp(s, rx, "")
	}
	return replace(s, reLeadingSpace, "")
}

func rtrim(str js.Value, chars js.Value) string {
	s := assertString(str)
	if js.Truthy(chars) {
		rx := newRegExp(js.Str("["+trimChars(chars)+"]+$"), js.Str("g"))
		return replaceRegExp(s, rx, "")
	}
	i := js.Len16(s) - 1
	for test(reSpace, js.CharAt(s, i)) {
		i--
	}
	return js.SliceJS(s, 0, i+1)
}

func escapeHTML(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	r := strings.NewReplacer("&", "&amp;", `"`, "&quot;", "'", "&#x27;", "<", "&lt;", ">", "&gt;", "/", "&#x2F;",
		`\`, "&#x5C;", "`", "&#96;")
	return js.Str(r.Replace(s))
}

func unescapeHTML(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	for _, p := range [][2]string{{"&quot;", `"`}, {"&#x27;", "'"}, {"&lt;", "<"}, {"&gt;", ">"}, {"&#x2F;", "/"},
		{"&#x5C;", `\`}, {"&#96;", "`"}, {"&amp;", "&"}} {
		s = strings.ReplaceAll(s, p[0], p[1])
	}
	return js.Str(s)
}

func blacklist(s string, chars string) string {
	return replaceRegExp(s, newRegExp(js.Str("["+chars+"]+"), js.Str("g")), "")
}

func stripLow(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	chars := `\x00-\x1F\x7F`
	if js.Truthy(arg(a, 1)) {
		chars = `\x00-\x09\x0B\x0C\x0E-\x1F\x7F`
	}
	return js.Str(blacklist(s, chars))
}

func isWhitelisted(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	chars := arg(a, 1)
	u := js.ToU16(s)
	for i := len(u) - 1; i >= 0; i-- {
		if indexOf(chars, js.Str(js.FromU16(u[i:i+1]))) == -1 {
			return js.Bool(false)
		}
	}
	return js.Bool(true)
}

// indexOf calls haystack.indexOf(needle) for a string or an array haystack, failing like JavaScript otherwise.
func indexOf(haystack js.Value, needle js.Value) int {
	switch {
	case haystack.IsString():
		return js.IndexOf16(haystack.String(), js.ToString(needle), 0)
	case haystack.IsArray():
		for i, e := range haystack.Obj().Elems {
			if js.StrictEquals(e, needle) {
				return i
			}
		}
		return -1
	}
	if f := js.Get(haystack, "indexOf"); !f.IsFunction() {
		js.Throw("TypeError", "chars.indexOf is not a function")
	}
	return -1
}
