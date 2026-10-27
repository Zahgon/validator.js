package validator

import (
	"encoding/json"
	"strings"

	"github.com/validatorjs/validator.js/internal/jsre"
	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isMagnetURI", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Bool(strings.HasPrefix(s, "magnet:?") && test(reMagnet, s))
	})
	register("isDataURI", isDataURI)
	register("isMimeType", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Bool(test(reMimeSimple, s) || test(reMimeText, s) || test(reMimeMultipart, s))
	})
	register("isBase32", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		options := merge(arg(a, 1), []option{{"crockford", js.Bool(false)}})
		if optTrue(options, "crockford") {
			return js.Bool(test(reCrockford32, s))
		}
		return js.Bool(js.Len16(s)%8 == 0 && test(reBase32, s))
	})
	registerPattern("isBase58", re(`^[A-HJ-NP-Za-km-z1-9]*$`, ""))
	register("isBase64", func(a []js.Value) js.Value { return js.Bool(isBase64(arg(a, 0), arg(a, 1))) })
	register("isJWT", func(a []js.Value) js.Value {
		parts := js.Split16(assertString(arg(a, 0)), ".")
		if len(parts) != 3 {
			return js.Bool(false)
		}
		ok := true
		for _, p := range parts {
			if ok {
				o := js.NewObject()
				o.Obj().SetProp("urlSafe", js.Bool(true))
				ok = isBase64(js.Str(p), o)
			}
		}
		return js.Bool(ok)
	})
	register("isHash", isHash)
	registerPattern("isMD5", re(`^[a-f0-9]{32}$`, ""))
	register("isUUID", isUUID)
	registerPattern("isULID", re(`^[0-7][0-9A-HJKMNP-TV-Z]{25}$`, "i"))
	register("isMongoId", func(a []js.Value) js.Value {
		s := assertString(arg(a, 0))
		return js.Bool(test(reHexadecimal, s) && js.Len16(s) == 24)
	})
	register("isJSON", isJSON)
	registerPattern("isSemVer", reSemVer)
	registerPattern("isISRC", re(`^[A-Z]{2}[0-9A-Z]{3}\d{2}\d{5}$`, ""))
}

var reMagnet = re(`(?:^magnet:\?|[^?&]&)xt(?:\.1)?=urn:(?:(?:aich|bitprint|btih|ed2k|ed2khash|kzhash|md5|sha1|tree:tiger):[a-z0-9]{32}(?:[a-z0-9]{8})?|btmh:1220[a-z0-9]{64})(?:$|&)`, "i")
var reMediaType = re(`^[a-z]+\/[a-z0-9\-\+\._]+$`, "i")
var reDataAttribute = re(`^[a-z\-]+=[a-z0-9\-]+$`, "i")
var reDataPayload = re(`^[a-z0-9!\$&'\(\)\*\+,;=\-\._~:@\/\?%\s]*$`, "i")
var reMimeSimple = re(`^(application|audio|font|image|message|model|multipart|text|video)\/[a-zA-Z0-9\.\-\+_]{1,100}$`, "i")
var reMimeText = re(`^text\/[a-zA-Z0-9\.\-\+]{1,100};\s?charset=("[a-zA-Z0-9\.\-\+\s]{0,70}"|[a-zA-Z0-9\.\-\+]{0,70})(\s?\([a-zA-Z0-9\.\-\+\s]{1,20}\))?$`, "i")
var reMimeMultipart = re(`^multipart\/[a-zA-Z0-9\.\-\+]{1,100}(;\s?(boundary|charset)=("[a-zA-Z0-9\.\-\+\s]{0,70}"|[a-zA-Z0-9\.\-\+]{0,70})(\s?\([a-zA-Z0-9\.\-\+\s]{1,20}\))?){0,2}$`, "i")
var reBase32 = re(`^[A-Z2-7]+=*$`, "")
var reCrockford32 = re(`^[A-HJKMNP-TV-Z0-9]+$`, "")
var reBase64Padded = re(`^[A-Za-z0-9+/]+={0,2}$`, "")
var reBase64 = re(`^[A-Za-z0-9+/]+$`, "")
var reBase64URLPadded = re(`^[A-Za-z0-9_-]+={0,2}$`, "")
var reBase64URL = re(`^[A-Za-z0-9_-]+$`, "")
var reHexadecimal = re(`^(0x|0h)?[0-9A-F]+$`, "i")
var reSemVer = re(strings.Join([]string{
	`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)`,
	`(?:-((?:0|[1-9]\d*|\d*[a-z-][0-9a-z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-z-][0-9a-z-]*))*))`,
	`?(?:\+([0-9a-z-]+(?:\.[0-9a-z-]+)*))?$`,
}, ""), "i")

