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
