package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// SizeType defines different ways to measure size
type SizeType string

const (
	SizeBytes      SizeType = "bytes"      // Byte length of string
	SizeChars      SizeType = "chars"      // Character count (UTF-8 aware)
	SizeRunes      SizeType = "runes"      // Rune count (same as chars)
	SizeCodePoints SizeType = "codepoints" // Unicode code points
	SizeValue      SizeType = "value"      // Numeric value (for int/float)
	SizeDefault    SizeType = ""           // Default behavior
)

// ByteUnit represents byte unit multipliers
type ByteUnit string

const (
	UnitBytes     ByteUnit = "B"
	UnitKilobytes ByteUnit = "KB"
	UnitMegabytes ByteUnit = "MB"
	UnitGigabytes ByteUnit = "GB"
	UnitTerabytes ByteUnit = "TB"
	UnitPetabytes ByteUnit = "PB"
)

// byteUnitMultipliers maps byte units to their multipliers
var byteUnitMultipliers = map[ByteUnit]int64{
	UnitBytes:     1,
	UnitKilobytes: 1024,
	UnitMegabytes: 1024 * 1024,
	UnitGigabytes: 1024 * 1024 * 1024,
	UnitTerabytes: 1024 * 1024 * 1024 * 1024,
	UnitPetabytes: 1024 * 1024 * 1024 * 1024 * 1024,
}

// SizeSpec represents a size specification with value, type, and optional byte unit
type SizeSpec struct {
	Value    int64
	Type     SizeType
	ByteUnit ByteUnit
	IsBytes  bool // True if this is a byte-based measurement
}

// ParseSizeSpec parses size specifications like "900B", "10MB", "50:chars", "100"
func ParseSizeSpec(rule string) (SizeSpec, error) {
	if rule == "" {
		return SizeSpec{}, fmt.Errorf("empty size rule")
	}

	rule = strings.TrimSpace(rule)

	// First check for byte units (e.g., "10MB", "500KB", "1GB")
	if spec, ok := parseByteUnit(rule); ok {
		return spec, nil
	}

	// Check if rule contains type specifier (value:type)
	if strings.Contains(rule, ":") {
		parts := strings.SplitN(rule, ":", 2)
		if len(parts) != 2 || strings.Contains(parts[1], ":") {
			return SizeSpec{}, fmt.Errorf("invalid size rule format '%s', expected 'value:type'", rule)
		}

		value, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			return SizeSpec{}, fmt.Errorf("invalid size value '%s'", parts[0])
		}

		sizeType := SizeType(strings.TrimSpace(parts[1]))
		return SizeSpec{Value: value, Type: sizeType, IsBytes: false}, nil
	}

	// No type specifier, parse as plain number
	value, err := strconv.ParseInt(strings.TrimSpace(rule), 10, 64)
	if err != nil {
		return SizeSpec{}, fmt.Errorf("invalid size value '%s'", rule)
	}

	return SizeSpec{Value: value, Type: SizeDefault, IsBytes: false}, nil
}

// parseByteUnit attempts to parse byte unit specifications like "10MB", "500KB"
func parseByteUnit(rule string) (SizeSpec, bool) {
	// Try each byte unit suffix
	for unit, multiplier := range byteUnitMultipliers {
		suffix := string(unit)
		if strings.HasSuffix(rule, suffix) {
			valueStr := strings.TrimSuffix(rule, suffix)
			if valueStr == "" {
				continue
			}
			
			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				continue
			}
			
			// Convert to bytes
			totalBytes := int64(value * float64(multiplier))
			
			return SizeSpec{
				Value:    totalBytes,
				Type:     SizeBytes,
				ByteUnit: unit,
				IsBytes:  true,
			}, true
		}
	}
	
	return SizeSpec{}, false
}

// GetSize calculates the size of a value according to the specified type
func GetSize(value any, sizeType SizeType) (int64, error) {
	switch v := value.(type) {
	case string:
		return int64(getStringSize(v, sizeType)), nil
	case int, int8, int16, int32, int64:
		if sizeType == SizeValue || sizeType == SizeDefault {
			return reflect.ValueOf(v).Int(), nil
		}
		return 0, fmt.Errorf("size type '%s' not supported for integer values", sizeType)
	case uint, uint8, uint16, uint32, uint64:
		if sizeType == SizeValue || sizeType == SizeDefault {
			return int64(reflect.ValueOf(v).Uint()), nil
		}
		return 0, fmt.Errorf("size type '%s' not supported for unsigned integer values", sizeType)
	case float32, float64:
		if sizeType == SizeValue || sizeType == SizeDefault {
			return int64(reflect.ValueOf(v).Float()), nil
		}
		return 0, fmt.Errorf("size type '%s' not supported for float values", sizeType)
	case []byte:
		if sizeType == SizeBytes || sizeType == SizeDefault {
			return int64(len(v)), nil
		}
		return 0, fmt.Errorf("size type '%s' not supported for byte slice", sizeType)
	default:
		return 0, fmt.Errorf("unsupported type for size calculation: %T", value)
	}
}

