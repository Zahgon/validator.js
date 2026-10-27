package validator

import (
	"math"
	"strings"
	"sync"

	"github.com/validatorjs/validator.js/internal/jsre"
	"github.com/validatorjs/validator.js/js"
)

// re compiles a regular expression literal of the original source.
func re(pattern, flags string) *jsre.Regexp { return jsre.MustCompile(pattern, flags) }

var compiled sync.Map

// compiledOf returns the engine program of a RegExp object.
func compiledOf(o *js.Object) *jsre.Regexp {
	key := o.Flags + "/" + o.Source
	if r, ok := compiled.Load(key); ok {
		return r.(*jsre.Regexp)
	}
	r := jsre.MustCompile(o.Source, o.Flags)
	compiled.Store(key, r)
	return r
}

// newRegExp implements new RegExp(pattern, flags) for the values a validator passes, throwing V8's SyntaxError.
func newRegExp(pattern js.Value, flags js.Value) js.Value {
	var src string
	if pattern.IsRegExp() {
		src = pattern.Obj().Source
		if flags.IsUndefined() {
			flags = js.Str(pattern.Obj().Flags)
		}
	} else if pattern.IsUndefined() {
		src = ""
	} else {
		src = js.ToString(pattern)
	}
	f := ""
	if !flags.IsUndefined() {
		f = js.ToString(flags)
	}
	r, err := jsre.Compile(src, f)
	if err != nil {
		js.Throw("SyntaxError", err.Message)
	}
	return js.NewRegExpObject(escapeSource(src), r.Flags())
}

// escapeSource returns the source text of a RegExp object: "(?:)" for an empty pattern, with unescaped slashes
// and line terminators escaped.
func escapeSource(src string) string {
	if src == "" {
		return "(?:)"
	}
	u := js.ToU16(src)
	var b []uint16
	inClass := false
	for i := 0; i < len(u); i++ {
		c := u[i]
		switch {
		case c == '\\' && i+1 < len(u):
			b = append(b, c, u[i+1])
			i++
			continue
		case c == '[':
			inClass = true
		case c == ']':
			inClass = false
		case c == '/' && !inClass:
			b = append(b, '\\', '/')
			continue
		case c == '\n':
			b = append(b, '\\', 'n')
			continue
		case c == '\r':
			b = append(b, '\\', 'r')
			continue
		case c == 0x2028:
			b = append(b, []uint16(js.ToU16("\\u2028"))...)
			continue
		case c == 0x2029:
			b = append(b, []uint16(js.ToU16("\\u2029"))...)
			continue
		}
		b = append(b, c)
	}
	return js.FromU16(b)
}

// regexpExec implements RegExpBuiltinExec for a RegExp object, including its lastIndex state.
func regexpExec(rx js.Value, s string) []int {
	o := rx.Obj()
	r := compiledOf(o)
	in := js.ToU16(s)
	li := js.ToIntegerOrInfinity(o.LastIndex)
	if li < 0 {
		li = 0
	}
	stateful := r.Global() || r.Sticky()
	if !stateful {
		li = 0
	}
	if li > float64(len(in)) {
		if stateful {
			o.LastIndex = js.Num(0)
		}
		return nil
	}
	var caps []int
	if r.Sticky() {
		caps = r.MatchAt(in, int(li))
	} else {
		caps = r.Search(in, int(li))
	}
	if caps == nil {
		if stateful {
			o.LastIndex = js.Num(0)
		}
		return nil
	}
	if stateful {
		o.LastIndex = js.Num(float64(caps[1]))
	}
	return caps
}

// regexpTest implements RegExp.prototype.test on a RegExp object with value s converted to a string.
func regexpTest(rx js.Value, s js.Value) bool {
	return regexpExec(rx, js.ToString(s)) != nil
}

// test runs a literal pattern without global state.
func test(r *jsre.Regexp, s string) bool { return r.Test(s) }

type match struct {
	index int
	caps  []*string
	text  string
}

