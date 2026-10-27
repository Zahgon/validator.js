// Package js models the small part of the JavaScript runtime that validator.js depends on: values and their
// conversions, UTF-16 strings, Unicode case mapping, number formatting and parsing, JSON.parse and thrown errors.
package js

import (
	"math"
	"strconv"
)

// Kind is the JavaScript type of a Value.
type Kind uint8

// The JavaScript types a Value can hold. Arrays, regular expressions, dates and functions are objects.
const (
	KindUndefined Kind = iota
	KindNull
	KindBoolean
	KindNumber
	KindString
	KindObject
)

// Class distinguishes the built-in object kinds.
type Class uint8

// Object classes.
const (
	ClassObject Class = iota
	ClassArray
	ClassRegExp
	ClassDate
	ClassFunction
)

// Value is one JavaScript value. Strings hold UTF-8 text, with lone surrogates kept in their three-byte form.
type Value struct {
	kind Kind
	b    bool
	n    float64
	s    string
	o    *Object
}

// Object is a JavaScript object. Arrays keep their elements in Elems; other own properties live in Props with
// their insertion order in keys.
type Object struct {
	Class Class
	keys  []string
	props map[string]Value
	Elems []Value
	// RegExp state.
	Source    string
	Flags     string
	LastIndex Value
	// Date time value in milliseconds since the epoch, NaN for an invalid date.
	Time float64
	// Name of a built-in function.
	FuncName string
}

// Undefined is the undefined value.
var Undefined = Value{kind: KindUndefined}

// Null is the null value.
var Null = Value{kind: KindNull}

// Bool returns a boolean value.
func Bool(b bool) Value { return Value{kind: KindBoolean, b: b} }

// Num returns a number value.
func Num(n float64) Value { return Value{kind: KindNumber, n: n} }

// Str returns a string value.
func Str(s string) Value { return Value{kind: KindString, s: s} }

// NewObject returns an empty plain object.
func NewObject() Value {
	return Value{kind: KindObject, o: &Object{Class: ClassObject, props: map[string]Value{}}}
}

// NewArray returns an array holding elems.
func NewArray(elems ...Value) Value {
	return Value{kind: KindObject, o: &Object{Class: ClassArray, props: map[string]Value{}, Elems: elems}}
}

// NewRegExpObject returns a RegExp object for source and flags; the caller has validated both.
func NewRegExpObject(source, flags string) Value {
	return Value{kind: KindObject, o: &Object{Class: ClassRegExp, props: map[string]Value{}, Source: source,
		Flags: flags, LastIndex: Num(0)}}
}

// NewDate returns a Date object with time value t.
func NewDate(t float64) Value {
	return Value{kind: KindObject, o: &Object{Class: ClassDate, props: map[string]Value{}, Time: t}}
}

// NewFunction returns a built-in function value named name.
func NewFunction(name string) Value {
	return Value{kind: KindObject, o: &Object{Class: ClassFunction, props: map[string]Value{}, FuncName: name}}
}

// Kind reports the value's type.
func (v Value) Kind() Kind { return v.kind }

// Obj returns the object a value refers to, or nil.
func (v Value) Obj() *Object { return v.o }

// IsUndefined reports whether v is undefined.
func (v Value) IsUndefined() bool { return v.kind == KindUndefined }

// IsNull reports whether v is null.
func (v Value) IsNull() bool { return v.kind == KindNull }

// IsNullish reports whether v is null or undefined.
func (v Value) IsNullish() bool { return v.kind == KindUndefined || v.kind == KindNull }

// IsString reports whether v is a string.
func (v Value) IsString() bool { return v.kind == KindString }

// IsNumber reports whether v is a number.
func (v Value) IsNumber() bool { return v.kind == KindNumber }

// IsBoolean reports whether v is a boolean.
func (v Value) IsBoolean() bool { return v.kind == KindBoolean }

// IsObject reports whether v is an object of any class.
func (v Value) IsObject() bool { return v.kind == KindObject }

// IsArray reports whether v is an array.
func (v Value) IsArray() bool { return v.kind == KindObject && v.o.Class == ClassArray }

// IsRegExp reports whether v is a RegExp object.
func (v Value) IsRegExp() bool { return v.kind == KindObject && v.o.Class == ClassRegExp }

// IsDate reports whether v is a Date object.
func (v Value) IsDate() bool { return v.kind == KindObject && v.o.Class == ClassDate }

