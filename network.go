package validator

import (
	"strings"

	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isFQDN", func(a []js.Value) js.Value { return js.Bool(isFQDN(assertString(arg(a, 0)), arg(a, 1))) })
	register("isIP", func(a []js.Value) js.Value { return js.Bool(isIP(a)) })
	register("isIPRange", isIPRange)
	register("isMACAddress", func(a []js.Value) js.Value { return js.Bool(isMACAddress(arg(a, 0), arg(a, 1))) })
	register("isLatLong", isLatLong)
}

var fqdnDefaults = []option{{"require_tld", js.Bool(true)}, {"allow_underscores", js.Bool(false)},
	{"allow_trailing_dot", js.Bool(false)}, {"allow_numeric_tld", js.Bool(false)}, {"allow_wildcard", js.Bool(false)},
	{"ignore_max_length", js.Bool(false)}}

var reTLD = re(`^([a-z\u00A1-\u00A8\u00AA-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF]{2,}|xn[a-z0-9-]{2,})$`, "i")
var reWhitespace = re(`\s`, "")
var reDigits = re(`^\d+$`, "")
var reFQDNPart = re(`^[a-z_\u00a1-\uffff0-9-]+$`, "i")
var reFullWidthASCII = re(`[\uff01-\uff5e]`, "")
var reHyphenEdge = re(`^-|-$`, "")

func isFQDN(str string, options js.Value) bool {
	options = merge(options, fqdnDefaults)
	if optTrue(options, "allow_trailing_dot") && js.CharAt(str, js.Len16(str)-1) == "." {
		str = js.Substring(str, 0, js.Len16(str)-1)
	}
	if w := opt(options, "allow_wildcard"); w.IsBoolean() && w.Boolean() && strings.HasPrefix(str, "*.") {
		str = js.Substring(str, 2, js.Len16(str))
	}
	parts := js.Split16(str, ".")
	tld := parts[len(parts)-1]
	if optTrue(options, "require_tld") {
		if len(parts) < 2 {
			return false
		}
		if !optTrue(options, "allow_numeric_tld") && !test(reTLD, tld) {
			return false
		}
		if test(reWhitespace, tld) {
			return false
		}
	}
	if !optTrue(options, "allow_numeric_tld") && test(reDigits, tld) {
		return false
	}
	for _, part := range parts {
		if js.Len16(part) > 63 && !optTrue(options, "ignore_max_length") {
			return false
		}
		if !test(reFQDNPart, part) || test(reFullWidthASCII, part) || test(reHyphenEdge, part) {
			return false
		}
		if !optTrue(options, "allow_underscores") && strings.Contains(part, "_") {
			return false
		}
	}
	return true
}

const ipv4Segment = `(?:[0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])`
const ipv4Address = `(` + ipv4Segment + `[.]){3}` + ipv4Segment
const ipv6Segment = `(?:[0-9a-fA-F]{1,4})`

var reIPv4 = re(`^`+ipv4Address+`$`, "")
var reIPv6 = re(`^(`+
	`(?:`+ipv6Segment+`:){7}(?:`+ipv6Segment+`|:)|`+
	`(?:`+ipv6Segment+`:){6}(?:`+ipv4Address+`|:`+ipv6Segment+`|:)|`+
	`(?:`+ipv6Segment+`:){5}(?::`+ipv4Address+`|(:`+ipv6Segment+`){1,2}|:)|`+
	`(?:`+ipv6Segment+`:){4}(?:(:`+ipv6Segment+`){0,1}:`+ipv4Address+`|(:`+ipv6Segment+`){1,3}|:)|`+
	`(?:`+ipv6Segment+`:){3}(?:(:`+ipv6Segment+`){0,2}:`+ipv4Address+`|(:`+ipv6Segment+`){1,4}|:)|`+
	`(?:`+ipv6Segment+`:){2}(?:(:`+ipv6Segment+`){0,3}:`+ipv4Address+`|(:`+ipv6Segment+`){1,5}|:)|`+
	`(?:`+ipv6Segment+`:){1}(?:(:`+ipv6Segment+`){0,4}:`+ipv4Address+`|(:`+ipv6Segment+`){1,6}|:)|`+
	`(?::((?::`+ipv6Segment+`){0,5}:`+ipv4Address+`|(?::`+ipv6Segment+`){1,7}|:))`+
	`)(%[0-9a-zA-Z.]{1,})?$`, "")

// isIP implements isIP(ipAddress, options = {}), where a non-object second argument is the version.
func isIP(a []js.Value) bool {
	ip := assertString(arg(a, 0))
	options := arg(a, 1)
	if options.IsUndefined() {
		options = js.NewObject()
	}
	var version js.Value
	if js.TypeOf(options) == "object" {
		version = js.Get(options, "version")
	} else {
		version = arg(a, 1)
	}
	if !js.Truthy(version) {
		return isIPVersion(ip, "4") || isIPVersion(ip, "6")
	}
	return isIPVersion(ip, callToString(version))
}

func isIPVersion(ip, version string) bool {
	switch version {
	case "4":
		return test(reIPv4, ip)
	case "6":
		return test(reIPv6, ip)
	}
	return false
}

