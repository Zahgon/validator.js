package validator

import (
	"math"
	"strings"

	"github.com/validatorjs/validator.js/internal/jsdate"
	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isIdentityCard", isIdentityCard)
}

type identityValidator struct {
	locale string
	check  func(str string) js.Value
}

var identityValidators []identityValidator

func init() {
	identityValidators = []identityValidator{
		{"PL", idPL}, {"ES", idES}, {"FI", idFI}, {"IN", idIN}, {"IR", idIR}, {"IT", idIT}, {"NO", idNO},
		{"TH", idTH}, {"LK", idLK}, {"he-IL", idHeIL}, {"ar-LY", trimmedPattern(re(`^(1|2)\d{11}$`, ""))},
		{"ar-TN", trimmedPattern(re(`^\d{8}$`, ""))}, {"zh-CN", idZhCN}, {"zh-HK", idZhHK}, {"zh-TW", idZhTW},
		{"PK", trimmedPattern(re(`^[1-7][0-9]{4}-[0-9]{7}-[1-9]$`, ""))},
	}
}

func trimmedPattern(r interface{ Test(string) bool }) func(string) js.Value {
	return func(s string) js.Value { return js.Bool(r.Test(js.Trim(s))) }
}

// callProtoMethod emulates table[key](str) for a key naming an Object.prototype member of a plain object of
// functions; has reports the object's own keys.
func callProtoMethod(key string, str string, has func(string) bool) js.Value {
	switch key {
	case "toString", "toLocaleString":
		return js.Str("[object Object]")
	case "hasOwnProperty", "propertyIsEnumerable":
		return js.Bool(has(str))
	case "isPrototypeOf":
		return js.Bool(false)
	case "valueOf":
		return js.NewObject()
	case "constructor":
		o := js.NewObject()
		u := js.ToU16(str)
		for i := range u {
			o.Obj().SetProp(itoa(i), js.Str(js.FromU16(u[i:i+1])))
		}
		return o
	case "__lookupGetter__", "__lookupSetter__":
		return js.Undefined
	case "__defineGetter__", "__defineSetter__":
		js.Throw("TypeError", "Object.prototype."+key+": Expecting function")
	}
	js.Throw("TypeError", "validators[locale] is not a function")
	return js.Undefined
}

func isIdentityCard(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	locale := arg(a, 1)
	key := js.ToPropertyKey(locale)
	for _, v := range identityValidators {
		if v.locale == key {
			return v.check(s)
		}
	}
	if js.IsObjectProtoKey(key) {
		return callProtoMethod(key, s, func(k string) bool {
			for _, v := range identityValidators {
				if v.locale == k {
					return true
				}
			}
			return false
		})
	}
	if locale.IsString() && locale.String() == "any" {
		for _, v := range identityValidators {
			if js.Truthy(v.check(s)) {
				return js.Bool(true)
			}
		}
		return js.Bool(false)
	}
	js.Throw("Error", "Invalid locale '"+js.ToString(locale)+"'")
	return js.Undefined
}

// num converts one code unit string with Number().
func num(s string) float64 { return js.StringToNumber(s) }

// unit returns the code unit at index i as a string ("" past the end), like str[i] used as a value.
func unit(u []uint16, i int) string {
	if i < 0 || i >= len(u) {
		return ""
	}
	return js.FromU16(u[i : i+1])
}

func idPL(s string) js.Value {
	weights := []float64{1, 3, 7, 9, 1, 3, 7, 9, 1, 3, 0}
	u := js.ToU16(s)
	lz := js.NewObject()
	lz.Obj().SetProp("allow_leading_zeroes", js.Bool(true))
	if len(u) == 11 && isInt(js.Str(s), lz) {
		sum := 0.0
		for i := 0; i < 10; i++ {
			sum += num(unit(u, i)) * weights[i]
		}
		modulo := math.Mod(sum, 10)
		last := num(unit(u, 10))
		if (modulo == 0 && last == 0) || last == 10-modulo {
			return js.Bool(true)
		}
	}
	return js.Bool(false)
}

var reDNI = re(`^[0-9X-Z][0-9]{7}[TRWAGMYFPDXBNJZSQVHLCKE]$`, "")
var reDNIPrefix = re(`[X,Y,Z]`, "g")

