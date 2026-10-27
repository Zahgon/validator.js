package validator

import (
	"math"
	"strings"

	"github.com/validatorjs/validator.js/internal/jsre"
	"github.com/validatorjs/validator.js/js"
)

// Checksum and birth-date rules of isTaxID.js, one function per locale.

func init() {
	register("isTaxID", isTaxID)
}

// pi implements parseInt(s, 10).
func pi(s string) float64 { return js.ParseInt(js.Str(s), js.Num(10)) }

// sl implements s.slice(start, end) with a non-negative start and end.
func sl(s string, start, end int) string { return js.SliceJS(s, start, end) }

// from implements s.slice(start).
func from(s string, start int) string { return js.SliceJS(s, start, js.Len16(s)) }

// digitsOf implements s.split(”).map(a => parseInt(a, 10)).
func digitsOf(s string) []float64 {
	u := js.ToU16(s)
	out := make([]float64, len(u))
	for i := range u {
		out[i] = pi(js.FromU16(u[i : i+1]))
	}
	return out
}

// at reads an array element, NaN past the end (undefined in arithmetic).
func at(d []float64, i int) float64 {
	if i < 0 || i >= len(d) {
		return math.NaN()
	}
	return d[i]
}

func isDateFormat(date, format string) bool { return isDate(js.Str(date), js.Str(format)) }

// padYear inserts the missing zero of a three-character year such as "195" (from `19${5}`).
func padYear(y string) string {
	if js.Len16(y) == 3 {
		return sl(y, 0, 2) + "0" + from(y, 2)
	}
	return y
}

// twoDigits implements `if (n < 10) n = `0${n}“, returning the string the date template prints.
func twoDigits(n float64) string {
	if n < 10 {
		return "0" + js.NumberToString(n)
	}
	return js.NumberToString(n)
}

func bgBgCheck(tin string) bool {
	centuryYear := sl(tin, 0, 2)
	month := pi(sl(tin, 2, 4))
	switch {
	case month > 40:
		month -= 40
		centuryYear = "20" + centuryYear
	case month > 20:
		month -= 20
		centuryYear = "18" + centuryYear
	default:
		centuryYear = "19" + centuryYear
	}
	if !isDateFormat(centuryYear+"/"+twoDigits(month)+"/"+sl(tin, 4, 6), "YYYY/MM/DD") {
		return false
	}
	digits := digitsOf(tin)
	lookup := []float64{2, 4, 8, 5, 10, 9, 7, 3, 6}
	checksum := 0.0
	for i, m := range lookup {
		checksum += at(digits, i) * m
	}
	if math.Mod(checksum, 11) == 10 {
		checksum = 0
	} else {
		checksum = math.Mod(checksum, 11)
	}
	return checksum == at(digits, 9)
}

func isCanadianSIN(input string) bool {
	u := js.ToU16(input)
	var evenDigits string
	var total float64
	for i := range u {
		if i%2 == 1 {
			evenDigits += js.NumberToString(js.StringToNumber(js.FromU16(u[i:i+1])) * 2)
		}
	}
	first := true
	add := func(s string) {
		v := js.StringToNumber(s)
		if first {
			total, first = v, false
			return
		}
		total += v
	}
	for i := range u {
		if i%2 == 0 {
			add(js.FromU16(u[i : i+1]))
		}
	}
	for _, c := range js.Split16(evenDigits, "") {
		add(c)
	}
	if first {
		js.Throw("TypeError", "Reduce of empty array with no initial value")
	}
	return math.Mod(total, 10) == 0
}

var reNonWordFirst = re(`\W`, "")

