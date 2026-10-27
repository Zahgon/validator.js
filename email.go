package validator

import (
	"strings"

	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isEmail", func(a []js.Value) js.Value { return js.Bool(isEmail(arg(a, 0), arg(a, 1))) })
	register("isMailtoURI", isMailtoURI)
	register("normalizeEmail", normalizeEmail)
}

func emailDefaults() []option {
	return []option{{"allow_display_name", js.Bool(false)}, {"allow_underscores", js.Bool(false)},
		{"require_display_name", js.Bool(false)}, {"allow_utf8_local_part", js.Bool(true)},
		{"require_tld", js.Bool(true)}, {"blacklisted_chars", js.Str("")}, {"ignore_max_length", js.Bool(false)},
		{"host_blacklist", js.NewArray()}, {"host_whitelist", js.NewArray()}}
}

var reSplitNameAddress = re(`^([^\x00-\x1F\x7F-\x9F\cX]+)<`, "i")
var reEmailUserPart = re("^[a-z\\d!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]+$", "i")
var reGmailUserPart = re(`^[a-z\d]+$`, "")
var reQuotedEmailUser = re(`^([\s\x01-\x08\x0b\x0c\x0e-\x1f\x7f\x21\x23-\x5b\x5d-\x7e]|(\\[\x01-\x09\x0b\x0c\x0d-\x7f]))*$`, "i")
var reEmailUserUtf8Part = re("^[a-z\\d!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~\\u00A1-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFEF]+$", "i")
var reQuotedEmailUserUtf8 = re(`^([\s\x01-\x08\x0b\x0c\x0e-\x1f\x7f\x21\x23-\x5b\x5d-\x7e\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF]|(\\[\x01-\x09\x0b\x0c\x0d-\x7f\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF]))*$`, "i")
var reQuotedDisplayName = re(`^"(.+)"$`, "")
var reDisplayNameIllegal = re(`[\.";<>]`, "")
var reAngleEdges = re(`(^<|>$)`, "g")
var reDot = re(`\.`, "g")

func validateDisplayName(name string) bool {
	unquoted := replace(name, reQuotedDisplayName, "$1")
	if js.Trim(unquoted) == "" {
		return false
	}
	if test(reDisplayNameIllegal, unquoted) {
		if unquoted == name {
			return false
		}
		if len(js.Split16(unquoted, `"`)) != len(js.Split16(unquoted, `\"`)) {
			return false
		}
	}
	return true
}

// byteLengthWithin calls isByteLength(s, {min, max}) with the given bounds; a negative bound is absent.
func byteLengthWithin(s string, min, max int) bool {
	o := js.NewObject()
	if min >= 0 {
		o.Obj().SetProp("min", js.Num(float64(min)))
	}
	if max >= 0 {
		o.Obj().SetProp("max", js.Num(float64(max)))
	}
	return js.Truthy(isByteLength([]js.Value{js.Str(s), o}))
}