func idES(s string) js.Value {
	sanitized := js.ToUpperCase(js.Trim(s))
	if !test(reDNI, sanitized) {
		return js.Bool(false)
	}
	number := replaceFunc(js.SliceJS(sanitized, 0, -1), reDNIPrefix, func(m match) string {
		switch m.text {
		case "X":
			return "0"
		case "Y":
			return "1"
		case "Z":
			return "2"
		}
		return "undefined"
	})
	control := "TRWAGMYFPDXBNJZSQVHLCKE"
	return js.Bool(strings.HasSuffix(sanitized, string(control[int(math.Mod(num(number), 23))])))
}

var reFinnishID = re(`^\d{6}[\-A\+]\d{3}[0-9ABCDEFHJKLMNPRSTUVWXY]{1}$`, "")

func idFI(s string) js.Value {
	if js.Len16(s) != 11 || !test(reFinnishID, s) {
		return js.Bool(false)
	}
	id := js.ParseInt(js.Str(s[0:6]), js.Num(10))*1000 + js.ParseInt(js.Str(s[7:10]), js.Num(10))
	check := "0123456789ABCDEFHJKLMNPRSTUVWXY"[int(math.Mod(id, 31))]
	return js.Bool(string(check) == s[10:11])
}

var reAadhaar = re(`^[1-9]\d{3}\s?\d{4}\s?\d{4}$`, "")

func idIN(s string) js.Value {
	sanitized := js.Trim(s)
	if !test(reAadhaar, sanitized) {
		return js.Bool(false)
	}
	return js.Bool(verhoeffCheck(replace(sanitized, reSpaces, "")))
}

var reTenDigits = re(`^\d{10}$`, "")

func idIR(s string) js.Value {
	if !test(reTenDigits, s) {
		return js.Bool(false)
	}
	if js.ParseInt(js.Str(s[3:9]), js.Num(10)) == 0 {
		return js.Bool(false)
	}
	last := int(s[9] - '0')
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(s[i]-'0') * (10 - i)
	}
	sum %= 11
	return js.Bool((sum < 2 && last == sum) || (sum >= 2 && last == 11-sum))
}

var reItalianID = re(`C[A-Z]\d{5}[A-Z]{2}`, "is")

func idIT(s string) js.Value {
	if js.Len16(s) != 9 || s == "CA00000AA" {
		return js.Bool(false)
	}
	return js.Bool(reItalianID.Search(js.ToU16(s), 0) != nil)
}

func idNO(s string) js.Value {
	sanitized := js.Trim(s)
	if math.IsNaN(js.StringToNumber(sanitized)) {
		return js.Bool(false)
	}
	u := js.ToU16(sanitized)
	if len(u) != 11 || sanitized == "00000000000" {
		return js.Bool(false)
	}
	f := make([]float64, 11)
	for i := range f {
		f[i] = num(unit(u, i))
	}
	k1 := math.Mod(11-math.Mod(3*f[0]+7*f[1]+6*f[2]+1*f[3]+8*f[4]+9*f[5]+4*f[6]+5*f[7]+2*f[8], 11), 11)
	k2 := math.Mod(11-math.Mod(5*f[0]+4*f[1]+3*f[2]+2*f[3]+7*f[4]+6*f[5]+5*f[6]+4*f[7]+3*f[8]+2*k1, 11), 11)
	return js.Bool(k1 == f[9] && k2 == f[10])
}

var reThaiID = re(`^[1-8]\d{12}$`, "")

func idTH(s string) js.Value {
	if !test(reThaiID, s) {
		return js.Bool(false)
	}
	sum := 0
	for i := 0; i < 12; i++ {
		sum += int(s[i]-'0') * (13 - i)
	}
	return js.Bool(s[12:13] == itoa((11-sum%11)%10))
}

var reOldNIC = re(`^[1-9]\d{8}[vx]$`, "i")
var reNewNIC = re(`^[1-9]\d{11}$`, "i")

func idLK(s string) js.Value {
	n := js.Len16(s)
	return js.Bool((n == 10 && test(reOldNIC, s)) || (n == 12 && test(reNewNIC, s)))
}

var reNineDigits = re(`^\d{9}$`, "")

func idHeIL(s string) js.Value {
	id := js.Trim(s)
	if !test(reNineDigits, id) {
		return js.Bool(false)
	}
	sum := 0
	for i := 0; i < len(id); i++ {
		inc := int(id[i]-'0') * (i%2 + 1)
		if inc > 9 {
			inc -= 9
		}
		sum += inc
	}
	return js.Bool(sum%10 == 0)
}

var chinaProvinces = stringSet("11", "12", "13", "14", "15", "21", "22", "23", "31", "32", "33", "34", "35", "36", "37",
	"41", "42", "43", "44", "45", "46", "50", "51", "52", "53", "54", "61", "62", "63", "64", "65", "71", "81", "82",
	"91")

