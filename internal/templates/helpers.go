package templates

import (
	"fmt"
	"strconv"
)

// centsToDisplay converts an integer cent value to a "$0.00"-style string.
func centsToDisplay(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

// itoa converts an int64 to a string, used for building URL paths in templ.
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
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
