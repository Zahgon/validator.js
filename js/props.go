package js

import "math"

// Error is a thrown JavaScript error: its constructor name and message.
type Error struct {
	Name    string
	Message string
}

func (e *Error) Error() string { return e.Name + ": " + e.Message }

// Throw raises a JavaScript error; Catch turns it back into an *Error.
func Throw(name, message string) {
	panic(&Error{Name: name, Message: message})
}

// Catch runs fn and returns the JavaScript error it threw, if any. Other panics propagate.
func Catch(fn func()) (err *Error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Error)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	fn()
	return nil
}

var objectProto = setOf("constructor", "__defineGetter__", "__defineSetter__", "hasOwnProperty",
	"__lookupGetter__", "__lookupSetter__", "isPrototypeOf", "propertyIsEnumerable", "toString", "valueOf",
	"__proto__", "toLocaleString")

var stringProto = setOf("length", "anchor", "at", "big", "blink", "bold", "charAt", "charCodeAt", "codePointAt",
	"concat", "endsWith", "fontcolor", "fontsize", "fixed", "includes", "indexOf", "isWellFormed", "italics",
	"lastIndexOf", "link", "localeCompare", "match", "matchAll", "normalize", "padEnd", "padStart", "repeat",
	"replace", "replaceAll", "search", "slice", "small", "split", "strike", "sub", "substr", "substring", "sup",
	"startsWith", "toString", "toWellFormed", "trim", "trimStart", "trimLeft", "trimEnd", "trimRight",
	"toLocaleLowerCase", "toLocaleUpperCase", "toLowerCase", "toUpperCase", "valueOf")

var numberProto = setOf("toExponential", "toFixed", "toPrecision", "toString", "toLocaleString", "valueOf")

var booleanProto = setOf("toString", "valueOf")

var arrayProto = setOf("length", "at", "concat", "copyWithin", "fill", "find", "findIndex", "findLast",
	"findLastIndex", "lastIndexOf", "pop", "push", "reverse", "shift", "unshift", "slice", "sort", "splice",
	"includes", "indexOf", "join", "keys", "entries", "values", "forEach", "filter", "flat", "flatMap", "map",
	"every", "some", "reduce", "reduceRight", "toLocaleString", "toString", "toReversed", "toSorted",
	"toSpliced", "with")

var regexpProto = setOf("exec", "dotAll", "flags", "global", "hasIndices", "ignoreCase", "multiline", "source",
	"sticky", "unicode", "unicodeSets", "lastIndex", "compile", "toString", "test")

var dateProto = setOf("toString", "toDateString", "toTimeString", "toISOString", "toUTCString", "toGMTString",
	"getDate", "setDate", "getDay", "getFullYear", "setFullYear", "getHours", "setHours", "getMilliseconds",
	"setMilliseconds", "getMinutes", "setMinutes", "getMonth", "setMonth", "getSeconds", "setSeconds", "getTime",
	"setTime", "getTimezoneOffset", "getUTCDate", "setUTCDate", "getUTCDay", "getUTCFullYear", "setUTCFullYear",
	"getUTCHours", "setUTCHours", "getUTCMilliseconds", "setUTCMilliseconds", "getUTCMinutes", "setUTCMinutes",
	"getUTCMonth", "setUTCMonth", "getUTCSeconds", "setUTCSeconds", "valueOf", "getYear", "setYear", "toJSON",
	"toLocaleString", "toLocaleDateString", "toLocaleTimeString")

func setOf(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// IsObjectProtoKey reports whether key names a member of Object.prototype, which every plain object inherits.
func IsObjectProtoKey(key string) bool { return objectProto[key] }

// HasProperty implements the in operator for a plain object, including inherited Object.prototype members.
func HasProperty(o *Object, key string) bool {
	if _, ok := o.OwnProp(key); ok {
		return true
	}
	return protoHas(o, key)
}

func protoHas(o *Object, key string) bool {
	switch o.Class {
	case ClassArray:
		if arrayProto[key] {
			return true
		}
	case ClassRegExp:
		if regexpProto[key] {
			return true
		}
	case ClassDate:
		if dateProto[key] {
			return true
		}
	case ClassFunction:
		if key == "name" || key == "length" || key == "call" || key == "apply" || key == "bind" {
			return true
		}
	}
	return objectProto[key]
}

// GetProp reads base[key] with the language's property lookup, throwing a TypeError on a nullish base.
func GetProp(base Value, key Value) Value {
	k := ToPropertyKey(key)
	switch base.kind {
	case KindUndefined, KindNull:
		Throw("TypeError", "Cannot read properties of "+ToString(base)+" (reading '"+k+"')")
	case KindString:
		if k == "length" {
			return Num(float64(Len16(base.s)))
		}
		if i, ok := canonicalIndex(k); ok {
			u := ToU16(base.s)
			if int(i) < len(u) {
				return Str(FromU16(u[i : i+1]))
			}
			return Undefined
		}
		if stringProto[k] || objectProto[k] {
			return NewFunction(k)
		}
		return Undefined
	case KindNumber:
		if numberProto[k] || objectProto[k] {
			return NewFunction(k)
		}
		return Undefined
	case KindBoolean:
		if booleanProto[k] || objectProto[k] {
			return NewFunction(k)
		}
		return Undefined
	}
	o := base.o
	if v, ok := o.OwnProp(k); ok {
		return v
	}
	switch o.Class {
	case ClassArray:
		if k == "length" {
			return Num(float64(len(o.Elems)))
		}
	case ClassRegExp:
		switch k {
		case "lastIndex":
			return o.LastIndex
		case "source":
			return Str(o.Source)
		case "flags":
			return Str(o.Flags)
		}
	case ClassFunction:
		if k == "name" {
			return Str(o.FuncName)
		}
		if k == "length" {
			return Num(0)
		}
	}
	if protoHas(o, k) {
		return NewFunction(k)
	}
	return Undefined
}

// Get reads base[key] for a string key.
func Get(base Value, key string) Value { return GetProp(base, Str(key)) }

// Len16 returns the length of s in UTF-16 code units.
func Len16(s string) int {
	n := 0
	for _, u := range decodeWTF8(s) {
		if u > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// ToIntegerOrInfinity converts v the way index and radix arguments are converted.
func ToIntegerOrInfinity(v Value) float64 {
	n := ToNumber(v)
	if math.IsNaN(n) || n == 0 {
		return 0
	}
	if math.IsInf(n, 0) {
		return n
	}
	return math.Trunc(n)
}

// HasOwnProperty implements Object.prototype.hasOwnProperty.call(v, key).
func HasOwnProperty(v Value, key string) bool {
	if v.kind == KindString {
		if key == "length" {
			return true
		}
		i, ok := canonicalIndex(key)
		return ok && int(i) < Len16(v.s)
	}
	if v.kind != KindObject {
		return false
	}
	if _, ok := v.o.OwnProp(key); ok {
		return true
	}
	switch v.o.Class {
	case ClassArray:
		return key == "length"
	case ClassRegExp:
		return key == "lastIndex"
	case ClassFunction:
		return key == "name" || key == "length"
	}
	return false
}
