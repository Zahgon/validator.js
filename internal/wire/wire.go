// Package wire converts between JSON and JavaScript values with the tags of the validator command's contract:
// {"$undefined": true}, {"$number": "NaN" | "Infinity" | "-Infinity" | "-0"}, {"$regexp": [source, flags]} and
// {"$date": ...}.
package wire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"

	validator "github.com/validatorjs/validator.js"
	"github.com/validatorjs/validator.js/js"
)

// DecodeArgs decodes a JSON array of arguments, keeping object key order and resolving the tagged values.
func DecodeArgs(raw json.RawMessage) ([]js.Value, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	if !v.IsArray() {
		return nil, errors.New("args must be an array")
	}
	return v.Obj().Elems, nil
}

func decodeValue(dec *json.Decoder) (js.Value, error) {
	tok, err := dec.Token()
	if err != nil {
		return js.Undefined, err
	}
	switch t := tok.(type) {
	case nil:
		return js.Null, nil
	case bool:
		return js.Bool(t), nil
	case string:
		return js.Str(t), nil
	case json.Number:
		f, err := strconv.ParseFloat(string(t), 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return js.Undefined, err
		}
		return js.Num(f), nil
	case json.Delim:
		if t == '[' {
			var elems []js.Value
			for dec.More() {
				e, err := decodeValue(dec)
				if err != nil {
					return js.Undefined, err
				}
				elems = append(elems, e)
			}
			if _, err := dec.Token(); err != nil {
				return js.Undefined, err
			}
			return js.NewArray(elems...), nil
		}
		obj := js.NewObject()
		var keys []string
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return js.Undefined, err
			}
			key, _ := kt.(string)
			val, err := decodeValue(dec)
			if err != nil {
				return js.Undefined, err
			}
			keys = append(keys, key)
			obj.Obj().SetProp(key, val)
		}
		if _, err := dec.Token(); err != nil {
			return js.Undefined, err
		}
		if len(keys) == 1 {
			return decodeTag(keys[0], obj)
		}
		return obj, nil
	}
	return js.Undefined, fmt.Errorf("unexpected token %v", tok)
}

func decodeTag(key string, obj js.Value) (js.Value, error) {
	v, _ := obj.Obj().OwnProp(key)
	switch key {
	case "$undefined":
		return js.Undefined, nil
	case "$number":
		switch js.ToString(v) {
		case "NaN":
			return js.Num(math.NaN()), nil
		case "Infinity":
			return js.Num(math.Inf(1)), nil
		case "-Infinity":
			return js.Num(math.Inf(-1)), nil
		case "-0":
			return js.Num(math.Copysign(0, -1)), nil
		}
		return js.Undefined, fmt.Errorf("bad $number tag")
	case "$regexp":
		if !v.IsArray() || len(v.Obj().Elems) != 2 {
			return js.Undefined, fmt.Errorf("bad $regexp tag")
		}
		rx, err := validator.NewRegExp(v.Obj().Elems[0], v.Obj().Elems[1])
		if err != nil {
			return js.Undefined, err
		}
		return rx, nil
	case "$date":
		if v.IsNull() {
			return js.NewDate(math.NaN()), nil
		}
		return js.NewDate(js.ToNumber(v)), nil
	}
	return obj, nil
}

// WriteString writes s as a JSON string without HTML escaping.
func WriteString(b *bytes.Buffer, s string) {
	const hex = "0123456789abcdef"
	b.WriteByte('"')
	for _, u := range js.ToU16(s) {
		switch {
		case u == '"' || u == '\\':
			b.WriteByte('\\')
			b.WriteByte(byte(u))
		case u == '\n':
			b.WriteString(`\n`)
		case u == '\r':
			b.WriteString(`\r`)
		case u == '\t':
			b.WriteString(`\t`)
		case u < 0x20 || (u >= 0xD800 && u <= 0xDFFF):
			b.WriteString(`\u`)
			for shift := 12; shift >= 0; shift -= 4 {
				b.WriteByte(hex[u>>uint(shift)&0xF])
			}
		default:
			b.WriteString(js.FromU16([]uint16{u}))
		}
	}
	b.WriteByte('"')
}

// Encode writes a result value as JSON with the tags of the command's contract; a Date is its toISOString text.
func Encode(b *bytes.Buffer, v js.Value) {
	switch v.Kind() {
	case js.KindUndefined:
		b.WriteString(`{"$undefined":true}`)
	case js.KindNull:
		b.WriteString("null")
	case js.KindBoolean:
		b.WriteString(strconv.FormatBool(v.Boolean()))
	case js.KindNumber:
		n := v.Number()
		switch {
		case math.IsNaN(n):
			b.WriteString(`{"$number":"NaN"}`)
		case math.IsInf(n, 1):
			b.WriteString(`{"$number":"Infinity"}`)
		case math.IsInf(n, -1):
			b.WriteString(`{"$number":"-Infinity"}`)
		case n == 0 && math.Signbit(n):
			b.WriteString(`{"$number":"-0"}`)
		default:
			b.WriteString(js.NumberToString(n))
		}
	case js.KindString:
		WriteString(b, v.String())
	case js.KindObject:
		encodeObject(b, v)
	}
}

func encodeObject(b *bytes.Buffer, v js.Value) {
	o := v.Obj()
	switch o.Class {
	case js.ClassDate:
		if math.IsNaN(o.Time) {
			b.WriteString(`{"$date":null}`)
			return
		}
		b.WriteString(`{"$date":`)
		WriteString(b, js.DateISOString(o.Time))
		b.WriteString("}")
	case js.ClassRegExp:
		b.WriteString(`{"$regexp":[`)
		WriteString(b, o.Source)
		b.WriteString(",")
		WriteString(b, o.Flags)
		b.WriteString("]}")
	case js.ClassArray:
		b.WriteString("[")
		for i, e := range o.Elems {
			if i > 0 {
				b.WriteString(",")
			}
			Encode(b, e)
		}
		b.WriteString("]")
	default:
		b.WriteString("{")
		for i, k := range o.OwnKeys() {
			if i > 0 {
				b.WriteString(",")
			}
			WriteString(b, k)
			b.WriteString(":")
			val, _ := o.OwnProp(k)
			Encode(b, val)
		}
		b.WriteString("}")
	}
}
