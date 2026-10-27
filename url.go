package validator

import (
	"strings"

	"github.com/validatorjs/validator.js/js"
)

func init() {
	register("isURL", func(a []js.Value) js.Value { return js.Bool(isURL(arg(a, 0), arg(a, 1))) })
}

func urlDefaults() []option {
	return []option{{"protocols", js.NewArray(js.Str("http"), js.Str("https"), js.Str("ftp"))},
		{"require_tld", js.Bool(true)}, {"require_protocol", js.Bool(false)}, {"require_host", js.Bool(true)},
		{"require_port", js.Bool(false)}, {"require_valid_protocol", js.Bool(true)},
		{"allow_underscores", js.Bool(false)}, {"allow_trailing_dot", js.Bool(false)},
		{"allow_protocol_relative_urls", js.Bool(false)}, {"allow_fragments", js.Bool(true)},
		{"allow_query_components", js.Bool(true)}, {"validate_length", js.Bool(true)},
		{"max_allowed_length", js.Num(2084)}}
}

var reURLForbidden = re(`[\s<>]`, "")
var reWrappedIPv6 = re(`^\[([^\]]+)\](?::([0-9]+))?$`, "")
var reProtocol = re(`^([a-z][a-z0-9+\-.]*):`, "i")
var reValidAuth = re(`^[a-zA-Z0-9\-_.%:]*$`, "")
var reEncoded = re(`%[0-9a-fA-F]{2}`, "")
var reStartsWithDigit = re(`^[0-9]`, "")
var rePortDigits = re(`^[0-9]+$`, "")

// shift removes and returns the first element, like Array.prototype.shift on a non-empty split result.
func shift(parts []string) (string, []string) { return parts[0], parts[1:] }

func isURL(input js.Value, options js.Value) bool {
	url := assertString(input)
	if url == "" || test(reURLForbidden, url) {
		return false
	}
	if strings.HasPrefix(url, "mailto:") {
		return false
	}
	options = merge(options, urlDefaults())
	if optTrue(options, "validate_length") && js.Greater(js.Num(float64(js.Len16(url))), opt(options, "max_allowed_length")) {
		return false
	}
	if !optTrue(options, "allow_fragments") && strings.Contains(url, "#") {
		return false
	}
	if !optTrue(options, "allow_query_components") && (strings.Contains(url, "?") || strings.Contains(url, "&")) {
		return false
	}
	url, _ = shift(js.Split16(url, "#"))
	url, _ = shift(js.Split16(url, "?"))
	m, hasProtocol := firstMatch(url, reProtocol)
	hadExplicitProtocol := false
	cleanUpProtocol := func(potential string) (string, bool) {
		hadExplicitProtocol = true
		protocol := js.ToLowerCase(potential)
		if optTrue(options, "require_valid_protocol") &&
			indexOfNamed(opt(options, "protocols"), js.Str(protocol), "options.protocols.indexOf") == -1 {
			return "", false
		}
		return js.Substring(url, js.Len16(m.text), js.Len16(url)), true
	}
	if hasProtocol {
		potential := *m.caps[0]
		afterColon := js.Substring(url, js.Len16(m.text), js.Len16(url))
		ok := true
		if js.SliceJS(afterColon, 0, 2) != "//" {
			beforeSlash := afterColon
			if i := js.IndexOf16(afterColon, "/", 0); i != -1 {
				beforeSlash = js.Substring(afterColon, 0, i)
			}
			if at := js.IndexOf16(beforeSlash, "@", 0); at != -1 {
				beforeAt := js.Substring(beforeSlash, 0, at)
				if test(reValidAuth, beforeAt) && !test(reEncoded, beforeAt) {
					if optTrue(options, "require_protocol") {
						return false
					}
				} else {
					url, ok = cleanUpProtocol(potential)
				}
			} else if test(reStartsWithDigit, afterColon) {
				if optTrue(options, "require_protocol") {
					return false
				}
			} else {
				url, ok = cleanUpProtocol(potential)
			}
		} else {
			url, ok = cleanUpProtocol(potential)
		}
		if !ok {
			return false
		}
	} else if optTrue(options, "require_protocol") {
		return false
	}
	if js.SliceJS(url, 0, 2) == "//" {
		if !hadExplicitProtocol && !optTrue(options, "allow_protocol_relative_urls") {
			return false
		}
		url = js.SliceJS(url, 2, js.Len16(url))
	}
	if url == "" {
		return false
	}
	url, _ = shift(js.Split16(url, "/"))
	if url == "" && !optTrue(options, "require_host") {
		return true
	}
	split := js.Split16(url, "@")
	if len(split) > 1 {
		if optTrue(options, "disallow_auth") || split[0] == "" {
			return false
		}
		var auth string
		auth, split = shift(split)
		if strings.Contains(auth, ":") && len(js.Split16(auth, ":")) > 2 {
			return false
		}
		up := js.Split16(auth, ":")
		if up[0] == "" && len(up) > 1 && up[1] == "" {
			return false
		}
	}
	hostname := strings.Join(split, "@")
	var host, ipv6 string
	var portStr *string
	if v6, ok := firstMatch(hostname, reWrappedIPv6); ok {
		ipv6 = *v6.caps[0]
		if v6.caps[1] != nil && *v6.caps[1] != "" {
			portStr = v6.caps[1]
		}
	} else {
		parts := js.Split16(hostname, ":")
		host, parts = shift(parts)
		if len(parts) > 0 {
			p := strings.Join(parts, ":")
			portStr = &p
		}
	}
	if portStr != nil && *portStr != "" {
		port := js.ParseInt(js.Str(*portStr), js.Num(10))
		if !test(rePortDigits, *portStr) || port <= 0 || port > 65535 {
			return false
		}
	} else if optTrue(options, "require_port") {
		return false
	}
	if wl := opt(options, "host_whitelist"); js.Truthy(wl) {
		return checkHost(js.Str(host), wl)
	}
	if host == "" && !optTrue(options, "require_host") {
		return true
	}
	if !isIP([]js.Value{js.Str(host)}) && !isFQDN(host, options) &&
		(ipv6 == "" || !isIP([]js.Value{js.Str(ipv6), js.Num(6)})) {
		return false
	}
	if host == "" {
		host = ipv6
	}
	if bl := opt(options, "host_blacklist"); js.Truthy(bl) && checkHost(js.Str(host), bl) {
		return false
	}
	return true
}
