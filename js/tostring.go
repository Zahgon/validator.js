package js

import (
	"math"
	"strconv"
	"strings"

	"github.com/validatorjs/validator.js/internal/jsdate"
)

// NumberToString formats n the way Number.prototype.toString() does with radix 10.
func NumberToString(n float64) string {
	switch {
	case math.IsNaN(n):
		return "NaN"
	case n == 0:
		return "0"
	case math.IsInf(n, 1):
		return "Infinity"
	case math.IsInf(n, -1):
		return "-Infinity"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	e := strconv.FormatFloat(n, 'e', -1, 64)
	mant, expPart, _ := strings.Cut(e, "e")
	digits := strings.Replace(mant, ".", "", 1)
	exp, _ := strconv.Atoi(expPart)
	k := len(digits)
	p := exp + 1
	var out string
	switch {
	case k <= p && p <= 21:
		out = digits + strings.Repeat("0", p-k)
	case 0 < p && p <= 21:
		out = digits[:p] + "." + digits[p:]
	case -6 < p && p <= 0:
		out = "0." + strings.Repeat("0", -p) + digits
	default:
		es := "+"
		if p-1 < 0 {
			es = "-"
		}
		ev := p - 1
		if ev < 0 {
			ev = -ev
		}
		if k == 1 {
			out = digits + "e" + es + strconv.Itoa(ev)
		} else {
			out = digits[:1] + "." + digits[1:] + "e" + es + strconv.Itoa(ev)
		}
	}
	return sign + out
}

// NumberToStringRadix formats n like Number.prototype.toString(radix) for an integral radix 2 to 36.
func NumberToStringRadix(n float64, radix int) string {
	if radix == 10 || math.IsNaN(n) || math.IsInf(n, 0) {
		return NumberToString(n)
	}
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	ip := math.Floor(n)
	frac := n - ip
	const digitChars = "0123456789abcdefghijklmnopqrstuvwxyz"
	var ib []byte
	if ip == 0 {
		ib = []byte{'0'}
	}
	for ip > 0 {
		d := math.Mod(ip, float64(radix))
		ib = append([]byte{digitChars[int(d)]}, ib...)
		ip = math.Floor(ip / float64(radix))
	}
	s := string(ib)
	if frac > 0 {
		var fb []byte
		for i := 0; i < 52 && frac > 0; i++ {
			frac *= float64(radix)
			d := math.Floor(frac)
			fb = append(fb, digitChars[int(d)])
			frac -= d
		}
		s += "." + string(fb)
	}
	if neg {
		s = "-" + s
	}
	return s
}

// ToString converts v with the abstract ToString operation.
func ToString(v Value) string {
	switch v.kind {
	case KindUndefined:
		return "undefined"
	case KindNull:
		return "null"
	case KindBoolean:
		if v.b {
			return "true"
		}
		return "false"
	case KindNumber:
		return NumberToString(v.n)
	case KindString:
		return v.s
	}
	return ToString(ToPrimitive(v, "string"))
}

// ToPrimitive converts an object to a primitive with the given hint ("string", "number" or "default").
func ToPrimitive(v Value, hint string) Value {
	if v.kind != KindObject {
		return v
	}
	o := v.o
	switch o.Class {
	case ClassArray:
		return Str(ArrayJoin(v, ","))
	case ClassRegExp:
		return Str("/" + o.Source + "/" + o.Flags)
	case ClassDate:
		if hint == "number" {
			return Num(o.Time)
		}
		return Str(jsdate.ToDateString(o.Time))
	case ClassFunction:
		return Str("function " + o.FuncName + "() { [native code] }")
	}
	if f, ok := o.OwnProp("toString"); ok && !f.IsFunction() {
		Throw("TypeError", "Cannot convert object to primitive value")
	}
	return Str("[object Object]")
}

// ArrayJoin implements Array.prototype.join.
func ArrayJoin(v Value, sep string) string {
	var b strings.Builder
	for i, e := range v.o.Elems {
		if i > 0 {
			b.WriteString(sep)
		}
		if !e.IsNullish() {
			b.WriteString(ToString(e))
		}
	}
	return b.String()
}

// ToPropertyKey converts v to the string used as a property key.
func ToPropertyKey(v Value) string {
	return ToString(v)
}

// ObjectToStringTag returns Object.prototype.toString.call(v).
func ObjectToStringTag(v Value) string {
	switch v.kind {
	case KindUndefined:
		return "[object Undefined]"
	case KindNull:
		return "[object Null]"
	case KindBoolean:
		return "[object Boolean]"
	case KindNumber:
		return "[object Number]"
	case KindString:
		return "[object String]"
	}
	switch v.o.Class {
	case ClassArray:
		return "[object Array]"
	case ClassRegExp:
		return "[object RegExp]"
	case ClassDate:
		return "[object Date]"
	case ClassFunction:
		return "[object Function]"
	}
	return "[object Object]"
}

// ConstructorName returns input.constructor.name for a non-nullish value, as assertString reads it.
func ConstructorName(v Value) (Value, bool) {
	switch v.kind {
	case KindBoolean:
		return Str("Boolean"), true
	case KindNumber:
		return Str("Number"), true
	case KindString:
		return Str("String"), true
	}
	o := v.o
	if c, ok := o.OwnProp("constructor"); ok {
		return GetProp(c, Str("name")), true
	}
	switch o.Class {
	case ClassArray:
		return Str("Array"), true
	case ClassRegExp:
		return Str("RegExp"), true
	case ClassDate:
		return Str("Date"), true
	case ClassFunction:
		return Str("Function"), true
	}
	return Str("Object"), true
}

// DateISOString formats a valid time value like Date.prototype.toISOString.
func DateISOString(t float64) string { return jsdate.ToISOString(t) }

// TypeTag returns the lowercase type tag of Object.prototype.toString, as the library's typeOf helper does
// ("array", "null", "date", "string", "undefined", "number", ...).
func TypeTag(v Value) string {
	tag := ObjectToStringTag(v)
	return strings.ToLower(tag[len("[object ") : len(tag)-1])
}
