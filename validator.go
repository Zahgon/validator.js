// Package validator is the Go port of validator.js: string validators and sanitizers that behave exactly like
// the JavaScript library when called with the same JavaScript values, errors included.
//
// Every export takes the arguments of the JavaScript call as js.Value items, so a missing argument and an
// explicit undefined stay distinguishable where the library reads arguments.length.
package validator

import (
	"fmt"

	"github.com/validatorjs/validator.js/js"
)

// Version is the version of validator.js this package reproduces.
const Version = "13.15.35"

type function func(args []js.Value) js.Value

// arg returns the i-th argument, or undefined when the call passed fewer arguments.
func arg(args []js.Value, i int) js.Value {
	if i < len(args) {
		return args[i]
	}
	return js.Undefined
}

// Call invokes the export named name with args, like validator[name](...args) in JavaScript. A thrown JavaScript
// error is returned as a *js.Error. Exports that are not functions (version and the locale lists) return their
// value and ignore args.
func Call(name string, args []js.Value) (result js.Value, err error) {
	if v, ok := values[name]; ok {
		return v(), nil
	}
	f, ok := functions[name]
	if !ok {
		return js.Undefined, fmt.Errorf("validator: unknown export %q", name)
	}
	if jsErr := js.Catch(func() { result = f(args) }); jsErr != nil {
		return js.Undefined, jsErr
	}
	return result, nil
}

// Names returns the names of all exports in the order of the original module object.
func Names() []string {
	return append([]string(nil), exportOrder...)
}

var exportOrder = []string{
	"version", "toDate", "toFloat", "toInt", "toBoolean", "equals", "contains", "matches", "isEmail", "isURL",
	"isMACAddress", "isIP", "isIPRange", "isFQDN", "isBoolean", "isIBAN", "isBIC", "isAbaRouting", "isAlpha",
	"isAlphaLocales", "isAlphanumeric", "isAlphanumericLocales", "isNumeric", "isPassportNumber",
	"passportNumberLocales", "isPort", "isLowercase", "isUppercase", "isAscii", "isFullWidth", "isHalfWidth",
	"isVariableWidth", "isMultibyte", "isSemVer", "isSurrogatePair", "isInt", "isIMEI", "isFloat", "isFloatLocales",
	"isDecimal", "isHexadecimal", "isOctal", "isDivisibleBy", "isHexColor", "isRgbColor", "isHSL", "isISRC", "isMD5",
	"isHash", "isJWT", "isJSON", "isEmpty", "isLength", "isLocale", "isByteLength", "isULID", "isUUID", "isMongoId",
	"isAfter", "isBefore", "isIn", "isLuhnNumber", "isCreditCard", "isIdentityCard", "isEAN", "isISIN", "isISBN",
	"isISSN", "isMobilePhone", "isMobilePhoneLocales", "isPostalCode", "isPostalCodeLocales", "isEthereumAddress",
	"isCurrency", "isBtcAddress", "isISO6346", "isFreightContainerID", "isISO6391", "isISO8601", "isISO15924",
	"isRFC3339", "isISO31661Alpha2", "isISO31661Alpha3", "isISO31661Numeric", "isISO4217", "isBase32", "isBase58",
	"isBase64", "isDataURI", "isMagnetURI", "isMailtoURI", "isMimeType", "isLatLong", "ltrim", "rtrim", "trim",
	"escape", "unescape", "stripLow", "whitelist", "blacklist", "isWhitelisted", "normalizeEmail", "toString",
	"isSlug", "isStrongPassword", "isTaxID", "isDate", "isTime", "isLicensePlate", "isVAT", "ibanLocales",
}

// values holds the exports that are not functions.
var values = map[string]func() js.Value{
	"version": func() js.Value { return js.Str(Version) },
}

// functions maps each function export to its implementation; the module files register themselves.
var functions = map[string]function{}

func register(name string, f function) { functions[name] = f }

func registerValue(name string, f func() js.Value) { values[name] = f }
