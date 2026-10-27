package validator

import (
	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isHexColor", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		options := merge(arg(a, 1), []option{{"require_hashtag", js.Bool(false)}})
		if optTrue(options, "require_hashtag") {
			return js.Bool(test(reHexColorPrefixed, s))
		}
		return js.Bool(test(reHexColor, s))
	})
	register("isRgbColor", isRgbColor)
	register("isHSL", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		stripped := replace(replace(s, reSpaceRuns, " "), reHSLSeparators, "$1")
		if js.IndexOf16(stripped, ",", 0) != -1 {
			return js.Bool(test(reHSLComma, stripped))
		}
		return js.Bool(test(reHSLSpace, stripped))
	})
	registerPattern("isSlug", re(`^[a-z0-9](?!.*[-_]{2,})(?:[a-z0-9_-]*[a-z0-9])?$`, ""))
	register("isStrongPassword", isStrongPassword)
}

var reHexColor = re(`^#?([0-9A-F]{3}|[0-9A-F]{4}|[0-9A-F]{6}|[0-9A-F]{8})$`, "i")
var reHexColorPrefixed = re(`^#([0-9A-F]{3}|[0-9A-F]{4}|[0-9A-F]{6}|[0-9A-F]{8})$`, "i")

const rgbChannel = `([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])`
const rgbPercent = `([0-9]%|[1-9][0-9]%|100%)`
const rgbAlpha = `(0?\.\d+|1(\.0+)?|0(\.0+)?)`

var reRgb = re(`^rgb\((`+rgbChannel+`,){2}`+rgbChannel+`\)$`, "")
var reRgba = re(`^rgba\((`+rgbChannel+`,){3}`+rgbAlpha+`\)$`, "")
var reRgbPercent = re(`^rgb\((`+rgbPercent+`,){2}`+rgbPercent+`\)$`, "")
var reRgbaPercent = re(`^rgba\((`+rgbPercent+`,){3}`+rgbAlpha+`\)$`, "")
var reStartsWithRgb = re(`^rgba?`, "")
var reSpaces = re(`\s`, "g")

// isRgbColor reads its second argument as an options object, or, when it is not an object, as the legacy
// includePercentValues flag (only when it was passed at all).
func isRgbColor(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := arg(a, 1)
	allowSpaces := js.Bool(false)
	includePercent := js.Bool(true)
	if js.TypeOf(options) != "object" {
		if len(a) >= 2 {
			includePercent = arg(a, 1)
		}
	} else {
		if v := js.Get(options, "allowSpaces"); !v.IsUndefined() {
			allowSpaces = v
		}
		if v := js.Get(options, "includePercentValues"); !v.IsUndefined() {
			includePercent = v
		}
	}
	if js.Truthy(allowSpaces) {
		if !test(reStartsWithRgb, s) {
			return js.Bool(false)
		}
		s = replace(s, reSpaces, "")
	}
	if !js.Truthy(includePercent) {
		return js.Bool(test(reRgb, s) || test(reRgba, s))
	}
	return js.Bool(test(reRgb, s) || test(reRgba, s) || test(reRgbPercent, s) || test(reRgbaPercent, s))
}

var reSpaceRuns = re(`\s+`, "g")
var reHSLSeparators = re(`\s?(hsla?\(|\)|,)\s?`, "gi")
var reHSLComma = re(`^hsla?\(((\+|\-)?([0-9]+(\.[0-9]+)?(e(\+|\-)?[0-9]+)?|\.[0-9]+(e(\+|\-)?[0-9]+)?))(deg|grad|rad|turn)?(,(\+|\-)?([0-9]+(\.[0-9]+)?(e(\+|\-)?[0-9]+)?|\.[0-9]+(e(\+|\-)?[0-9]+)?)%){2}(,((\+|\-)?([0-9]+(\.[0-9]+)?(e(\+|\-)?[0-9]+)?|\.[0-9]+(e(\+|\-)?[0-9]+)?)%?))?\)$`, "i")
var reHSLSpace = re(`^hsla?\(((\+|\-)?([0-9]+(\.[0-9]+)?(e(\+|\-)?[0-9]+)?|\.[0-9]+(e(\+|\-)?[0-9]+)?))(deg|grad|rad|turn)?(\s(\+|\-)?([0-9]+(\.[0-9]+)?(e(\+|\-)?[0-9]+)?|\.[0-9]+(e(\+|\-)?[0-9]+)?)%){2}\s?(\/\s((\+|\-)?([0-9]+(\.[0-9]+)?(e(\+|\-)?[0-9]+)?|\.[0-9]+(e(\+|\-)?[0-9]+)?)%?)\s?)?\)$`, "i")

