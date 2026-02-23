package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// date layouts for parsing
const (
	layoutDateFull     = "2006-01-02"
	layoutDateTime     = "2006-01-02 15:04:05"
	layoutDateYearMonth = "2006-01"
)

var (
	// DD-MM-YYYY or D-M-YYYY (with -)
	ddmmyyyyDashRegex = regexp.MustCompile(`^(\d{1,2})-(\d{1,2})-(\d{4})$`)
	// DD.MM.YYYY or D.M.YYYY
	ddmmyyyyDotRegex = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\.(\d{4})$`)
	// DD/MM/YYYY
	ddmmyyyySlashRegex = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4})$`)
	// MM.YYYY or M.YYYY (month and year)
	mmyyyyDotRegex = regexp.MustCompile(`^(\d{1,2})\.(\d{4})$`)
	// MM-YYYY or M-YYYY (month and year; 2 parts only)
	mmyyyyDashRegex = regexp.MustCompile(`^(\d{1,2})-(\d{4})$`)
)

func ParseDateStringAsTime(dateString string) (time.Time, error) {
	s := strings.TrimSpace(dateString)
	if s == "" {
		return time.Time{}, fmt.Errorf("ParseDateStringAsTime failed: empty string")
	}

	// RFC3339
	t, e := time.Parse(time.RFC3339, s)
	if e == nil {
		return t, nil
	}

	// YYYY-MM-DD
	t, e = time.Parse(layoutDateFull, s)
	if e == nil {
		return t, nil
	}

	// YYYY-MM-DD HH:MM:SS
	t, e = time.Parse(layoutDateTime, s)
	if e == nil {
		return t, nil
	}

	// YYYY-MM (partial: year and month)
	if len(s) == 7 && s[4] == '-' {
		t, e = time.Parse(layoutDateYearMonth, s)
		if e == nil {
			return t, nil
		}
	}

	// YYYY (partial: year only) — use mid-year (1 June) for calculations
	if len(s) == 4 {
		year, err := strconv.Atoi(s)
		if err == nil && year >= 1900 && year <= 2100 {
			return time.Date(year, 6, 1, 0, 0, 0, 0, time.UTC), nil
		}
	}

	// MM.YYYY or M.YYYY (month and year)
	if m := mmyyyyDotRegex.FindStringSubmatch(s); m != nil {
		if t, ok := parseMY(m[1], m[2]); ok {
			return t, nil
		}
	}

	// MM-YYYY or M-YYYY (month and year; only if first part is 1-12 to avoid clash with DD-MM-YYYY)
	if m := mmyyyyDashRegex.FindStringSubmatch(s); m != nil {
		month, err := strconv.Atoi(m[1])
		if err == nil && month >= 1 && month <= 12 {
			if t, ok := parseMY(m[1], m[2]); ok {
				return t, nil
			}
		}
	}

	// DD-MM-YYYY
	if m := ddmmyyyyDashRegex.FindStringSubmatch(s); m != nil {
		if t, ok := parseDMY(m[1], m[2], m[3]); ok {
			return t, nil
		}
	}

	// DD.MM.YYYY
	if m := ddmmyyyyDotRegex.FindStringSubmatch(s); m != nil {
		if t, ok := parseDMY(m[1], m[2], m[3]); ok {
			return t, nil
		}
	}

	// DD/MM/YYYY
	if m := ddmmyyyySlashRegex.FindStringSubmatch(s); m != nil {
		if t, ok := parseDMY(m[1], m[2], m[3]); ok {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("ParseDateStringAsTime failed: dateString <%s>", dateString)
}

// DateDisplayString returns the string to store for "date display" when the user
// entered a partial date: "YYYY" for year only, "YYYY-MM" for year+month, or "" for full date.
// Caller should store non-empty in *_display column; empty means show full date.
func DateDisplayString(dateString string) string {
	s := strings.TrimSpace(dateString)
	if s == "" {
		return ""
	}
	// Year only (exactly 4 digits)
	if len(s) == 4 {
		if year, err := strconv.Atoi(s); err == nil && year >= 1900 && year <= 2100 {
			return s
		}
	}
	// Year and month (YYYY-MM)
	if len(s) == 7 && s[4] == '-' {
		if _, err := time.Parse(layoutDateYearMonth, s); err == nil {
			return s
		}
	}
	// Month and year (MM.YYYY or MM-YYYY) — normalize to YYYY-MM for storage
	if m := mmyyyyDotRegex.FindStringSubmatch(s); m != nil {
		if _, ok := parseMY(m[1], m[2]); ok {
			return myToYYYYMM(m[1], m[2])
		}
	}
	if m := mmyyyyDashRegex.FindStringSubmatch(s); m != nil {
		month, err := strconv.Atoi(m[1])
		if err == nil && month >= 1 && month <= 12 {
			if _, ok := parseMY(m[1], m[2]); ok {
				return myToYYYYMM(m[1], m[2])
			}
		}
	}
	return ""
}

// parseMY parses month (1-12) and year, returns time.Time for 1st of that month.
func parseMY(monthStr, yearStr string) (time.Time, bool) {
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return time.Time{}, false
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > 2100 {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC), true
}

// myToYYYYMM returns "YYYY-MM" from month and year strings.
func myToYYYYMM(monthStr, yearStr string) string {
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return ""
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > 2100 {
		return ""
	}
	return fmt.Sprintf("%d-%02d", year, month)
}

// parseDMY parses day, month, year and returns time.Time if valid (day/month/year order).
func parseDMY(dayStr, monthStr, yearStr string) (time.Time, bool) {
	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		return time.Time{}, false
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return time.Time{}, false
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > 2100 {
		return time.Time{}, false
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	// check that the date is valid (e.g. 31 Feb becomes 3 Mar in Go)
	if t.Day() != day || t.Month() != time.Month(month) {
		return time.Time{}, false
	}
	return t, true
}