func isDataURI(a []js.Value) js.Value {
	data := js.Split16(assertString(arg(a, 0)), ",")
	if len(data) < 2 {
		return js.Bool(false)
	}
	attributes := js.Split16(js.Trim(data[0]), ";")
	data = data[1:]
	scheme := attributes[0]
	attributes = attributes[1:]
	if js.SliceJS(scheme, 0, 5) != "data:" {
		return js.Bool(false)
	}
	mediaType := js.SliceJS(scheme, 5, js.Len16(scheme))
	if mediaType != "" && !test(reMediaType, mediaType) {
		return js.Bool(false)
	}
	for i, attr := range attributes {
		if !(i == len(attributes)-1 && js.ToLowerCase(attr) == "base64") && !test(reDataAttribute, attr) {
			return js.Bool(false)
		}
	}
	for _, d := range data {
		if !test(reDataPayload, d) {
			return js.Bool(false)
		}
	}
	return js.Bool(true)
}

func isBase64(str js.Value, options js.Value) bool {
	s := assertString(str)
	padding := !js.Truthy(optional(options, "urlSafe"))
	options = merge(options, []option{{"urlSafe", js.Bool(false)}, {"padding", js.Bool(padding)}})
	if s == "" {
		return true
	}
	n := js.Len16(s)
	if optTrue(options, "padding") && n%4 != 0 {
		return false
	}
	var r *jsre.Regexp
	switch {
	case optTrue(options, "urlSafe") && optTrue(options, "padding"):
		r = reBase64URLPadded
	case optTrue(options, "urlSafe"):
		r = reBase64URL
	case optTrue(options, "padding"):
		r = reBase64Padded
	default:
		r = reBase64
	}
	return (!optTrue(options, "padding") || n%4 == 0) && test(r, s)
}

var hashLengths = map[string]string{"md5": "32", "md4": "32", "sha1": "40", "sha256": "64", "sha384": "96",
	"sha512": "128", "ripemd128": "32", "ripemd160": "40", "tiger128": "32", "tiger160": "40", "tiger192": "48",
	"crc32": "8", "crc32b": "8"}

func isHash(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	key := js.ToPropertyKey(arg(a, 1))
	n, ok := hashLengths[key]
	switch {
	case ok:
	case js.IsObjectProtoKey(key):
		n = js.ToString(js.NewFunction(key))
	default:
		n = "undefined"
	}
	rx := newRegExp(js.Str("^[a-fA-F0-9]{"+n+"}$"), js.Undefined)
	return js.Bool(regexpTest(rx, js.Str(s)))
}

var uuidPatterns = map[string]*jsre.Regexp{
	"1":     re(`^[0-9A-F]{8}-[0-9A-F]{4}-1[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, "i"),
	"2":     re(`^[0-9A-F]{8}-[0-9A-F]{4}-2[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, "i"),
	"3":     re(`^[0-9A-F]{8}-[0-9A-F]{4}-3[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, "i"),
	"4":     re(`^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, "i"),
	"5":     re(`^[0-9A-F]{8}-[0-9A-F]{4}-5[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, "i"),
	"6":     re(`^[0-9A-F]{8}-[0-9A-F]{4}-6[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, "i"),
	"7":     re(`^[0-9A-F]{8}-[0-9A-F]{4}-7[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, "i"),
	"8":     re(`^[0-9A-F]{8}-[0-9A-F]{4}-8[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, "i"),
	"nil":   re(`^00000000-0000-0000-0000-000000000000$`, "i"),
	"max":   re(`^ffffffff-ffff-ffff-ffff-ffffffffffff$`, "i"),
	"loose": re(`^[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}$`, "i"),
	"all":   re(`^(?:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$`, "i"),
}

func isUUID(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	version := arg(a, 1)
	if version.IsNullish() {
		version = js.Str("all")
	}
	key := js.ToPropertyKey(version)
	if r, ok := uuidPatterns[key]; ok {
		return js.Bool(test(r, s))
	}
	if js.IsObjectProtoKey(key) {
		js.Throw("TypeError", "uuid[version].test is not a function")
	}
	return js.Bool(false)
}

func isJSON(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	options := merge(arg(a, 1), []option{{"allow_primitives", js.Bool(false)}, {"allow_any_value", js.Bool(false)}})
	if !json.Valid([]byte(s)) {
		return js.Bool(false)
	}
	if optTrue(options, "allow_any_value") {
		return js.Bool(true)
	}
	switch strings.TrimLeft(s, " \t\r\n")[0] {
	case '{', '[':
		return js.Bool(true)
	case 'n', 't', 'f':
		return js.Bool(optTrue(options, "allow_primitives"))
	}
	return js.Bool(false)
}