// getStringSize calculates string size based on type
func getStringSize(s string, sizeType SizeType) int {
	switch sizeType {
	case SizeBytes:
		return len(s) // Byte length
	case SizeChars, SizeRunes, SizeCodePoints:
		return utf8.RuneCountInString(s) // UTF-8 rune count
	case SizeDefault:
		return len(s) // Default to byte length for backward compatibility
	default:
		return len(s) // Fallback to byte length
	}
}

// ValidateSize validates a value against a size specification
func ValidateSize(fieldName string, value any, spec SizeSpec, comparison string) error {
	size, err := GetSize(value, spec.Type)
	if err != nil {
		return fmt.Errorf("field '%s': %v", fieldName, err)
	}

	switch comparison {
	case "min":
		if size < spec.Value {
			return fmt.Errorf("field '%s' must be at least %s", fieldName, formatSizeValue(spec))
		}
	case "max":
		if size > spec.Value {
			return fmt.Errorf("field '%s' must be at most %s", fieldName, formatSizeValue(spec))
		}
	case "eq", "len":
		if size != spec.Value {
			return fmt.Errorf("field '%s' must be exactly %s", fieldName, formatSizeValue(spec))
		}
	default:
		return fmt.Errorf("unsupported size comparison: %s", comparison)
	}

	return nil
}

// formatSizeValue formats a size specification for error messages
func formatSizeValue(spec SizeSpec) string {
	if spec.IsBytes && spec.ByteUnit != "" {
		// Convert back to original unit for display
		multiplier := byteUnitMultipliers[spec.ByteUnit]
		if multiplier > 1 && spec.Value%multiplier == 0 {
			originalValue := spec.Value / multiplier
			return fmt.Sprintf("%d%s", originalValue, spec.ByteUnit)
		}
		// If not a clean divisor, show in bytes
		return fmt.Sprintf("%d bytes", spec.Value)
	}
	
	return fmt.Sprintf("%d %s", spec.Value, formatSizeType(spec.Type))
}

// formatSizeType returns a human-readable description of the size type
func formatSizeType(sizeType SizeType) string {
	switch sizeType {
	case SizeBytes:
		return "bytes"
	case SizeChars, SizeRunes:
		return "characters"
	case SizeCodePoints:
		return "code points"
	case SizeValue:
		return "in value"
	case SizeDefault:
		return "in length"
	default:
		return string(sizeType)
	}
}

// Generic comparator for primitive equality validation
func equals[T comparable](fieldName string, v T, expected T) error {
	if v != expected {
		return fmt.Errorf("field '%s' must equal %v", fieldName, expected)
	}
	return nil
}

// Generic range validator for ordered types
func inRange[T comparable](fieldName string, v T, min, max T, compare func(a, b T) int) error {
	if compare(v, min) < 0 || compare(v, max) > 0 {
		return fmt.Errorf("field '%s' must be between %v and %v", fieldName, min, max)
	}
	return nil
}

// Generic minimum validator for ordered types
func minValue[T comparable](fieldName string, v T, min T, compare func(a, b T) int) error {
	if compare(v, min) < 0 {
		return fmt.Errorf("field '%s' must be at least %v", fieldName, min)
	}
	return nil
}

// Generic maximum validator for ordered types
func maxValue[T comparable](fieldName string, v T, max T, compare func(a, b T) int) error {
	if compare(v, max) > 0 {
		return fmt.Errorf("field '%s' must be at most %v", fieldName, max)
	}
	return nil
}

// String validation helpers

// validateStringRequired checks if string is non-empty
func validateStringRequired(fieldName string, value string, _ string) error {
	if value == "" {
		return fmt.Errorf("field '%s' is required", fieldName)
	}
	return nil
}

// validateStringLen validates exact string length with optional size type
func validateStringLen(fieldName string, value string, rule string) error {
	spec, err := ParseSizeSpec(rule)
	if err != nil {
		return fmt.Errorf("invalid len rule '%s' for field '%s': %v", rule, fieldName, err)
	}
	return ValidateSize(fieldName, value, spec, "len")
}

// validateStringMinLen validates minimum string length with optional size type
func validateStringMinLen(fieldName string, value string, rule string) error {
	spec, err := ParseSizeSpec(rule)
	if err != nil {
		return fmt.Errorf("invalid minlen rule '%s' for field '%s': %v", rule, fieldName, err)
	}
	return ValidateSize(fieldName, value, spec, "min")
}

// validateStringMaxLen validates maximum string length with optional size type
func validateStringMaxLen(fieldName string, value string, rule string) error {
	spec, err := ParseSizeSpec(rule)
	if err != nil {
		return fmt.Errorf("invalid maxlen rule '%s' for field '%s': %v", rule, fieldName, err)
	}
	return ValidateSize(fieldName, value, spec, "max")
}

// validateStringRegex validates string against regex pattern
func validateStringRegex(fieldName string, value string, rule string) error {
	regex, err := regexp.Compile(rule)
	if err != nil {
		return fmt.Errorf("invalid regex rule '%s' for field '%s': %v", rule, fieldName, err)
	}
	if !regex.MatchString(value) {
		return fmt.Errorf("field '%s' does not match pattern '%s'", fieldName, rule)
	}
	return nil
}

