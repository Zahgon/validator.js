package validator

import (
	"testing"

	"github.com/validatorjs/validator.js/js"
)

// TestTypeOf ports the typeOf case of test/util.test.js.
func TestTypeOf(t *testing.T) {
	cases := []struct {
		in   js.Value
		want string
	}{
		{js.NewArray(), "array"}, {js.Null, "null"}, {js.NewObject(), "object"}, {js.NewDate(0), "date"},
		{js.Str("ezkemboi"), "string"}, {js.Undefined, "undefined"}, {js.Num(2021), "number"},
	}
	for _, c := range cases {
		if got := js.TypeTag(c.in); got != c.want {
			t.Errorf("typeOf(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestAssertString ports the assertString cases of test/util.test.js.
func TestAssertString(t *testing.T) {
	throws := map[string]js.Value{
		"undefined": js.Undefined, "null": js.Null, "Boolean": js.Bool(true), "Date": js.NewDate(0),
		"Number(NaN)": js.Num(nan()), "Number": js.Num(2024), "Object": js.NewObject(), "Array": js.NewArray(),
	}
	for name, v := range throws {
		err := js.Catch(func() { assertString(v) })
		if err == nil || err.Name != "TypeError" {
			t.Errorf("assertString(%s) error = %v, want a TypeError", name, err)
		}
	}
	for _, s := range []string{"", "antidisestablishmentarianism"} {
		if err := js.Catch(func() { assertString(js.Str(s)) }); err != nil {
			t.Errorf("assertString(%q) threw %v", s, err)
		}
	}
	err := js.Catch(func() { assertString(js.Num(2024)) })
	if err == nil || err.Message != "Expected a string but received a Number" {
		t.Errorf("assertString(2024) message = %v, want %q", err, "Expected a string but received a Number")
	}
}