func capsOf(in []uint16, c []int) match {
	m := match{index: c[0], text: js.FromU16(in[c[0]:c[1]])}
	for i := 2; i < len(c); i += 2 {
		if c[i] < 0 {
			m.caps = append(m.caps, nil)
			continue
		}
		s := js.FromU16(in[c[i]:c[i+1]])
		m.caps = append(m.caps, &s)
	}
	return m
}

// allMatches returns the matches String.prototype.replace and match visit: every match for a global pattern,
// the first otherwise.
func allMatches(r *jsre.Regexp, s string) []match {
	in := js.ToU16(s)
	var out []match
	pos := 0
	for pos <= len(in) {
		var c []int
		if r.Sticky() {
			c = r.MatchAt(in, pos)
		} else {
			c = r.Search(in, pos)
		}
		if c == nil {
			break
		}
		out = append(out, capsOf(in, c))
		if !r.Global() {
			break
		}
		if c[1] == c[0] {
			pos = r.AdvanceIndex(in, c[1])
		} else {
			pos = c[1]
		}
	}
	return out
}

// replace implements String.prototype.replace with a literal pattern and a replacement text.
func replace(s string, r *jsre.Regexp, repl string) string {
	return replaceFunc(s, r, func(m match) string {
		return js.ExpandReplacement(repl, m.text, s, m.index, m.caps, nil)
	})
}

// replaceFunc implements String.prototype.replace with a literal pattern and a replacer function.
func replaceFunc(s string, r *jsre.Regexp, f func(m match) string) string {
	ms := allMatches(r, s)
	if len(ms) == 0 {
		return s
	}
	in := js.ToU16(s)
	var b strings.Builder
	last := 0
	for _, m := range ms {
		b.WriteString(js.FromU16(in[last:m.index]))
		b.WriteString(f(m))
		last = m.index + js.Len16(m.text)
	}
	b.WriteString(js.FromU16(in[last:]))
	return b.String()
}

// matchAll returns the matched texts of a global pattern (String.prototype.match), or nil when there is none.
func matchTexts(s string, r *jsre.Regexp) []string {
	ms := allMatches(r, s)
	if len(ms) == 0 {
		return nil
	}
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.text
	}
	return out
}

// firstMatch returns the first match of a non-global pattern (String.prototype.match), and whether there is one.
func firstMatch(s string, r *jsre.Regexp) (match, bool) {
	ms := allMatches(r, s)
	if len(ms) == 0 {
		return match{}, false
	}
	return ms[0], true
}

// splitRe implements String.prototype.split with a pattern and no limit.
func splitRe(s string, r *jsre.Regexp) []string {
	in := js.ToU16(s)
	if len(in) == 0 {
		if r.MatchAt(in, 0) != nil {
			return []string{}
		}
		return []string{s}
	}
	var out []string
	p, q := 0, 0
	for q < len(in) {
		c := r.MatchAt(in, q)
		if c == nil || c[1] == p {
			q = r.AdvanceIndex(in, q)
			continue
		}
		e := c[1]
		if e > len(in) {
			e = len(in)
		}
		out = append(out, js.FromU16(in[p:q]))
		for i := 2; i < len(c); i += 2 {
			if c[i] < 0 {
				out = append(out, "")
			} else {
				out = append(out, js.FromU16(in[c[i]:c[i+1]]))
			}
		}
		p = e
		q = p
	}
	return append(out, js.FromU16(in[p:]))
}

// toLength implements ToLength for a numeric value.
func toLength(n float64) float64 {
	if math.IsNaN(n) || n <= 0 {
		return 0
	}
	return math.Min(math.Floor(n), 9007199254740991)
}

// NewRegExp builds a RegExp object like new RegExp(pattern, flags) and returns the SyntaxError that throws.
func NewRegExp(pattern, flags js.Value) (rx js.Value, err error) {
	if e := js.Catch(func() { rx = newRegExp(pattern, flags) }); e != nil {
		return js.Undefined, e
	}
	return rx, nil
}