// validateStringOneOf validates string is one of allowed values
func validateStringOneOf(fieldName string, value string, rule string) error {
	options := strings.Split(rule, "|")
	for _, option := range options {
		if strings.TrimSpace(option) == value {
			return nil
		}
	}
	return fmt.Errorf("field '%s' must be one of [%s]", fieldName, rule)
}

// Pre-compiled regex patterns for common validations
var (
	emailRegex      = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	urlRegex        = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://[^\s]*$`)
	alphaRegex      = regexp.MustCompile(`^[a-zA-Z]+$`)
	alphaNumRegex   = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	numericRegex    = regexp.MustCompile(`^[0-9]+$`)
)

// validateStringEmail validates email format
func validateStringEmail(fieldName string, value string, _ string) error {
	if !emailRegex.MatchString(value) {
		return fmt.Errorf("field '%s' must be a valid email address", fieldName)
	}
	return nil
}

// validateStringURL validates URL format
func validateStringURL(fieldName string, value string, _ string) error {
	if !urlRegex.MatchString(value) {
		return fmt.Errorf("field '%s' must be a valid URL", fieldName)
	}
	return nil
}

// validateStringAlpha validates alphabetic characters only
func validateStringAlpha(fieldName string, value string, _ string) error {
	if !alphaRegex.MatchString(value) {
		return fmt.Errorf("field '%s' must contain only alphabetic characters", fieldName)
	}
	return nil
}

// validateStringAlphaNumeric validates alphanumeric characters only
func validateStringAlphaNumeric(fieldName string, value string, _ string) error {
	if !alphaNumRegex.MatchString(value) {
		return fmt.Errorf("field '%s' must contain only alphanumeric characters", fieldName)
	}
	return nil
}

// validateStringNumeric validates numeric characters only
func validateStringNumeric(fieldName string, value string, _ string) error {
	if !numericRegex.MatchString(value) {
		return fmt.Errorf("field '%s' must contain only numeric characters", fieldName)
	}
	return nil
}

// Integer validation helpers

// validateIntRequired checks if int is non-zero
func validateIntRequired(fieldName string, value int, _ string) error {
	if value == 0 {
		return fmt.Errorf("field '%s' is required", fieldName)
	}
	return nil
}

// validateIntMin validates minimum integer value
func validateIntMin(fieldName string, value int, rule string) error {
	spec, err := ParseSizeSpec(rule)
	if err != nil {
		return fmt.Errorf("invalid min rule '%s' for field '%s': %v", rule, fieldName, err)
	}
	return ValidateSize(fieldName, value, spec, "min")
}

// validateIntMax validates maximum integer value
func validateIntMax(fieldName string, value int, rule string) error {
	spec, err := ParseSizeSpec(rule)
	if err != nil {
		return fmt.Errorf("invalid max rule '%s' for field '%s': %v", rule, fieldName, err)
	}
	return ValidateSize(fieldName, value, spec, "max")
}

// validateIntRange validates integer range
func validateIntRange(fieldName string, value int, rule string) error {
	parts := strings.Split(rule, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid range rule '%s' for field '%s', expected 'min:max'", rule, fieldName)
	}

	min, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid range min '%s' for field '%s'", parts[0], fieldName)
	}

	max, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid range max '%s' for field '%s'", parts[1], fieldName)
	}

	return inRange(fieldName, value, min, max, func(a, b int) int { return a - b })
}

// Float validation helpers

// validateFloatRequired checks if float is non-zero
func validateFloatRequired(fieldName string, value float64, _ string) error {
	if value == 0 {
		return fmt.Errorf("field '%s' is required", fieldName)
	}
	return nil
}

// validateFloatMin validates minimum float value
func validateFloatMin(fieldName string, value float64, rule string) error {
	spec, err := ParseSizeSpec(rule)
	if err != nil {
		return fmt.Errorf("invalid min rule '%s' for field '%s': %v", rule, fieldName, err)
	}
	return ValidateSize(fieldName, value, spec, "min")
}

// validateFloatMax validates maximum float value
func validateFloatMax(fieldName string, value float64, rule string) error {
	spec, err := ParseSizeSpec(rule)
	if err != nil {
		return fmt.Errorf("invalid max rule '%s' for field '%s': %v", rule, fieldName, err)
	}
	return ValidateSize(fieldName, value, spec, "max")
}

// validateFloatRange validates float range
func validateFloatRange(fieldName string, value float64, rule string) error {
	parts := strings.Split(rule, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid range rule '%s' for field '%s', expected 'min:max'", rule, fieldName)
	}

	min, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Errorf("invalid range min '%s' for field '%s'", parts[0], fieldName)
	}

	max, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return fmt.Errorf("invalid range max '%s' for field '%s'", parts[1], fieldName)
	}

	return inRange(fieldName, value, min, max, func(a, b float64) int {
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0
	})
}
