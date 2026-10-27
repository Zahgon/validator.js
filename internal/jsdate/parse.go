package jsdate

import "math"

const none = math.MaxInt32

type tokKind uint8

const (
	tokInvalid tokKind = iota
	tokUnknown
	tokNumber
	tokSymbol
	tokWhiteSpace
	tokKeyword
	tokEnd
)

const (
	kwInvalid = iota
	kwMonthName
	kwAmPm
	kwTimeZoneName
	kwTimeSeparator
)

type token struct {
	kind   tokKind
	value  int
	length int
	kwType int
	sym    uint16
}

func (t token) isNumber() bool                 { return t.kind == tokNumber }
func (t token) isFixedLengthNumber(n int) bool { return t.kind == tokNumber && t.length == n }
func (t token) isSymbol(c uint16) bool         { return t.kind == tokSymbol && t.sym == c }
func (t token) isAsciiSign() bool              { return t.kind == tokSymbol && (t.sym == '+' || t.sym == '-') }
func (t token) isKeywordType(k int) bool       { return t.kind == tokKeyword && t.kwType == k }
func (t token) isKeywordZ() bool {
	return t.kind == tokKeyword && t.kwType == kwTimeZoneName && t.length == 1 && t.value == 0
}

type keyword struct {
	prefix [3]uint32
	kind   int
	value  int
}

var keywords = []keyword{
	{[3]uint32{'j', 'a', 'n'}, kwMonthName, 1}, {[3]uint32{'f', 'e', 'b'}, kwMonthName, 2},
	{[3]uint32{'m', 'a', 'r'}, kwMonthName, 3}, {[3]uint32{'a', 'p', 'r'}, kwMonthName, 4},
	{[3]uint32{'m', 'a', 'y'}, kwMonthName, 5}, {[3]uint32{'j', 'u', 'n'}, kwMonthName, 6},
	{[3]uint32{'j', 'u', 'l'}, kwMonthName, 7}, {[3]uint32{'a', 'u', 'g'}, kwMonthName, 8},
	{[3]uint32{'s', 'e', 'p'}, kwMonthName, 9}, {[3]uint32{'o', 'c', 't'}, kwMonthName, 10},
	{[3]uint32{'n', 'o', 'v'}, kwMonthName, 11}, {[3]uint32{'d', 'e', 'c'}, kwMonthName, 12},
	{[3]uint32{'a', 'm', 0}, kwAmPm, 0}, {[3]uint32{'p', 'm', 0}, kwAmPm, 12},
	{[3]uint32{'u', 't', 0}, kwTimeZoneName, 0}, {[3]uint32{'u', 't', 'c'}, kwTimeZoneName, 0},
	{[3]uint32{'z', 0, 0}, kwTimeZoneName, 0}, {[3]uint32{'g', 'm', 't'}, kwTimeZoneName, 0},
	{[3]uint32{'c', 'd', 't'}, kwTimeZoneName, -5}, {[3]uint32{'c', 's', 't'}, kwTimeZoneName, -6},
	{[3]uint32{'e', 'd', 't'}, kwTimeZoneName, -4}, {[3]uint32{'e', 's', 't'}, kwTimeZoneName, -5},
	{[3]uint32{'m', 'd', 't'}, kwTimeZoneName, -6}, {[3]uint32{'m', 's', 't'}, kwTimeZoneName, -7},
	{[3]uint32{'p', 'd', 't'}, kwTimeZoneName, -7}, {[3]uint32{'p', 's', 't'}, kwTimeZoneName, -8},
	{[3]uint32{'t', 0, 0}, kwTimeSeparator, 0},
}

func lookupKeyword(pre [3]uint32, length int) (int, int) {
	for _, k := range keywords {
		if k.prefix == pre && (length <= 3 || k.kind == kwMonthName) {
			return k.kind, k.value
		}
	}
	return kwInvalid, 0
}

