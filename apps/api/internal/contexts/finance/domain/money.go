package domain

import (
	"fmt"
	"strings"
)

// currencySymbols maps the currencies this app actually deals in to their
// common symbols. Anything unmapped falls back to the ISO code prefix.
var currencySymbols = map[string]string{
	"PHP": "₱",
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"JPY": "¥",
	"CNY": "¥",
	"AUD": "A$",
	"CAD": "C$",
	"SGD": "S$",
	"INR": "₹",
}

// FormatMinor renders an integer minor-unit amount (cents) as a human-readable
// string with thousands separators and two decimals, e.g. 420000 PHP becomes
// "₱4,200.00". Unmapped currencies use the ISO code, e.g. "XYZ 1,234.56". This
// is a display helper for explanation strings only; it never participates in
// arithmetic.
func FormatMinor(cents int64, currency string) string {
	symbol, ok := currencySymbols[currency]
	if !ok {
		symbol = currency + " "
	}
	neg := cents < 0
	if neg {
		cents = -cents
	}
	intPart := cents / 100
	fracPart := cents % 100

	// Thousands separators on the integer part.
	digits := fmt.Sprintf("%d", intPart)
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}

	prefix := ""
	if neg {
		prefix = "-"
	}
	return fmt.Sprintf("%s%s%s.%02d", prefix, symbol, b.String(), fracPart)
}
