package validator

import (
	"math"
	"strings"

	"github.com/validatorjs/validator.js/js"
)

func etEeCheck(tin string) bool {
	fullYear := sl(tin, 1, 3)
	switch sl(tin, 0, 1) {
	case "1", "2":
		fullYear = "18" + fullYear
	case "3", "4":
		fullYear = "19" + fullYear
	default:
		fullYear = "20" + fullYear
	}
	if !isDateFormat(fullYear+"/"+sl(tin, 3, 5)+"/"+sl(tin, 5, 7), "YYYY/MM/DD") {
		return false
	}
	digits := digitsOf(tin)
	sum := func(weight float64) float64 {
		checksum := 0.0
		for i := 0; i < 10; i++ {
			checksum += at(digits, i) * weight
			weight++
			if weight == 10 {
				weight = 1
			}
		}
		return checksum
	}
	checksum := sum(1)
	if math.Mod(checksum, 11) == 10 {
		checksum = sum(3)
		if math.Mod(checksum, 11) == 10 {
			return at(digits, 10) == 0
		}
	}
	return math.Mod(checksum, 11) == at(digits, 10)
}

func fiFiCheck(tin string) bool {
	fullYear := sl(tin, 4, 6)
	switch sl(tin, 6, 7) {
	case "+":
		fullYear = "18" + fullYear
	case "-":
		fullYear = "19" + fullYear
	default:
		fullYear = "20" + fullYear
	}
	if !isDateFormat(fullYear+"/"+sl(tin, 2, 4)+"/"+sl(tin, 0, 2), "YYYY/MM/DD") {
		return false
	}
	checksum := math.Mod(pi(sl(tin, 0, 6)+sl(tin, 7, 10)), 31)
	if checksum < 10 {
		return checksum == pi(from(tin, 10))
	}
	letters := "ABCDEFHJKLMNPRSTUVWXY"
	checksum -= 10
	return int(checksum) < len(letters) && string(letters[int(checksum)]) == from(tin, 10)
}

func frBeCheck(tin string) bool {
	if sl(tin, 2, 4) != "00" || sl(tin, 4, 6) != "00" {
		if !isDateFormat(sl(tin, 0, 2)+"/"+sl(tin, 2, 4)+"/"+sl(tin, 4, 6), "YY/MM/DD") {
			return false
		}
	}
	checksum := 97 - math.Mod(pi(sl(tin, 0, 9)), 97)
	checkDigits := pi(sl(tin, 9, 11))
	if checksum != checkDigits {
		checksum = 97 - math.Mod(pi("2"+sl(tin, 0, 9)), 97)
		if checksum != checkDigits {
			return false
		}
	}
	return true
}

func frFrCheck(tin string) bool {
	tin = replace(tin, reSpaces, "")
	return math.Mod(pi(sl(tin, 0, 10)), 511) == pi(sl(tin, 10, 13))
}

func frLuCheck(tin string) bool {
	if !isDateFormat(sl(tin, 0, 4)+"/"+sl(tin, 4, 6)+"/"+sl(tin, 6, 8), "YYYY/MM/DD") {
		return false
	}
	if !luhnCheck(sl(tin, 0, 12)) {
		return false
	}
	u := js.ToU16(tin)
	last := "undefined"
	if len(u) > 12 {
		last = unit(u, 12)
	}
	return verhoeffCheck(sl(tin, 0, 11) + last)
}

func huHuCheck(tin string) bool {
	digits := digitsOf(tin)
	checksum := 8.0
	for i := 1; i < 9; i++ {
		checksum += at(digits, i) * float64(i+1)
	}
	return math.Mod(checksum, 11) == at(digits, 9)
}

func itItNameCheck(name []string) bool {
	vowel, x := false, false
	isVowel := func(s string) bool { return strings.ContainsAny(s, "AEIOU") }
	for i := 0; i < 3; i++ {
		switch {
		case !vowel && isVowel(name[i]):
			vowel = true
		case !x && vowel && name[i] == "X":
			x = true
		case i > 0:
			if vowel && !x && !isVowel(name[i]) {
				return false
			}
			if x && !strings.Contains(name[i], "X") {
				return false
			}
		}
	}
	return true
}

