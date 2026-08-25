package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// priceRegex matches contiguous numeric digits, commas, and dots
	priceRegex = regexp.MustCompile(`[\d.,]+`)
)

// CleanPrice converts raw messy price strings into a clean float64.
// Supports both European format ("1.299,99 €") and US/UK format ("$1,299.00").
func CleanPrice(raw string) (float64, error) {
	match := priceRegex.FindString(raw)
	if match == "" {
		return 0, fmt.Errorf("no numeric price found in: %q", raw)
	}

	cleaned := match
	lastComma := strings.LastIndex(cleaned, ",")
	lastDot := strings.LastIndex(cleaned, ".")

	if lastComma > lastDot {
		// European format: 1.299,99 -> 1299.99
		cleaned = strings.ReplaceAll(cleaned, ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	} else if lastDot > lastComma {
		// US/UK format: 1,299.99 -> 1299.99
		cleaned = strings.ReplaceAll(cleaned, ",", "")
	}

	val, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %q as float: %w", cleaned, err)
	}
	return val, nil
}
