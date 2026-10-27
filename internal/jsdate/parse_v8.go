package jsdate

import "math"

const maxTimeBeforeUTCInMs = maxTimeInMs + 864000000

// Parse implements Date.parse for a string given as UTF-16 code units, with UTC as the local time zone.
func Parse(s []uint16) float64 {
	var out [8]float64
	if !parseFields(s, &out) {
		return math.NaN()
	}
	day := MakeDay(out[0], out[1], out[2])
	t := MakeTime(out[3], out[4], out[5], out[6])
	date := MakeDate(day, t)
	if math.IsNaN(out[7]) {
		if date < -maxTimeBeforeUTCInMs || date > maxTimeBeforeUTCInMs {
			return math.NaN()
		}
	} else {
		date -= out[7] * 1000
		if date < -maxTimeInMs || date > maxTimeInMs {
			return math.NaN()
		}
	}
	return TimeClip(date)
}

func parseFields(s []uint16, out *[8]float64) bool {
	sc := &scanner{s: s}
	sc.next = sc.scan()
	day := &dayComposer{namedMonth: none}
	tm := &timeComposer{hourOffset: none}
	tz := &tzComposer{sign: none, hour: none, minute: none}
	next := parseES5(sc, day, tm, tz)
	if next.kind == tokInvalid {
		return false
	}
	hasReadNumber := day.index > 0
	for tok := next; tok.kind != tokEnd; tok = sc.Next() {
		switch {
		case tok.isNumber():
			hasReadNumber = true
			n := tok.value
			if sc.SkipSymbol(':') {
				if sc.SkipSymbol(':') {
					if !tm.isEmpty() {
						return false
					}
					tm.add(n)
					tm.add(0)
				} else {
					if !tm.add(n) {
						return false
					}
					if sc.Peek().isSymbol('.') {
						sc.Next()
					}
				}
			} else if sc.SkipSymbol('.') && tm.isExpecting(n) {
				tm.add(n)
				if !sc.Peek().isNumber() {
					return false
				}
				ms := readMilliseconds(sc.Next())
				if ms < 0 {
					return false
				}
				tm.addFinal(ms)
			} else if tz.isExpecting(n) {
				tz.minute = n
			} else if tm.isExpecting(n) {
				tm.addFinal(n)
				p := sc.Peek()
				if p.kind != tokEnd && p.kind != tokWhiteSpace && !p.isKeywordZ() && !p.isAsciiSign() {
					return false
				}
			} else {
				if !day.add(n) {
					return false
				}
				sc.SkipSymbol('-')
			}
		case tok.kind == tokKeyword:
			if tok.kwType == kwAmPm && !tm.isEmpty() {
				tm.hourOffset = tok.value
			} else if tok.kwType == kwMonthName {
				day.namedMonth = tok.value
				sc.SkipSymbol('-')
			} else if tok.kwType == kwTimeZoneName && hasReadNumber {
				tz.set(tok.value)
			} else {
				if hasReadNumber {
					return false
				}
				if sc.Peek().isNumber() {
					return false
				}
			}
		case tok.isAsciiSign() && (tz.isUTC() || !tm.isEmpty()):
			sign := 1
			if tok.sym == '-' {
				sign = -1
			}
			tz.setSign(sign)
			n, length := 0, 0
			if sc.Peek().isNumber() {
				nt := sc.Next()
				length = nt.length
				n = nt.value
			}
			hasReadNumber = true
			switch {
			case sc.Peek().isSymbol(':'):
				tz.hour = n
				tz.minute = none
			case length == 2 || length == 1:
				tz.hour = n
				tz.minute = 0
			case length == 4 || length == 3:
				tz.hour = n / 100
				tz.minute = n % 100
			default:
				return false
			}
		case (tok.isAsciiSign() || tok.isSymbol(')')) && hasReadNumber:
			return false
		}
	}
	return day.write(out) && tm.write(out) && tz.write(out)
}

func parseES5(sc *scanner, day *dayComposer, tm *timeComposer, tz *tzComposer) token {
	if sc.Peek().isAsciiSign() {
		signTok := sc.Next()
		if !sc.Peek().isFixedLengthNumber(6) {
			return signTok
		}
		sign := 1
		if signTok.sym == '-' {
			sign = -1
		}
		year := sc.Next().value
		if sign < 0 && year == 0 {
			return signTok
		}
		day.add(sign * year)
	} else if sc.Peek().isFixedLengthNumber(4) {
		day.add(sc.Next().value)
	} else {
		return sc.Next()
	}
	if sc.SkipSymbol('-') {
		if !sc.Peek().isFixedLengthNumber(2) || !isMonth(sc.Peek().value) {
			return sc.Next()
		}
		day.add(sc.Next().value)
		if sc.SkipSymbol('-') {
			if !sc.Peek().isFixedLengthNumber(2) || !isDay(sc.Peek().value) {
				return sc.Next()
			}
			day.add(sc.Next().value)
		}
	}
	if !sc.Peek().isKeywordType(kwTimeSeparator) {
		if sc.Peek().kind != tokEnd {
			return sc.Next()
		}
	} else {
		sc.Next()
		if !sc.Peek().isFixedLengthNumber(2) || !between(sc.Peek().value, 0, 24) {
			return token{kind: tokInvalid}
		}
		hourIs24 := sc.Peek().value == 24
		tm.add(sc.Next().value)
		if !sc.SkipSymbol(':') {
			return token{kind: tokInvalid}
		}
		if !sc.Peek().isFixedLengthNumber(2) || !isMinute(sc.Peek().value) || (hourIs24 && sc.Peek().value > 0) {
			return token{kind: tokInvalid}
		}
		tm.add(sc.Next().value)
		if sc.SkipSymbol(':') {
			if !sc.Peek().isFixedLengthNumber(2) || !isSecond(sc.Peek().value) ||
				(hourIs24 && sc.Peek().value > 0) {
				return token{kind: tokInvalid}
			}
			tm.add(sc.Next().value)
			if sc.SkipSymbol('.') {
				if !sc.Peek().isNumber() || (hourIs24 && sc.Peek().value > 0) {
					return token{kind: tokInvalid}
				}
				tm.add(readMilliseconds(sc.Next()))
			}
		}
		if sc.Peek().isKeywordZ() {
			sc.Next()
			tz.set(0)
		} else if sc.Peek().isSymbol('+') || sc.Peek().isSymbol('-') {
			sign := 1
			if sc.Next().sym == '-' {
				sign = -1
			}
			tz.setSign(sign)
			if sc.Peek().isFixedLengthNumber(4) {
				hm := sc.Next().value
				h, m := hm/100, hm%100
				if !isHour(h) || !isMinute(m) {
					return token{kind: tokInvalid}
				}
				tz.hour, tz.minute = h, m
			} else {
				if !sc.Peek().isFixedLengthNumber(2) || !isHour(sc.Peek().value) {
					return token{kind: tokInvalid}
				}
				tz.hour = sc.Next().value
				if !sc.SkipSymbol(':') {
					return token{kind: tokInvalid}
				}
				if !sc.Peek().isFixedLengthNumber(2) || !isMinute(sc.Peek().value) {
					return token{kind: tokInvalid}
				}
				tz.minute = sc.Next().value
			}
		}
		if sc.Peek().kind != tokEnd {
			return token{kind: tokInvalid}
		}
	}
	if tz.isEmpty() && tm.isEmpty() {
		tz.set(0)
	}
	day.iso = true
	return token{kind: tokEnd}
}