func isEmail(str js.Value, options js.Value) bool {
	s := assertString(str)
	options = merge(options, emailDefaults())
	if optTrue(options, "require_display_name") || optTrue(options, "allow_display_name") {
		if m, ok := firstMatch(s, reSplitNameAddress); ok {
			name := *m.caps[0]
			s = replace(js.ReplaceFirst(s, name, ""), reAngleEdges, "")
			if strings.HasSuffix(name, " ") {
				name = js.SliceJS(name, 0, -1)
			}
			if !validateDisplayName(name) {
				return false
			}
		} else if optTrue(options, "require_display_name") {
			return false
		}
	}
	if !optTrue(options, "ignore_max_length") && js.Len16(s) > 254 {
		return false
	}
	parts := js.Split16(s, "@")
	domain := parts[len(parts)-1]
	parts = parts[:len(parts)-1]
	lowerDomain := js.ToLowerCase(domain)
	if bl := opt(options, "host_blacklist"); js.Greater(length(bl), js.Num(0)) && checkHost(js.Str(lowerDomain), bl) {
		return false
	}
	if wl := opt(options, "host_whitelist"); js.Greater(length(wl), js.Num(0)) && !checkHost(js.Str(lowerDomain), wl) {
		return false
	}
	user := strings.Join(parts, "@")
	if optTrue(options, "domain_specific_validation") && (lowerDomain == "gmail.com" || lowerDomain == "googlemail.com") {
		user = js.ToLowerCase(user)
		username := js.Split16(user, "+")[0]
		if !byteLengthWithin(replace(username, reDot, ""), 6, 30) {
			return false
		}
		for _, p := range js.Split16(username, ".") {
			if !test(reGmailUserPart, p) {
				return false
			}
		}
	}
	if v := opt(options, "ignore_max_length"); v.IsBoolean() && !v.Boolean() &&
		(!byteLengthWithin(user, -1, 64) || !byteLengthWithin(domain, -1, 254)) {
		return false
	}
	fqdnOptions := js.NewObject()
	for _, k := range []string{"require_tld", "ignore_max_length", "allow_underscores"} {
		fqdnOptions.Obj().SetProp(k, opt(options, k))
	}
	if !isFQDN(domain, fqdnOptions) {
		if !optTrue(options, "allow_ip_domain") {
			return false
		}
		if !isIP([]js.Value{js.Str(domain)}) {
			if !strings.HasPrefix(domain, "[") || !strings.HasSuffix(domain, "]") {
				return false
			}
			inner := js.SliceJS(domain, 1, -1)
			if inner == "" || !isIP([]js.Value{js.Str(inner)}) {
				return false
			}
		}
	}
	if chars := opt(options, "blacklisted_chars"); js.Truthy(chars) {
		rx := newRegExp(js.Str("["+js.ToString(chars)+"]+"), js.Str("g"))
		if compiledOf(rx.Obj()).Search(js.ToU16(user), 0) != nil {
			return false
		}
	}
	u := js.ToU16(user)
	if len(u) > 0 && u[0] == '"' && u[len(u)-1] == '"' {
		user = js.SliceJS(user, 1, len(u)-1)
		if optTrue(options, "allow_utf8_local_part") {
			return test(reQuotedEmailUserUtf8, user)
		}
		return test(reQuotedEmailUser, user)
	}
	pattern := reEmailUserPart
	if optTrue(options, "allow_utf8_local_part") {
		pattern = reEmailUserUtf8Part
	}
	for _, p := range js.Split16(user, ".") {
		if !test(pattern, p) {
			return false
		}
	}
	return true
}

func parseMailtoQuery(query string) (cc, bcc string, ok bool) {
	allowed := map[string]bool{"subject": true, "body": true, "cc": true, "bcc": true}
	params := js.Split16(query, "&")
	if len(params) > 4 {
		return "", "", false
	}
	for _, q := range params {
		kv := js.Split16(q, "=")
		key := kv[0]
		value := ""
		if len(kv) > 1 {
			value = kv[1]
		}
		if key != "" && !allowed[key] {
			return "", "", false
		}
		if value != "" && key == "cc" {
			cc = value
		}
		if value != "" && key == "bcc" {
			bcc = value
		}
		if key != "" {
			delete(allowed, key)
		}
	}
	return cc, bcc, true
}

func isMailtoURI(a []js.Value) js.Value {
	url := assertString(arg(a, 0))
	if !strings.HasPrefix(url, "mailto:") {
		return js.Bool(false)
	}
	pieces := js.Split16(js.ReplaceFirst(url, "mailto:", ""), "?")
	to, query := pieces[0], ""
	if len(pieces) > 1 {
		query = pieces[1]
	}
	if to == "" && query == "" {
		return js.Bool(true)
	}
	cc, bcc, ok := parseMailtoQuery(query)
	if !ok {
		return js.Bool(false)
	}
	for _, email := range js.Split16(to+","+cc+","+bcc, ",") {
		email = rtrim(js.Str(ltrim(js.Str(email), js.Str(" "))), js.Str(" "))
		if email != "" && !isEmail(js.Str(email), arg(a, 1)) {
			return js.Bool(false)
		}
	}
	return js.Bool(true)
}

var icloudDomains = stringSet("icloud.com", "me.com")

var outlookDomains = stringSet("hotmail.at", "hotmail.be", "hotmail.ca", "hotmail.cl", "hotmail.co.il",
	"hotmail.co.nz", "hotmail.co.th", "hotmail.co.uk", "hotmail.com", "hotmail.com.ar", "hotmail.com.au",
	"hotmail.com.br", "hotmail.com.gr", "hotmail.com.mx", "hotmail.com.pe", "hotmail.com.tr", "hotmail.com.vn",
	"hotmail.cz", "hotmail.de", "hotmail.dk", "hotmail.es", "hotmail.fr", "hotmail.hu", "hotmail.id", "hotmail.ie",
	"hotmail.in", "hotmail.it", "hotmail.jp", "hotmail.kr", "hotmail.lv", "hotmail.my", "hotmail.ph", "hotmail.pt",
	"hotmail.sa", "hotmail.sg", "hotmail.sk", "live.be", "live.co.uk", "live.com", "live.com.ar", "live.com.mx",
	"live.de", "live.es", "live.eu", "live.fr", "live.it", "live.nl", "msn.com", "outlook.at", "outlook.be",
	"outlook.cl", "outlook.co.il", "outlook.co.nz", "outlook.co.th", "outlook.com", "outlook.com.ar",
	"outlook.com.au", "outlook.com.br", "outlook.com.gr", "outlook.com.pe", "outlook.com.tr", "outlook.com.vn",
	"outlook.cz", "outlook.de", "outlook.dk", "outlook.es", "outlook.fr", "outlook.hu", "outlook.id", "outlook.ie",
	"outlook.in", "outlook.it", "outlook.jp", "outlook.kr", "outlook.lv", "outlook.my", "outlook.ph", "outlook.pt",
	"outlook.sa", "outlook.sg", "outlook.sk", "passport.com")

