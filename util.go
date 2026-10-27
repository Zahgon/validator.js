package validator

import (
	"math"
	"strconv"

	"github.com/validatorjs/validator.js/internal/jsdate"
	"github.com/validatorjs/validator.js/js"
)

// dateParse implements Date.parse.
func dateParse(s string) float64 { return jsdate.Parse(js.ToU16(s)) }

// assertString throws the library's TypeError unless input is a string.
func assertString(input js.Value) string {
	if input.IsNullish() {
		js.Throw("TypeError", "Expected a string but received a "+js.ToString(input))
	}
	name, _ := js.ConstructorName(input)
	if !(name.IsString() && name.String() == "String") {
		js.Throw("TypeError", "Expected a string but received a "+js.ToString(name))
	}
	return input.String()
}

// toStringValue converts any value to the string the library's toString helper produces.
func toStringValue(input js.Value) string {
	if input.IsObject() {
		if f, ok := input.Obj().OwnProp("toString"); ok && !f.IsFunction() {
			return "[object Object]"
		}
		return js.ToString(input)
	}
	if input.IsNullish() {
		return ""
	}
	if js.IsNaNGlobal(input) && !js.Truthy(js.Get(input, "length")) {
		return ""
	}
	return js.ToString(input)
}

// option is one entry of a defaults object, in its declaration order.
type option struct {
	key   string
	value js.Value
}

// merge fills the keys of defaults that are undefined in obj, mutating and returning obj like the library's
// merge helper; a non-object obj is replaced by a new object.
func merge(obj js.Value, defaults []option) js.Value {
	if !obj.IsObject() {
		obj = js.NewObject()
	}
	for _, d := range defaults {
		if js.Get(obj, d.key).IsUndefined() {
			obj.Obj().SetProp(d.key, d.value)
		}
	}
	return obj
}

// opt reads an option property.
func opt(options js.Value, key string) js.Value { return js.Get(options, key) }

// optTrue reports whether an option is truthy.
func optTrue(options js.Value, key string) bool { return js.Truthy(js.Get(options, key)) }

// length returns value.length converted for a numeric loop bound.
func length(v js.Value) js.Value { return js.Get(v, "length") }

// index reads v[i].
func index(v js.Value, i int) js.Value { return js.GetProp(v, js.Num(float64(i))) }

// forEachIndex runs f for i = 0 while i < v.length, as a JavaScript for loop does.
func forEachIndex(v js.Value, f func(i int) bool) {
	for i := 0; js.Less(js.Num(float64(i)), length(v)); i++ {
		if f(i) {
			return
		}
	}
}

// includesArray reports whether the array holds a value strictly equal to val.
func includesArray(arr js.Value, val js.Value) bool {
	found := false
	forEachIndex(arr, func(i int) bool {
		if js.StrictEquals(val, index(arr, i)) {
			found = true
			return true
		}
		return false
	})
	return found
}

// checkHost reports whether host equals one of matches or is matched by one of its RegExp members.
func checkHost(host js.Value, matches js.Value) bool {
	found := false
	forEachIndex(matches, func(i int) bool {
		m := index(matches, i)
		if js.StrictEquals(host, m) || (m.IsRegExp() && regexpTest(m, host)) {
			found = true
			return true
		}
		return false
	})
	return found
}

// digitAt returns parseInt(s[i], 10) for a string of digits, NaN past the end or for a non-digit.
func digitAt(u []uint16, i int) float64 {
	if i < 0 || i >= len(u) || u[i] < '0' || u[i] > '9' {
		return math.NaN()
	}
	return float64(u[i] - '0')
}

// iso7064Check validates a digit string with ISO 7064 MOD 11, 10.
func iso7064Check(str string) bool {
	u := js.ToU16(str)
	check := 10.0
	for i := 0; i < len(u)-1; i++ {
		d := digitAt(u, i)
		if math.Mod(d+check, 10) == 0 {
			check = math.Mod(10*2, 11)
		} else {
			check = math.Mod(math.Mod(d+check, 10)*2, 11)
		}
	}
	if check == 1 {
		check = 0
	} else {
		check = 11 - check
	}
	return check == digitAt(u, 10)
}

// luhnCheck validates a digit string with the Luhn algorithm.
func luhnCheck(str string) bool {
	u := js.ToU16(str)
	sum := 0.0
	second := false
	for i := len(u) - 1; i >= 0; i-- {
		if second {
			product := digitAt(u, i) * 2
			if product > 9 {
				for _, c := range js.NumberToString(product) {
					sum += float64(c - '0')
				}
			} else {
				sum += product
			}
		} else {
			sum += digitAt(u, i)
		}
		second = !second
	}
	return math.Mod(sum, 10) == 0
}

// reverseMultiplyAndSum multiplies digits by base, base-1, ... and sums the products.
func reverseMultiplyAndSum(digits []float64, base float64) float64 {
	total := 0.0
	for i, d := range digits {
		total += d * (base - float64(i))
	}
	return total
}

var verhoeffD = [10][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, {1, 2, 3, 4, 0, 6, 7, 8, 9, 5}, {2, 3, 4, 0, 1, 7, 8, 9, 5, 6},
	{3, 4, 0, 1, 2, 8, 9, 5, 6, 7}, {4, 0, 1, 2, 3, 9, 5, 6, 7, 8}, {5, 9, 8, 7, 6, 0, 4, 3, 2, 1},
	{6, 5, 9, 8, 7, 1, 0, 4, 3, 2}, {7, 6, 5, 9, 8, 2, 1, 0, 4, 3}, {8, 7, 6, 5, 9, 3, 2, 1, 0, 4},
	{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
}

var verhoeffP = [8][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, {1, 5, 7, 6, 2, 8, 3, 0, 9, 4}, {5, 8, 0, 3, 7, 9, 6, 1, 4, 2},
	{8, 9, 1, 6, 0, 4, 3, 5, 2, 7}, {9, 4, 5, 3, 1, 2, 6, 8, 7, 0}, {4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
	{2, 7, 9, 3, 8, 0, 6, 4, 1, 5}, {7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
}

// verhoeffCheck validates a digit string with the Verhoeff algorithm; callers pass digits only.
func verhoeffCheck(str string) bool {
	u := js.ToU16(str)
	checksum := 0
	for i := 0; i < len(u); i++ {
		d := int(u[len(u)-1-i] - '0')
		checksum = verhoeffD[checksum][verhoeffP[i%8][d]]
	}
	return checksum == 0
}

// itoa formats an int the way JavaScript prints an integral number.
func itoa(n int) string { return strconv.Itoa(n) }