var reUpperChar = re(`^[A-Z]$`, "")
var reLowerChar = re(`^[a-z]$`, "")
var reNumberChar = re(`^[0-9]$`, "")
var reSymbolChar = re(`^[-#!$@£%^&*()_+|~=`+"`"+`{}\[\]:";'<>?,.\/\\ ]$`, "")

type passwordAnalysis struct {
	length, unique, upper, lower, number, symbol float64
}

// analyzePassword counts characters by code point, as Array.from does, and classifies each distinct one.
func analyzePassword(s string) passwordAnalysis {
	counts := map[string]float64{}
	var order []string
	for _, r := range js.CodePoints(s) {
		ch := js.FromCodePoints([]rune{r})
		if counts[ch] == 0 {
			order = append(order, ch)
		}
		counts[ch]++
	}
	an := passwordAnalysis{length: float64(js.Len16(s)), unique: float64(len(order))}
	for _, ch := range order {
		switch {
		case test(reUpperChar, ch):
			an.upper += counts[ch]
		case test(reLowerChar, ch):
			an.lower += counts[ch]
		case test(reNumberChar, ch):
			an.number += counts[ch]
		case test(reSymbolChar, ch):
			an.symbol += counts[ch]
		}
	}
	return an
}

func passwordDefaults() []option {
	return []option{{"minLength", js.Num(8)}, {"minLowercase", js.Num(1)}, {"minUppercase", js.Num(1)},
		{"minNumbers", js.Num(1)}, {"minSymbols", js.Num(1)}, {"returnScore", js.Bool(false)},
		{"pointsPerUnique", js.Num(1)}, {"pointsPerRepeat", js.Num(0.5)}, {"pointsForContainingLower", js.Num(10)},
		{"pointsForContainingUpper", js.Num(10)}, {"pointsForContainingNumber", js.Num(10)},
		{"pointsForContainingSymbol", js.Num(10)}}
}

func isStrongPassword(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	an := analyzePassword(s)
	options := arg(a, 1)
	if !js.Truthy(options) {
		options = js.NewObject()
	}
	options = merge(options, passwordDefaults())
	if optTrue(options, "returnScore") {
		points := js.Num(0)
		points = js.Add(points, js.Num(an.unique*js.ToNumber(opt(options, "pointsPerUnique"))))
		points = js.Add(points, js.Num((an.length-an.unique)*js.ToNumber(opt(options, "pointsPerRepeat"))))
		if an.lower > 0 {
			points = js.Add(points, opt(options, "pointsForContainingLower"))
		}
		if an.upper > 0 {
			points = js.Add(points, opt(options, "pointsForContainingUpper"))
		}
		if an.number > 0 {
			points = js.Add(points, opt(options, "pointsForContainingNumber"))
		}
		if an.symbol > 0 {
			points = js.Add(points, opt(options, "pointsForContainingSymbol"))
		}
		return points
	}
	return js.Bool(js.GreaterEq(js.Num(an.length), opt(options, "minLength")) &&
		js.GreaterEq(js.Num(an.lower), opt(options, "minLowercase")) &&
		js.GreaterEq(js.Num(an.upper), opt(options, "minUppercase")) &&
		js.GreaterEq(js.Num(an.number), opt(options, "minNumbers")) &&
		js.GreaterEq(js.Num(an.symbol), opt(options, "minSymbols")))
}