// IsFunction reports whether v is a function.
func (v Value) IsFunction() bool { return v.kind == KindObject && v.o.Class == ClassFunction }

// Boolean returns the payload of a boolean value.
func (v Value) Boolean() bool { return v.b }

// Number returns the payload of a number value.
func (v Value) Number() float64 { return v.n }

// String returns the payload of a string value; use ToString for conversion.
func (v Value) String() string { return v.s }

// TypeOf returns the typeof operator's result.
func TypeOf(v Value) string {
	switch v.kind {
	case KindUndefined:
		return "undefined"
	case KindBoolean:
		return "boolean"
	case KindNumber:
		return "number"
	case KindString:
		return "string"
	case KindObject:
		if v.o.Class == ClassFunction {
			return "function"
		}
	}
	return "object"
}

// Truthy converts v with ToBoolean.
func Truthy(v Value) bool {
	switch v.kind {
	case KindBoolean:
		return v.b
	case KindNumber:
		return v.n != 0 && !math.IsNaN(v.n)
	case KindString:
		return v.s != ""
	case KindObject:
		return true
	}
	return false
}

// StrictEquals implements the === operator.
func StrictEquals(a, b Value) bool {
	if a.kind != b.kind {
		return false
	}
	switch a.kind {
	case KindUndefined, KindNull:
		return true
	case KindBoolean:
		return a.b == b.b
	case KindNumber:
		return a.n == b.n
	case KindString:
		return a.s == b.s
	}
	return a.o == b.o
}

// LooseEquals implements the == operator for the value kinds a validator meets.
func LooseEquals(a, b Value) bool {
	if a.kind == b.kind {
		return StrictEquals(a, b)
	}
	if a.IsNullish() && b.IsNullish() {
		return true
	}
	if a.IsNullish() || b.IsNullish() {
		return false
	}
	if a.kind == KindObject && b.kind != KindObject {
		return LooseEquals(ToPrimitive(a, "default"), b)
	}
	if b.kind == KindObject && a.kind != KindObject {
		return LooseEquals(a, ToPrimitive(b, "default"))
	}
	return ToNumber(a) == ToNumber(b)
}

// SetProp sets an own property, keeping insertion order for new keys.
func (o *Object) SetProp(key string, v Value) {
	if o.Class == ClassArray {
		if i, ok := arrayIndex(key); ok {
			for len(o.Elems) <= i {
				o.Elems = append(o.Elems, Undefined)
			}
			o.Elems[i] = v
			return
		}
	}
	if _, ok := o.props[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.props[key] = v
}

// OwnProp returns an own property and whether it exists.
func (o *Object) OwnProp(key string) (Value, bool) {
	if o.Class == ClassArray {
		if i, ok := arrayIndex(key); ok {
			if i < len(o.Elems) {
				return o.Elems[i], true
			}
			return Undefined, false
		}
	}
	v, ok := o.props[key]
	return v, ok
}

// OwnKeys returns the own enumerable string keys in JavaScript order: integer keys ascending, then the rest in
// insertion order.
func (o *Object) OwnKeys() []string {
	var ints []string
	var rest []string
	for i := range o.Elems {
		ints = append(ints, strconv.Itoa(i))
	}
	var extra []uint64
	for _, k := range o.keys {
		if n, ok := canonicalIndex(k); ok {
			extra = append(extra, n)
			continue
		}
		rest = append(rest, k)
	}
	if len(extra) > 0 {
		sortUint64(extra)
		for _, n := range extra {
			ints = append(ints, strconv.FormatUint(n, 10))
		}
	}
	return append(ints, rest...)
}

func sortUint64(a []uint64) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}

// canonicalIndex reports whether k is a canonical array index ("0", "17", never "01").
func canonicalIndex(k string) (uint64, bool) {
	if k == "" || len(k) > 10 || (len(k) > 1 && k[0] == '0') {
		return 0, false
	}
	var n uint64
	for i := 0; i < len(k); i++ {
		if k[i] < '0' || k[i] > '9' {
			return 0, false
		}
		n = n*10 + uint64(k[i]-'0')
	}
	if n >= 4294967295 {
		return 0, false
	}
	return n, true
}

func arrayIndex(k string) (int, bool) {
	n, ok := canonicalIndex(k)
	if !ok || n > 1<<24 {
		return 0, false
	}
	return int(n), true
}
