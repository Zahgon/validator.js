package validator

import (
	"math"
	"time"

	"github.com/validatorjs/validator.js/js"
)

func plPlCheck(tin string) bool {
	u := js.ToU16(tin)
	if len(u) == 10 {
		lookup := []float64{6, 5, 7, 2, 3, 4, 5, 6, 7}
		checksum := 0.0
		for i, m := range lookup {
			checksum += pi(unit(u, i)) * m
		}
		checksum = math.Mod(checksum, 11)
		if checksum == 10 {
			return false
		}
		return checksum == pi(unit(u, 9))
	}
	fullYear := sl(tin, 0, 2)
	month := pi(sl(tin, 2, 4))
	switch {
	case month > 80:
		fullYear, month = "18"+fullYear, month-80
	case month > 60:
		fullYear, month = "22"+fullYear, month-60
	case month > 40:
		fullYear, month = "21"+fullYear, month-40
	case month > 20:
		fullYear, month = "20"+fullYear, month-20
	default:
		fullYear = "19" + fullYear
	}
	if !isDateFormat(fullYear+"/"+twoDigits(month)+"/"+sl(tin, 4, 6), "YYYY/MM/DD") {
		return false
	}
	checksum, multiplier := 0.0, 1.0
	for i := 0; i < len(u)-1; i++ {
		checksum += math.Mod(pi(unit(u, i))*multiplier, 10)
		multiplier += 2
		if multiplier > 10 {
			multiplier = 1
		} else if multiplier == 5 {
			multiplier += 2
		}
	}
	checksum = math.Mod(10-math.Mod(checksum, 10), 10)
	return checksum == pi(unit(u, 10))
}

var reSameChars = re(`^(.)\1+$`, "")

func validateCnpj(cnpj string) bool {
	identifiers := js.ToUpperCase(js.Substring(cnpj, 0, 12))
	checkDigits := js.Substring(cnpj, 12, js.Len16(cnpj))
	if test(reSameChars, js.ToUpperCase(cnpj)) {
		return false
	}
	value := func(i int) float64 {
		c := js.CharCodeAt(identifiers, i)
		if c < 0 {
			return math.NaN()
		}
		return float64(c - 48)
	}
	sum, weight := 0.0, 5.0
	for i := 0; i < 12; i++ {
		sum += value(i) * weight
		if weight == 2 {
			weight = 9
		} else {
			weight--
		}
	}
	remainder := math.Mod(sum, 11)
	first := 11 - remainder
	if remainder < 2 {
		first = 0
	}
	if first != pi(js.CharAt(checkDigits, 0)) {
		return false
	}
	sum, weight = 0, 6
	for i := 0; i < 12; i++ {
		sum += value(i) * weight
		if weight == 2 {
			weight = 9
		} else {
			weight--
		}
	}
	sum += first * 2
	remainder = math.Mod(sum, 11)
	second := 11 - remainder
	if remainder < 2 {
		second = 0
	}
	return second == pi(js.CharAt(checkDigits, 1))
}

var reCPFSeparators = re(`[.\-/]`, "g")

func ptBrCheck(tin string) bool {
	tin = replace(tin, reCPFSeparators, "")
	if js.Len16(tin) != 11 {
		return validateCnpj(tin)
	}
	for _, d := range "0123456789" {
		if tin == string(d)+string(d)+string(d)+string(d)+string(d)+string(d)+string(d)+string(d)+string(d)+
			string(d)+string(d) {
			return false
		}
	}
	check := func(n int) bool {
		sum := 0.0
		for i := 1; i <= n; i++ {
			sum += pi(js.Substring(tin, i-1, i)) * float64(n+2-i)
		}
		remainder := math.Mod(sum*10, 11)
		if remainder == 10 {
			remainder = 0
		}
		return remainder == pi(js.Substring(tin, n, n+1))
	}
	return check(9) && check(10)
}

func ptPtCheck(tin string) bool {
	checksum := 11 - math.Mod(reverseMultiplyAndSum(digitsOf(sl(tin, 0, 8)), 9), 11)
	last := pi(js.CharAt(tin, 8))
	if checksum > 9 {
		return last == 0
	}
	return checksum == last
}

