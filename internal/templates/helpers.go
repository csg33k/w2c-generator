package templates

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// centsToDisplay converts an integer cent value to a "$0.00"-style string.
func centsToDisplay(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

// itoa converts an int64 to a string, used for building URL paths in templ.
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

// formatSSN converts a stored 9-digit SSN (no dashes) to XXX-XX-XXXX display format.
// Returns the original value unchanged if it is not exactly 9 digits.
func formatSSN(ssn string) string {
	digits := strings.ReplaceAll(ssn, "-", "")
	if len(digits) == 9 {
		return digits[:3] + "-" + digits[3:5] + "-" + digits[5:]
	}
	return ssn
}

// formatEIN formats a stored 9-digit EIN (no hyphens) as XX-XXXXXXX.
// Returns the original value unchanged if it is not exactly 9 digits.
func formatEIN(ein string) string {
	digits := strings.ReplaceAll(ein, "-", "")
	if len(digits) == 9 {
		return digits[:2] + "-" + digits[2:]
	}
	return ein
}

// formatPhone formats a stored digit-only US phone number as (XXX) XXX-XXXX.
// Exactly 10 digits are formatted; any other length is returned as-is.
func formatPhone(p string) string {
	var b strings.Builder
	for _, r := range p {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	d := b.String()
	if len(d) == 10 {
		return "(" + d[0:3] + ") " + d[3:6] + "-" + d[6:]
	}
	return p
}

// taxYearPubURL returns the SSA publication URL for a given 4-digit tax year
// string (e.g. "2024" → "https://www.ssa.gov/employer/efw/24efw2c.pdf").
// Returns an empty string for unrecognised input.
func taxYearPubURL(year string) string {
	if len(year) == 4 {
		return "https://www.ssa.gov/employer/efw/" + year[2:] + "efw2c.pdf"
	}
	return ""
}

// boolPtrToFormVal converts a *bool to a <select> form value:
//
//	nil   → "" (no correction)
//	&true → "1" (was/is checked)
//	&false → "0" (was/is unchecked)
func boolPtrToFormVal(b *bool) string {
	if b == nil {
		return ""
	}
	if *b {
		return "1"
	}
	return "0"
}
