package validator_test

import (
	"testing"

	validator "github.com/validatorjs/validator.js"
	"github.com/validatorjs/validator.js/js"
)

// TestExports ports test/exports.test.js: the locale lists are arrays, validators and sanitizers are callable,
// toString is the library's own and version matches the module.
func TestExports(t *testing.T) {
	for _, name := range []string{"passportNumberLocales", "isPostalCodeLocales", "isAlphaLocales",
		"isAlphanumericLocales", "isMobilePhoneLocales", "isFloatLocales", "ibanLocales"} {
		v, err := validator.Call(name, nil)
		if err != nil || !v.IsArray() || len(v.Obj().Elems) == 0 {
			t.Errorf("%s = %v, %v; want a non-empty array", name, v, err)
		}
	}
	for _, name := range []string{"isEmail", "isAlpha", "toBoolean", "toFloat"} {
		if _, err := validator.Call(name, []js.Value{js.Str("x")}); err != nil {
			t.Errorf("%s is not callable: %v", name, err)
		}
	}
	toString := map[string]js.Value{"test": js.Str("test"), "123": js.Num(123), "": js.Null}
	for want, in := range toString {
		got, err := validator.Call("toString", []js.Value{in})
		if err != nil || got.String() != want {
			t.Errorf("toString(%v) = %v, %v; want %q", in, got, err, want)
		}
	}
	if got, _ := validator.Call("toString", []js.Value{js.Undefined}); got.String() != "" {
		t.Errorf("toString(undefined) = %q, want empty", got.String())
	}
	if v, err := validator.Call("version", nil); err != nil || v.String() != validator.Version || v.String() != "13.15.35" {
		t.Errorf("version = %v, %v; want %q", v, err, "13.15.35")
	}
	if n := len(validator.Names()); n != 112 {
		t.Errorf("Names() has %d exports, want 112", n)
	}
}