func roRoCheck(tin string) bool {
	if sl(tin, 0, 4) == "9000" {
		return true
	}
	fullYear := sl(tin, 1, 3)
	switch js.CharAt(tin, 0) {
	case "1", "2":
		fullYear = "19" + fullYear
	case "3", "4":
		fullYear = "18" + fullYear
	case "5", "6":
		fullYear = "20" + fullYear
	}
	date := fullYear + "/" + sl(tin, 3, 5) + "/" + sl(tin, 5, 7)
	if js.Len16(date) == 8 {
		if !isDateFormat(date, "YY/MM/DD") {
			return false
		}
	} else if !isDateFormat(date, "YYYY/MM/DD") {
		return false
	}
	digits := digitsOf(tin)
	multipliers := []float64{2, 7, 9, 1, 4, 6, 3, 5, 8, 2, 7, 9}
	checksum := 0.0
	for i, m := range multipliers {
		checksum += at(digits, i) * m
	}
	if math.Mod(checksum, 11) == 10 {
		return at(digits, 12) == 1
	}
	return at(digits, 12) == math.Mod(checksum, 11)
}

func skSkCheck(tin string) bool {
	if js.Len16(tin) != 9 {
		return true
	}
	tin = replace(tin, reNonWordFirst, "")
	if from(tin, 6) == "000" {
		return false
	}
	y := pi(sl(tin, 0, 2))
	if y > 53 {
		return false
	}
	fullYear := "19" + js.NumberToString(y)
	if y < 10 {
		fullYear = "190" + js.NumberToString(y)
	}
	month := pi(sl(tin, 2, 4))
	if month > 50 {
		month -= 50
	}
	return isDateFormat(fullYear+"/"+twoDigits(month)+"/"+sl(tin, 4, 6), "YYYY/MM/DD")
}

func slSiCheck(tin string) bool {
	checksum := 11 - math.Mod(reverseMultiplyAndSum(digitsOf(sl(tin, 0, 7)), 8), 11)
	last := pi(js.CharAt(tin, 7))
	if checksum == 10 {
		return last == 0
	}
	return checksum == last
}

func svSeCheck(tin string) bool {
	n := js.Len16(tin)
	tinCopy := tin
	if n > 11 {
		tinCopy = from(tinCopy, 2)
	}
	month := sl(tinCopy, 2, 4)
	day := pi(sl(tinCopy, 4, 6))
	var fullYear string
	if n > 11 {
		fullYear = sl(tin, 0, 4)
	} else {
		fullYear = sl(tin, 0, 2)
		if n == 11 && day < 60 {
			currentYear := js.NumberToString(float64(time.Now().UTC().Year()))
			century := pi(sl(currentYear, 0, 2))
			current := pi(currentYear)
			if js.CharAt(tin, 6) == "-" {
				if pi(js.NumberToString(century)+fullYear) > current {
					fullYear = js.NumberToString(century-1) + fullYear
				} else {
					fullYear = js.NumberToString(century) + fullYear
				}
			} else {
				fullYear = js.NumberToString(century-1) + fullYear
				if current-pi(fullYear) < 100 {
					return false
				}
			}
		}
	}
	if day > 60 {
		day -= 60
	}
	date := fullYear + "/" + month + "/" + twoDigits(day)
	if js.Len16(date) == 8 {
		if !isDateFormat(date, "YY/MM/DD") {
			return false
		}
	} else if !isDateFormat(date, "YYYY/MM/DD") {
		return false
	}
	return luhnCheck(replace(tin, reNonWordFirst, ""))
}

func ukUaCheck(tin string) bool {
	digits := digitsOf(tin)
	multipliers := []float64{-1, 5, 7, 9, 4, 6, 10, 5, 7}
	checksum := 0.0
	for i, m := range multipliers {
		checksum += at(digits, i) * m
	}
	if math.Mod(checksum, 11) == 10 {
		return at(digits, 9) == 0
	}
	return at(digits, 9) == math.Mod(checksum, 11)
}