var reChinaID = re(`^\d{15}|(\d{17}(\d|x|X))$`, "")
var reChinaID15 = re(`^[1-9]\d{7}((0[1-9])|(1[0-2]))((0[1-9])|([1-2][0-9])|(3[0-1]))\d{3}$`, "")
var reChinaID18 = re(`^[1-9]\d{5}[1-9]\d{3}((0[1-9])|(1[0-2]))((0[1-9])|([1-2][0-9])|(3[0-1]))\d{3}(\d|x|X)$`, "")

// chinaBirthday checks a yyyymmdd code with new Date(yyyy, mm - 1, dd) against the clock and the calendar.
func chinaBirthday(code string) bool {
	yyyy := js.ParseInt(js.Str(js.Substring(code, 0, 4)), js.Num(10))
	mm := js.ParseInt(js.Str(js.Substring(code, 4, 6)), js.Num(10))
	dd := js.ParseInt(js.Str(js.Substring(code, 6, js.Len16(code))), js.Num(10))
	t := jsdate.Construct(yyyy, mm-1, dd, 0, 0, 0, 0)
	if !math.IsNaN(t) && t > jsdate.Now() {
		return false
	}
	if math.IsNaN(t) {
		return false
	}
	f := jsdate.Split(t)
	return float64(f.Year) == yyyy && float64(f.Month) == mm-1 && float64(f.Day) == dd
}

func chinaParity(id string) bool {
	powers := []float64{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	u := js.ToU16(id)
	power := 0.0
	for i := 0; i < 17; i++ {
		power += js.ParseInt(js.Str(unit(u, i)), js.Num(10)) * powers[i]
	}
	parity := []string{"1", "0", "X", "9", "8", "7", "6", "5", "4", "3", "2"}
	mod := math.Mod(power, 11)
	if math.IsNaN(mod) {
		return false
	}
	return parity[int(mod)] == js.ToUpperCase(unit(u, 17))
}

func idZhCN(s string) js.Value {
	if !test(reChinaID, s) {
		return js.Bool(false)
	}
	if js.Len16(s) == 15 {
		return js.Bool(test(reChinaID15, s) && chinaProvinces[s[0:2]] && chinaBirthday("19"+s[6:12]))
	}
	return js.Bool(test(reChinaID18, s) && chinaProvinces[s[0:2]] && chinaBirthday(s[6:14]) && chinaParity(s))
}

var reHKID = re(`^[A-Z]{1,2}[0-9]{6}((\([0-9A]\))|(\[[0-9A]\])|([0-9A]))$`, "")
var reBrackets = re(`\[|\]|\(|\)`, "g")

func idZhHK(s string) js.Value {
	s = js.ToUpperCase(js.Trim(s))
	if !test(reHKID, s) {
		return js.Bool(false)
	}
	s = replace(s, reBrackets, "")
	if len(s) == 8 {
		s = "3" + s
	}
	sum := 0
	for i := 0; i <= 7; i++ {
		c := int(s[i])
		v := c - '0'
		if c < '0' || c > '9' {
			v = (c - 55) % 11
		}
		sum += v * (9 - i)
	}
	sum %= 11
	var want string
	switch sum {
	case 0:
		want = "0"
	case 1:
		want = "A"
	default:
		want = itoa(11 - sum)
	}
	return js.Bool(want == s[len(s)-1:])
}

var reTaiwanID = re(`^[A-Z][0-9]{9}$`, "")

func idZhTW(s string) js.Value {
	codes := map[byte]int{'A': 10, 'B': 11, 'C': 12, 'D': 13, 'E': 14, 'F': 15, 'G': 16, 'H': 17, 'I': 34, 'J': 18,
		'K': 19, 'L': 20, 'M': 21, 'N': 22, 'O': 35, 'P': 23, 'Q': 24, 'R': 25, 'S': 26, 'T': 27, 'U': 28, 'V': 29,
		'W': 32, 'X': 30, 'Y': 31, 'Z': 33}
	sanitized := js.ToUpperCase(js.Trim(s))
	if !test(reTaiwanID, sanitized) {
		return js.Bool(false)
	}
	code := codes[sanitized[0]]
	sum := code%10*9 + code/10
	for i := 1; i < 9; i++ {
		sum += int(sanitized[i]-'0') * (9 - i)
	}
	return js.Bool((10-sum%10-int(sanitized[9]-'0'))%10 == 0)
}
