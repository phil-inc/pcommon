package ndc

import (
	"fmt"
	"strings"
)

func IsInList(ndc string, ndcs []string) bool {
	ndc = Sanitize(ndc)
	if ndc == "" {
		return false
	}
	return isFound(ndc, ndcs)
}

func isFound(ndc string, ndcs []string) bool {
	for _, n := range ndcs {
		if n == "" {
			continue
		}
		if IsMatch(n, ndc) {
			return true
		}
	}
	return false
}

// IsNDCMatch compares 2 NDCs and checks if it's same
func IsMatch(ndc1 string, ndc2 string) bool {
	ndc1 = Sanitize(ndc1)
	ndc2 = Sanitize(ndc2)

	if len(ndc2) == 11 {
		ndc2 = AddDashesTo11Digit(ndc2)
		ndc2 = ConvertTo10Digits(ndc2)
	}
	if len(ndc1) == 11 {
		ndc1 = AddDashesTo11Digit(ndc1)
		ndc1 = ConvertTo10Digits(ndc1)
	}

	if len(ndc2) == 9 {
		ndc2 = "0" + ndc2
	}
	if len(ndc1) == 9 {
		ndc1 = "0" + ndc1
	}

	return ndc1 == ndc2
}

// ValidateForStandard11DigitNDC checks if the NDC is valid 11-digit standard NDC
func ValidateForStandard11Digit(ndc string) bool {
	if ndc == "" {
		return false
	}

	sndc := Sanitize(ndc)

	return len(sndc) == 11
}

// ConvertNDCTo11Digits returns ndc in 11 digits
// Could be written as 4-4-2 or 5-3-2 or 5-4-1 and should be converted to 11 digit NDC code is 5-4-2.
func ConvertTo11Digits(ndc string) string {
	parts := strings.Split(ndc, "-")
	//NDC does not have dashes, we can't do conversion
	if len(parts) < 3 {
		return ndc
	}
	//NDC already has 11 digits
	if len(Sanitize(ndc)) != 10 {
		return ndc
	}

	//first part could be a candidate
	candidate := parts[0]
	if len(candidate) == 4 {
		parts[0] = addLeadingZero(candidate)
		return strings.Join(parts, "")
	}

	//second part could be a candidate
	candidate = parts[1]
	if len(candidate) == 3 {
		parts[1] = addLeadingZero(candidate)
		return strings.Join(parts, "")
	}

	//last part could be a candidate
	candidate = parts[2]
	if len(candidate) == 1 {
		parts[2] = addLeadingZero(candidate)
		return strings.Join(parts, "")
	}

	return strings.Replace(ndc, "-", "", -1)
}

// ConvertNDCTo10Digits returns ndc in 10 digits
// 11 digit NDC code is 5-4-2. Could be written as 4-4-2 or 5-3-2 or 5-4-1
func ConvertTo10Digits(ndc string) string {
	parts := strings.Split(ndc, "-")
	if len(parts) < 3 {
		return Sanitize(ndc)
	}

	//first part could be a candidate
	candidate := parts[0]
	if len(candidate) == 5 && isLeadingZero(candidate) {
		parts[0] = removeLeadingZero(candidate)
		return strings.Join(parts, "")
	}

	//second part could be a candidate
	candidate = parts[1]
	if len(candidate) == 4 && isLeadingZero(candidate) {
		parts[1] = removeLeadingZero(candidate)
		return strings.Join(parts, "")
	}

	//last part could be a candidate
	candidate = parts[2]
	if len(candidate) == 2 && isLeadingZero(candidate) {
		parts[2] = removeLeadingZero(candidate)
		return strings.Join(parts, "")
	}

	return strings.Replace(ndc, "-", "", -1)
}

// SanitizeNDC cleans up the ndc code
func Sanitize(ndc string) string {
	sanitized := strings.Replace(ndc, "-", "", -1)
	return strings.TrimSpace(sanitized)
}

// AddDashesTo11DigitNDC adds the dash to format the NDC
func AddDashesTo11Digit(s string) string {
	if len(s) < 11 {
		return s
	}

	return fmt.Sprintf("%s-%s-%s", s[:5], s[5:9], s[9:])
}

// AddDashesTo10DigitNDC adds the dash to format the NDC
func AddDashesTo10Digit(s string) string {
	if len(s) != 10 {
		return s
	}

	// if first character is 0 add 0 in first part eg. 00XXX-XXXX-XX
	if s[0] == '0' {
		return fmt.Sprintf("0%s-%s-%s", s[:4], s[4:8], s[8:])
	}

	// if first character is not 0 add 0 in second part eg. XXXXX-0XXX-XX
	return fmt.Sprintf("%s-0%s-%s", s[:5], s[5:8], s[8:])
}

// NDCFromZohoSKU returns NDC from SKU with format X10-digit-NDCY
func FromZohoSKU(formattedSKu string) string {
	sku := strings.ReplaceAll(formattedSKu, "-", "")
	skuLength := len(sku)

	// if SKU format is not based on the agreed spec
	if skuLength != 12 {
		return formattedSKu
	}

	// remove first and last charatcer from sku
	sku = string(sku[1 : skuLength-1])

	return AddDashesTo10Digit(sku)
}

func isLeadingZero(part string) bool {
	return part[0] == '0'
}

func removeLeadingZero(part string) string {
	return part[1:]
}

func addLeadingZero(part string) string {
	return "0" + part
}