func csCzCheck(tin string) bool {
	tin = replace(tin, reNonWordFirst, "")
	y := pi(sl(tin, 0, 2))
	n := js.Len16(tin)
	var fullYear string
	if n == 10 {
		if y < 54 {
			fullYear = "20" + js.NumberToString(y)
		} else {
			fullYear = "19" + js.NumberToString(y)
		}
	} else {
		if from(tin, 6) == "000" {
			return false
		}
		if y < 54 {
			fullYear = "19" + js.NumberToString(y)
		} else {
			return false
		}
	}
	fullYear = padYear(fullYear)
	month := pi(sl(tin, 2, 4))
	if month > 50 {
		month -= 50
	}
	if month > 20 {
		if pi(fullYear) < 2004 {
			return false
		}
		month -= 20
	}
	if !isDateFormat(fullYear+"/"+twoDigits(month)+"/"+sl(tin, 4, 6), "YYYY/MM/DD") {
		return false
	}
	if n == 10 && math.Mod(pi(tin), 11) != 0 {
		check := math.Mod(pi(sl(tin, 0, 9)), 11)
		if pi(fullYear) < 1986 && check == 10 {
			if pi(from(tin, 9)) != 0 {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func deDeCheck(tin string) bool {
	digits := digitsOf(tin)
	var occurrences []string
	for i := 0; i < len(digits)-1; i++ {
		occ := ""
		for j := 0; j < len(digits)-1; j++ {
			if digits[i] == digits[j] {
				occ += itoa(j)
			}
		}
		if js.Len16(occ) > 1 {
			occurrences = append(occurrences, occ)
		}
	}
	if len(occurrences) != 2 && len(occurrences) != 3 {
		return false
	}
	if js.Len16(occurrences[0]) == 3 {
		trip := digitsOf(occurrences[0])
		recurrent := 0
		for i := 0; i < len(trip)-1; i++ {
			if trip[i]+1 == trip[i+1] {
				recurrent++
			}
		}
		if recurrent == 2 {
			return false
		}
	}
	return iso7064Check(tin)
}

func dkDkCheck(tin string) bool {
	tin = replace(tin, reNonWordFirst, "")
	y := pi(sl(tin, 4, 6))
	var year string
	switch sl(tin, 6, 7) {
	case "0", "1", "2", "3":
		year = "19" + js.NumberToString(y)
	case "4", "9":
		if y < 37 {
			year = "20" + js.NumberToString(y)
		} else {
			year = "19" + js.NumberToString(y)
		}
	default:
		if y < 58 {
			year = "20" + js.NumberToString(y)
		} else {
			year = "18" + js.NumberToString(y)
		}
	}
	year = padYear(year)
	if !isDateFormat(year+"/"+sl(tin, 2, 4)+"/"+sl(tin, 0, 2), "YYYY/MM/DD") {
		return false
	}
	digits := digitsOf(tin)
	checksum, weight := 0.0, 4.0
	for i := 0; i < 9; i++ {
		checksum += at(digits, i) * weight
		weight--
		if weight == 1 {
			weight = 7
		}
	}
	checksum = math.Mod(checksum, 11)
	if checksum == 1 {
		return false
	}
	if checksum == 0 {
		return at(digits, 9) == 0
	}
	return at(digits, 9) == 11-checksum
}

func elCyCheck(tin string) bool {
	digits := digitsOf(sl(tin, 0, 8))
	checksum := 0.0
	for i := 1; i < len(digits); i += 2 {
		checksum += digits[i]
	}
	for i := 0; i < len(digits); i += 2 {
		if digits[i] < 2 {
			checksum += 1 - digits[i]
		} else {
			checksum += 2*(digits[i]-2) + 5
			if digits[i] > 4 {
				checksum += 2
			}
		}
	}
	return fromCharCode(math.Mod(checksum, 26)+65) == js.CharAt(tin, 8)
}

// fromCharCode implements String.fromCharCode for one number.
func fromCharCode(n float64) string {
	return js.FromU16([]uint16{uint16(js.ToInt32(n))})
}

func elGrCheck(tin string) bool {
	digits := digitsOf(tin)
	checksum := 0.0
	for i := 0; i < 8; i++ {
		checksum += at(digits, i) * math.Pow(2, float64(8-i))
	}
	return math.Mod(math.Mod(checksum, 11), 10) == at(digits, 8)
}

func enIeCheck(tin string) bool {
	u := js.ToU16(tin)
	first := digitsOf(sl(tin, 0, 7))
	checksum := reverseMultiplyAndSum(first, 8)
	if len(u) == 9 && unit(u, 8) != "W" {
		checksum += (float64(u[8]) - 64) * 9
	}
	checksum = math.Mod(checksum, 23)
	if checksum == 0 {
		return js.ToUpperCase(unit(u, 7)) == "W"
	}
	return js.ToUpperCase(unit(u, 7)) == fromCharCode(64+checksum)
}

var usCampusPrefixes = stringSet("10", "12", "60", "67", "50", "53", "01", "02", "03", "04", "05", "06", "11", "13",
	"14", "16", "21", "22", "23", "25", "34", "51", "52", "54", "55", "56", "57", "58", "59", "65", "30", "32", "35",
	"36", "37", "38", "61", "15", "24", "20", "26", "27", "45", "46", "47", "40", "44", "94", "95", "80", "90", "33",
	"39", "41", "42", "43", "48", "62", "63", "64", "66", "68", "71", "72", "73", "74", "75", "76", "77", "81", "82",
	"83", "84", "85", "86", "87", "88", "91", "92", "93", "98", "99", "31")

func enUsCheck(tin string) bool { return usCampusPrefixes[sl(tin, 0, 2)] }

func esArCheck(tin string) bool {
	chars := js.Split16(tin, "")
	digit := pi(chars[len(chars)-1])
	chars = chars[:len(chars)-1]
	accum := 0.0
	for i := 0; i < len(chars); i++ {
		v := math.NaN()
		if j := 9 - i; j >= 0 && j < len(chars) {
			v = js.StringToNumber(chars[j])
		}
		accum += v * float64(2+i%6)
	}
	verif := 11 - math.Mod(accum, 11)
	switch verif {
	case 11:
		verif = 0
	case 10:
		verif = 9
	}
	return digit == verif
}

func esEsCheck(tin string) bool {
	chars := js.Split16(js.ToUpperCase(tin), "")
	if math.IsNaN(pi(chars[0])) && len(chars) > 1 {
		lead := "0"
		switch chars[0] {
		case "Y":
			lead = "1"
		case "Z":
			lead = "2"
		}
		chars[0] = lead
	} else {
		for len(chars) < 9 {
			chars = append([]string{"0"}, chars...)
		}
	}
	joined := strings.Join(chars, "")
	checksum := math.Mod(pi(sl(joined, 0, 8)), 23)
	lookup := "TRWAGMYFPDXBNJZSQVHLCKE"
	if math.IsNaN(checksum) {
		return false
	}
	return js.CharAt(joined, 8) == string(lookup[int(checksum)])
}

var taxIDChecks = map[string]func(string) bool{
	"bg-BG": bgBgCheck, "cs-CZ": csCzCheck, "de-AT": luhnCheck, "de-DE": deDeCheck, "dk-DK": dkDkCheck,
	"el-CY": elCyCheck, "el-GR": elGrCheck, "en-CA": isCanadianSIN, "en-IE": enIeCheck, "en-US": enUsCheck,
	"es-AR": esArCheck, "es-ES": esEsCheck, "et-EE": etEeCheck, "fi-FI": fiFiCheck, "fr-BE": frBeCheck,
	"fr-FR": frFrCheck, "fr-LU": frLuCheck, "hr-HR": iso7064Check, "hu-HU": huHuCheck, "it-IT": itItCheck,
	"lv-LV": lvLvCheck, "mt-MT": mtMtCheck, "nl-NL": nlNlCheck, "pl-PL": plPlCheck, "pt-BR": ptBrCheck,
	"pt-PT": ptPtCheck, "ro-RO": roRoCheck, "sk-SK": skSkCheck, "sl-SI": slSiCheck, "sv-SE": svSeCheck,
	"uk-UA": ukUaCheck, "lb-LU": frLuCheck, "lt-LT": etEeCheck, "nl-BE": frBeCheck, "fr-CA": isCanadianSIN,
}

var reAllSymbols = re(`[-\\\/!@#$%\^&\*\(\)\+\=\[\]]+`, "g")

var taxIDSanitizers = map[string]*jsre.Regexp{
	"de-AT": reAllSymbols, "de-DE": re(`[\/\\]`, "g"), "fr-BE": reAllSymbols, "nl-BE": reAllSymbols,
}

func isTaxID(a []js.Value) js.Value {
	s := assertString(arg(a, 0))
	locale := arg(a, 1)
	if locale.IsUndefined() {
		locale = js.Str("en-US")
	}
	key := js.ToPropertyKey(locale)
	format, ok := lookupPattern(taxIDFormats, key)
	if !ok {
		if js.IsObjectProtoKey(key) {
			js.Throw("TypeError", "taxIdFormat[locale].test is not a function")
		}
		js.Throw("Error", "Invalid locale '"+js.ToString(locale)+"'")
	}
	if r, ok := taxIDSanitizers[key]; ok {
		s = replace(s, r, "")
	}
	if !format.re.Test(s) {
		return js.Bool(false)
	}
	if check, ok := taxIDChecks[key]; ok {
		return js.Bool(check(s))
	}
	return js.Bool(true)
}