var yahooDomains = stringSet("rocketmail.com", "yahoo.ca", "yahoo.co.uk", "yahoo.com", "yahoo.de", "yahoo.fr",
	"yahoo.in", "yahoo.it", "ymail.com")

var yandexDomains = stringSet("yandex.ru", "yandex.ua", "yandex.kz", "yandex.com", "yandex.by", "ya.ru")

var reDots = re(`\.+`, "g")

func normalizeEmail(a []js.Value) js.Value {
	options := merge(arg(a, 1), []option{{"all_lowercase", js.Bool(true)}, {"gmail_lowercase", js.Bool(true)},
		{"gmail_remove_dots", js.Bool(true)}, {"gmail_remove_subaddress", js.Bool(true)},
		{"gmail_convert_googlemaildotcom", js.Bool(true)}, {"outlookdotcom_lowercase", js.Bool(true)},
		{"outlookdotcom_remove_subaddress", js.Bool(true)}, {"yahoo_lowercase", js.Bool(true)},
		{"yahoo_remove_subaddress", js.Bool(true)}, {"yandex_lowercase", js.Bool(true)},
		{"yandex_convert_yandexru", js.Bool(true)}, {"icloud_lowercase", js.Bool(true)},
		{"icloud_remove_subaddress", js.Bool(true)}})
	email := arg(a, 0)
	if !email.IsString() {
		js.Get(email, "split")
		js.Throw("TypeError", "email.split is not a function")
	}
	raw := js.Split16(email.String(), "@")
	domain := raw[len(raw)-1]
	user := strings.Join(raw[:len(raw)-1], "@")
	domain = js.ToLowerCase(domain)
	lower := func(flag string) bool { return optTrue(options, "all_lowercase") || optTrue(options, flag) }
	switch {
	case domain == "gmail.com" || domain == "googlemail.com":
		if optTrue(options, "gmail_remove_subaddress") {
			user = js.Split16(user, "+")[0]
		}
		if optTrue(options, "gmail_remove_dots") {
			user = replaceFunc(user, reDots, func(m match) string {
				if js.Len16(m.text) > 1 {
					return m.text
				}
				return ""
			})
		}
		if user == "" {
			return js.Bool(false)
		}
		if lower("gmail_lowercase") {
			user = js.ToLowerCase(user)
		}
		if optTrue(options, "gmail_convert_googlemaildotcom") {
			domain = "gmail.com"
		}
	case icloudDomains[domain]:
		if optTrue(options, "icloud_remove_subaddress") {
			user = js.Split16(user, "+")[0]
		}
		if user == "" {
			return js.Bool(false)
		}
		if lower("icloud_lowercase") {
			user = js.ToLowerCase(user)
		}
	case outlookDomains[domain]:
		if optTrue(options, "outlookdotcom_remove_subaddress") {
			user = js.Split16(user, "+")[0]
		}
		if user == "" {
			return js.Bool(false)
		}
		if lower("outlookdotcom_lowercase") {
			user = js.ToLowerCase(user)
		}
	case yahooDomains[domain]:
		if optTrue(options, "yahoo_remove_subaddress") {
			components := js.Split16(user, "-")
			if len(components) > 1 {
				user = strings.Join(components[:len(components)-1], "-")
			} else {
				user = components[0]
			}
		}
		if user == "" {
			return js.Bool(false)
		}
		if lower("yahoo_lowercase") {
			user = js.ToLowerCase(user)
		}
	case yandexDomains[domain]:
		if lower("yandex_lowercase") {
			user = js.ToLowerCase(user)
		}
		if optTrue(options, "yandex_convert_yandexru") {
			domain = "yandex.ru"
		}
	default:
		if optTrue(options, "all_lowercase") {
			user = js.ToLowerCase(user)
		}
	}
	return js.Str(user + "@" + domain)
}