// callToString implements value.toString() for a value whose toString is looked up as a method.
func callToString(v js.Value) string {
	if v.IsObject() {
		if f, ok := v.Obj().OwnProp("toString"); ok && !f.IsFunction() {
			js.Throw("TypeError", "version.toString is not a function")
		}
		if v.Obj().Class == js.ClassObject {
			return "[object Object]"
		}
	}
	js.Get(v, "toString")
	return js.ToString(v)
}

var reSubnetMaybe = re(`^\d{1,3}$`, "")

func isIPRange(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	version := arg(a, 1)
	if version.IsUndefined() {
		version = js.Str("")
	}
	parts := js.Split16(s, "/")
	if len(parts) != 2 || !test(reSubnetMaybe, parts[1]) {
		return js.Bool(false)
	}
	if js.Len16(parts[1]) > 1 && strings.HasPrefix(parts[1], "0") {
		return js.Bool(false)
	}
	if !isIP([]js.Value{js.Str(parts[0]), version}) {
		return js.Bool(false)
	}
	var expected float64
	switch js.ToString(version) {
	case "4":
		expected = 32
	case "6":
		expected = 128
	default:
		expected = 32
		if isIP([]js.Value{js.Str(parts[0]), js.Str("6")}) {
			expected = 128
		}
	}
	sub := js.Str(parts[1])
	return js.Bool(js.LessEq(sub, js.Num(expected)) && js.GreaterEq(sub, js.Num(0)))
}

var reMAC48 = re(`^(?:[0-9a-fA-F]{2}([-:\s]))([0-9a-fA-F]{2}\1){4}([0-9a-fA-F]{2})$`, "")
var reMAC48NoSep = re(`^([0-9a-fA-F]){12}$`, "")
var reMAC48Dots = re(`^([0-9a-fA-F]{4}\.){2}([0-9a-fA-F]{4})$`, "")
var reMAC64 = re(`^(?:[0-9a-fA-F]{2}([-:\s]))([0-9a-fA-F]{2}\1){6}([0-9a-fA-F]{2})$`, "")
var reMAC64NoSep = re(`^([0-9a-fA-F]){16}$`, "")
var reMAC64Dots = re(`^([0-9a-fA-F]{4}\.){3}([0-9a-fA-F]{4})$`, "")

// optional reads options?.key.
func optional(options js.Value, key string) js.Value {
	if options.IsNullish() {
		return js.Undefined
	}
	return js.Get(options, key)
}

func isMACAddress(str js.Value, options js.Value) bool {
	s := assertString(str)
	if js.Truthy(optional(options, "eui")) {
		options.Obj().SetProp("eui", js.Str(js.ToString(js.Get(options, "eui"))))
	}
	eui := optional(options, "eui")
	is := func(want string) bool { return eui.IsString() && eui.String() == want }
	if js.Truthy(optional(options, "no_colons")) || js.Truthy(optional(options, "no_separators")) {
		if is("48") {
			return test(reMAC48NoSep, s)
		}
		if is("64") {
			return test(reMAC64NoSep, s)
		}
		return test(reMAC48NoSep, s) || test(reMAC64NoSep, s)
	}
	if is("48") {
		return test(reMAC48, s) || test(reMAC48Dots, s)
	}
	if is("64") {
		return test(reMAC64, s) || test(reMAC64Dots, s)
	}
	o48, o64 := js.NewObject(), js.NewObject()
	o48.Obj().SetProp("eui", js.Str("48"))
	o64.Obj().SetProp("eui", js.Str("64"))
	return isMACAddress(js.Str(s), o48) || isMACAddress(js.Str(s), o64)
}

var reLat = re(`^\(?[+-]?(90(\.0+)?|[1-8]?\d(\.\d+)?)$`, "")
var reLong = re(`^\s?[+-]?(180(\.0+)?|1[0-7]\d(\.\d+)?|\d{1,2}(\.\d+)?)\)?$`, "")
var reLatDMS = re(`^(([1-8]?\d)\D+([1-5]?\d|60)\D+([1-5]?\d|60)(\.\d+)?|90\D+0\D+0)\D+[NSns]?$`, "i")
var reLongDMS = re(`^\s*([1-7]?\d{1,2}\D+([1-5]?\d|60)\D+([1-5]?\d|60)(\.\d+)?|180\D+0\D+0)\D+[EWew]?$`, "i")

func isLatLong(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := merge(arg(a, 1), []option{{"checkDMS", js.Bool(false)}})
	if !strings.Contains(s, ",") {
		return js.Bool(false)
	}
	pair := js.Split16(s, ",")
	if (strings.HasPrefix(pair[0], "(") && !strings.HasSuffix(pair[1], ")")) ||
		(strings.HasSuffix(pair[1], ")") && !strings.HasPrefix(pair[0], "(")) {
		return js.Bool(false)
	}
	if optTrue(options, "checkDMS") {
		return js.Bool(test(reLatDMS, pair[0]) && test(reLongDMS, pair[1]))
	}
	return js.Bool(test(reLat, pair[0]) && test(reLong, pair[1]))
}