func isWhiteSpaceOrLT(c uint16) bool {
	switch c {
	case 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x20, 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return c >= 0x2000 && c <= 0x200A
}

// scanner tokenizes a date string the way V8's DateStringTokenizer does.
type scanner struct {
	s    []uint16
	pos  int
	next token
}

func (r *scanner) ch() uint16 {
	if r.pos < len(r.s) {
		return r.s[r.pos]
	}
	return 0
}

func (r *scanner) scan() token {
	if r.pos >= len(r.s) {
		return token{kind: tokEnd}
	}
	c := r.s[r.pos]
	if c >= '0' && c <= '9' {
		start := r.pos
		n := 0
		for i := 0; r.pos < len(r.s) && r.s[r.pos] >= '0' && r.s[r.pos] <= '9'; i++ {
			if i < 9 {
				n = n*10 + int(r.s[r.pos]-'0')
			}
			r.pos++
		}
		return token{kind: tokNumber, value: n, length: r.pos - start}
	}
	switch c {
	case ':', '-', '+', '.', ')':
		r.pos++
		return token{kind: tokSymbol, sym: c}
	}
	if c >= 'A' && !isWhiteSpaceOrLT(c) {
		var pre [3]uint32
		length := 0
		for r.pos < len(r.s) && r.s[r.pos] >= 'A' && !isWhiteSpaceOrLT(r.s[r.pos]) {
			if length < 3 {
				pre[length] = uint32(r.s[r.pos] | 0x20)
			}
			length++
			r.pos++
		}
		kind, value := lookupKeyword(pre, length)
		return token{kind: tokKeyword, kwType: kind, value: value, length: length}
	}
	if isWhiteSpaceOrLT(c) {
		r.pos++
		return token{kind: tokWhiteSpace, length: 1}
	}
	if c == '(' {
		balance := 0
		for {
			ch := r.ch()
			if ch == ')' {
				balance--
			} else if ch == '(' {
				balance++
			}
			r.pos++
			if !(balance > 0 && r.ch() != 0) {
				break
			}
		}
		return token{kind: tokUnknown}
	}
	r.pos++
	return token{kind: tokUnknown}
}

func (r *scanner) Next() token {
	t := r.next
	r.next = r.scan()
	return t
}

func (r *scanner) Peek() token { return r.next }

func (r *scanner) SkipSymbol(c uint16) bool {
	if r.next.isSymbol(c) {
		r.Next()
		return true
	}
	return false
}

type dayComposer struct {
	comp       [3]int
	index      int
	namedMonth int
	iso        bool
}

func (d *dayComposer) add(n int) bool {
	if d.index < 3 {
		d.comp[d.index] = n
		d.index++
		return true
	}
	return false
}

func between(x, lo, hi int) bool { return x >= lo && x <= hi }
func isMonth(x int) bool         { return between(x, 1, 12) }
func isDay(x int) bool           { return between(x, 1, 31) }
func isHour(x int) bool          { return between(x, 0, 23) }
func isMinute(x int) bool        { return between(x, 0, 59) }
func isSecond(x int) bool        { return between(x, 0, 59) }
func isHour12(x int) bool        { return between(x, 0, 12) }
func isMillisecond(x int) bool   { return between(x, 0, 999) }

func (d *dayComposer) write(out *[8]float64) bool {
	if d.index < 1 {
		return false
	}
	for d.index < 3 {
		d.comp[d.index] = 1
		d.index++
	}
	year, month, day := 0, none, none
	if d.namedMonth == none {
		if d.iso || !isDay(d.comp[0]) {
			year, month, day = d.comp[0], d.comp[1], d.comp[2]
		} else {
			month, day, year = d.comp[0], d.comp[1], d.comp[2]
		}
	} else {
		month = d.namedMonth
		if !isDay(d.comp[0]) {
			year, day = d.comp[0], d.comp[1]
		} else {
			day, year = d.comp[0], d.comp[1]
		}
	}
	if !d.iso {
		if between(year, 0, 49) {
			year += 2000
		} else if between(year, 50, 99) {
			year += 1900
		}
	}
	if year < -(1<<30) || year > (1<<30)-1 || !isMonth(month) || !isDay(day) {
		return false
	}
	out[0], out[1], out[2] = float64(year), float64(month-1), float64(day)
	return true
}

type timeComposer struct {
	comp       [4]int
	index      int
	hourOffset int
}

func (t *timeComposer) isEmpty() bool { return t.index == 0 }
func (t *timeComposer) isExpecting(n int) bool {
	return (t.index == 1 && isMinute(n)) || (t.index == 2 && isSecond(n)) || (t.index == 3 && isMillisecond(n))
}
func (t *timeComposer) add(n int) bool {
	if t.index < 4 {
		t.comp[t.index] = n
		t.index++
		return true
	}
	return false
}
func (t *timeComposer) addFinal(n int) bool {
	if !t.add(n) {
		return false
	}
	for t.index < 4 {
		t.comp[t.index] = 0
		t.index++
	}
	return true
}

func (t *timeComposer) write(out *[8]float64) bool {
	for t.index < 4 {
		t.comp[t.index] = 0
		t.index++
	}
	hour, minute, second, ms := t.comp[0], t.comp[1], t.comp[2], t.comp[3]
	if t.hourOffset != none {
		if !isHour12(hour) {
			return false
		}
		hour %= 12
		hour += t.hourOffset
	}
	if !isHour(hour) || !isMinute(minute) || !isSecond(second) || !isMillisecond(ms) {
		if hour != 24 || minute != 0 || second != 0 || ms != 0 {
			return false
		}
	}
	out[3], out[4], out[5], out[6] = float64(hour), float64(minute), float64(second), float64(ms)
	return true
}

type tzComposer struct{ sign, hour, minute int }

func (z *tzComposer) set(offset int) {
	z.sign = 1
	if offset < 0 {
		z.sign = -1
	}
	z.hour = offset * z.sign
	z.minute = 0
}
func (z *tzComposer) setSign(s int) {
	z.sign = 1
	if s < 0 {
		z.sign = -1
	}
}
func (z *tzComposer) isExpecting(n int) bool {
	return z.hour != none && z.minute == none && isMinute(n)
}
func (z *tzComposer) isUTC() bool   { return z.hour == 0 && z.minute == 0 }
func (z *tzComposer) isEmpty() bool { return z.hour == none }

func (z *tzComposer) write(out *[8]float64) bool {
	if z.sign != none {
		if z.hour == none {
			z.hour = 0
		}
		if z.minute == none {
			z.minute = 0
		}
		total := uint64(uint32(z.hour))*3600 + uint64(uint32(z.minute))*60
		total &= 0xFFFFFFFF
		if total > 1<<31-1 {
			return false
		}
		t := int(total)
		if z.sign < 0 {
			t = -t
		}
		out[7] = float64(t)
	} else {
		out[7] = math.NaN()
	}
	return true
}

func readMilliseconds(t token) int {
	n, length := t.value, t.length
	switch {
	case length == 1:
		n *= 100
	case length == 2:
		n *= 10
	case length > 3:
		if length > 9 {
			length = 9
		}
		factor := 1
		for length > 3 {
			factor *= 10
			length--
		}
		n /= factor
	}
	return n
}