func itItCheck(tin string) bool {
	chars := js.Split16(js.ToUpperCase(tin), "")
	if !itItNameCheck(chars[0:3]) || !itItNameCheck(chars[3:6]) {
		return false
	}
	numberReplace := map[string]string{"L": "0", "M": "1", "N": "2", "P": "3", "Q": "4", "R": "5", "S": "6", "T": "7",
		"U": "8", "V": "9"}
	for _, i := range []int{6, 7, 9, 10, 12, 13, 14} {
		if r, ok := numberReplace[chars[i]]; ok {
			chars[i] = r
		}
	}
	monthReplace := map[string]string{"A": "01", "B": "02", "C": "03", "D": "04", "E": "05", "H": "06", "L": "07",
		"M": "08", "P": "09", "R": "10", "S": "11", "T": "12"}
	month, ok := monthReplace[chars[8]]
	if !ok {
		month = "undefined"
	}
	day := pi(chars[9] + chars[10])
	if day > 40 {
		day -= 40
	}
	if !isDateFormat(chars[6]+chars[7]+"/"+month+"/"+twoDigits(day), "YY/MM/DD") {
		return false
	}
	checksum := 0.0
	for i := 1; i < len(chars)-1; i += 2 {
		v := pi(chars[i])
		if math.IsNaN(v) {
			v = float64(js.CharCodeAt(chars[i], 0)) - 65
		}
		checksum += v
	}
	oddConvert := map[string]float64{"A": 1, "B": 0, "C": 5, "D": 7, "E": 9, "F": 13, "G": 15, "H": 17, "I": 19,
		"J": 21, "K": 2, "L": 4, "M": 18, "N": 20, "O": 11, "P": 3, "Q": 6, "R": 8, "S": 12, "T": 14, "U": 16,
		"V": 10, "W": 22, "X": 25, "Y": 24, "Z": 23, "0": 1, "1": 0}
	for i := 0; i < len(chars)-1; i += 2 {
		v, ok := oddConvert[chars[i]]
		if !ok {
			m := pi(chars[i])
			v = 2*m + 1
			if m > 4 {
				v += 2
			}
		}
		checksum += v
	}
	return fromCharCode(65+math.Mod(checksum, 26)) == chars[15]
}

func lvLvCheck(tin string) bool {
	tin = replace(tin, reNonWordFirst, "")
	day := sl(tin, 0, 2)
	if day == "32" {
		return true
	}
	u := js.ToU16(tin)
	if sl(tin, 2, 4) != "00" {
		fullYear := sl(tin, 4, 6)
		switch unit(u, 6) {
		case "0":
			fullYear = "18" + fullYear
		case "1":
			fullYear = "19" + fullYear
		default:
			fullYear = "20" + fullYear
		}
		if !isDateFormat(fullYear+"/"+sl(tin, 2, 4)+"/"+day, "YYYY/MM/DD") {
			return false
		}
	}
	checksum := 1101.0
	lookup := []float64{1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	for i := 0; i < len(u)-1; i++ {
		m := math.NaN()
		if i < len(lookup) {
			m = lookup[i]
		}
		checksum -= pi(unit(u, i)) * m
	}
	return pi(unit(u, 10)) == math.Mod(math.Mod(checksum, 11), 10)
}

func mtMtCheck(tin string) bool {
	u := js.ToU16(tin)
	if len(u) == 9 {
		return true
	}
	chars := js.Split16(js.ToUpperCase(tin), "")
	for len(chars) < 8 {
		chars = append([]string{"0"}, chars...)
	}
	switch unit(u, 7) {
	case "A", "P":
		if pi(chars[6]) == 0 {
			return false
		}
	default:
		joined := strings.Join(chars, "")
		first := pi(sl(joined, 0, 5))
		if first > 32000 {
			return false
		}
		if first == pi(sl(joined, 5, 7)) {
			return false
		}
	}
	return true
}

func nlNlCheck(tin string) bool {
	return math.Mod(reverseMultiplyAndSum(digitsOf(sl(tin, 0, 8)), 9), 11) == pi(js.CharAt(tin, 8))
}
