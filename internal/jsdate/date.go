// Package jsdate implements the JavaScript Date arithmetic, Date.parse and the date string formats that
// validator.js relies on, for a runtime whose local time zone is UTC.
package jsdate

import (
	"fmt"
	"math"
	"time"
)

const (
	msPerDay    = 86400000.0
	msPerHour   = 3600000.0
	msPerMinute = 60000.0
	msPerSecond = 1000.0
	maxTimeInMs = 8.64e15
	minYear     = -1000000
	maxYear     = 1000000
	minMonth    = -10000000
	maxMonth    = 10000000
)

// Now returns the current time value in milliseconds, like Date.now().
func Now() float64 {
	return float64(time.Now().UnixMilli())
}

func toInteger(x float64) float64 {
	if math.IsNaN(x) {
		return 0
	}
	return math.Trunc(x) + 0
}

// MakeDay computes the day number of year, month and date the way the V8 engine does.
func MakeDay(year, month, date float64) float64 {
	if !(year >= minYear && year <= maxYear && month >= minMonth && month <= maxMonth) ||
		math.IsInf(date, 0) || math.IsNaN(date) {
		return math.NaN()
	}
	y := int64(year)
	m := int64(month)
	y += m / 12
	m %= 12
	if m < 0 {
		m += 12
		y--
	}
	const yearDelta = 399999
	base := int64(365*(1970+yearDelta) + (1970+yearDelta)/4 - (1970+yearDelta)/100 + (1970+yearDelta)/400)
	yd := y + yearDelta
	dayFromYear := 365*yd + yd/4 - yd/100 + yd/400 - base
	common := [12]int64{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	leap := [12]int64{0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335}
	if y%4 != 0 || (y%100 == 0 && y%400 != 0) {
		dayFromYear += common[m]
	} else {
		dayFromYear += leap[m]
	}
	return float64(dayFromYear-1) + toInteger(date)
}

// MakeTime combines hours, minutes, seconds and milliseconds into a time within a day.
func MakeTime(h, m, s, ms float64) float64 {
	for _, x := range []float64{h, m, s, ms} {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return math.NaN()
		}
	}
	return toInteger(h)*msPerHour + toInteger(m)*msPerMinute + toInteger(s)*msPerSecond + toInteger(ms)
}

// MakeDate combines a day number and a time within the day.
func MakeDate(day, t float64) float64 {
	if math.IsNaN(day) || math.IsInf(day, 0) || math.IsNaN(t) || math.IsInf(t, 0) {
		return math.NaN()
	}
	return day*msPerDay + t
}

// TimeClip limits a time value to the range a Date can hold.
func TimeClip(t float64) float64 {
	if math.IsNaN(t) || math.IsInf(t, 0) || math.Abs(t) > maxTimeInMs {
		return math.NaN()
	}
	return toInteger(t)
}

// Construct implements new Date(year, month[, date[, hours[, minutes[, seconds[, ms]]]]]) for numbers already
// converted; absent trailing components are passed as their defaults (date 1, others 0).
func Construct(y, m, dt, h, mi, s, ms float64) float64 {
	yr := y
	if !math.IsNaN(y) {
		yi := toInteger(y)
		if yi >= 0 && yi <= 99 {
			yr = 1900 + yi
		}
	}
	return TimeClip(MakeDate(MakeDay(yr, m, dt), MakeTime(h, mi, s, ms)))
}

// Fields holds the calendar fields of a time value.
type Fields struct {
	Year, Month, Day, Hour, Minute, Second, Millisecond, Weekday int64
}

// Split returns the UTC calendar fields of a valid time value.
func Split(t float64) Fields {
	ms := int64(t)
	days := floorDiv(ms, int64(msPerDay))
	rem := ms - days*int64(msPerDay)
	y, mo, d := civilFromDays(days)
	wd := (days + 4) % 7
	if wd < 0 {
		wd += 7
	}
	return Fields{Year: y, Month: mo, Day: d, Hour: rem / 3600000, Minute: rem / 60000 % 60,
		Second: rem / 1000 % 60, Millisecond: rem % 1000, Weekday: wd}
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// civilFromDays converts days since 1970-01-01 to year, zero-based month and day of month.
func civilFromDays(z int64) (int64, int64, int64) {
	z += 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d := doy - (153*mp+2)/5 + 1
	m := mp + 3
	if m > 12 {
		m -= 12
	}
	if m <= 2 {
		y++
	}
	return y, m - 1, d
}

var dayNames = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
var monthNames = [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// ToISOString formats a valid time value like Date.prototype.toISOString.
func ToISOString(t float64) string {
	f := Split(t)
	var year string
	switch {
	case f.Year >= 0 && f.Year <= 9999:
		year = fmt.Sprintf("%04d", f.Year)
	case f.Year < 0:
		year = fmt.Sprintf("-%06d", -f.Year)
	default:
		year = fmt.Sprintf("+%06d", f.Year)
	}
	return fmt.Sprintf("%s-%02d-%02dT%02d:%02d:%02d.%03dZ", year, f.Month+1, f.Day, f.Hour, f.Minute, f.Second,
		f.Millisecond)
}

// ToDateString formats a time value like Date.prototype.toString in the UTC time zone.
func ToDateString(t float64) string {
	if math.IsNaN(t) {
		return "Invalid Date"
	}
	f := Split(t)
	year := fmt.Sprintf("%04d", f.Year)
	if f.Year < 0 {
		year = fmt.Sprintf("-%06d", -f.Year)
	}
	return fmt.Sprintf("%s %s %02d %s %02d:%02d:%02d GMT+0000 (Coordinated Universal Time)", dayNames[f.Weekday],
		monthNames[f.Month], f.Day, year, f.Hour, f.Minute, f.Second)
}
