package validation

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Enhanced validators with proper error handling and comprehensive coverage

// IPv4 validation
func ValidateIPv4(field string, value string) error {
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() == nil {
		return ValidationError{
			Field:   field,
			Tag:     "ipv4",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid IPv4 address", field),
		}
	}
	return nil
}

// IPv6 validation
func ValidateIPv6(field string, value string) error {
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() != nil {
		return ValidationError{
			Field:   field,
			Tag:     "ipv6",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid IPv6 address", field),
		}
	}
	return nil
}

// IP validation (IPv4 or IPv6)
func ValidateIP(field string, value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return ValidationError{
			Field:   field,
			Tag:     "ip",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid IP address", field),
		}
	}
	return nil
}

// CIDR validation
func ValidateCIDR(field string, value string) error {
	_, _, err := net.ParseCIDR(value)
	if err != nil {
		return ValidationError{
			Field:   field,
			Tag:     "cidr",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid CIDR notation", field),
		}
	}
	return nil
}

// MAC address validation
func ValidateMAC(field string, value string) error {
	_, err := net.ParseMAC(value)
	if err != nil {
		return ValidationError{
			Field:   field,
			Tag:     "mac",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid MAC address", field),
		}
	}
	return nil
}

// UUID validation with version support
func ValidateUUID(field string, value string) error {
	_, err := uuid.Parse(value)
	if err != nil {
		return ValidationError{
			Field:   field,
			Tag:     "uuid",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid UUID", field),
		}
	}
	return nil
}

// UUID v4 specific validation
func ValidateUUIDv4(field string, value string) error {
	id, err := uuid.Parse(value)
	if err != nil {
		return ValidationError{
			Field:   field,
			Tag:     "uuid4",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid UUID", field),
		}
	}
	if id.Version() != 4 {
		return ValidationError{
			Field:   field,
			Tag:     "uuid4",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid UUID v4", field),
		}
	}
	return nil
}

// Enhanced email validation (RFC 5322 compliant)
var emailRegexRFC5322 = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

func ValidateEmail(field string, value string) error {
	if len(value) > 254 {
		return ValidationError{
			Field:   field,
			Tag:     "email",
			Value:   value,
			Message: fmt.Sprintf("field '%s' email address is too long", field),
		}
	}
	
	if !emailRegexRFC5322.MatchString(value) {
		return ValidationError{
			Field:   field,
			Tag:     "email",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid email address", field),
		}
	}
	
	// Additional validation: check for valid domain length
	parts := strings.Split(value, "@")
	if len(parts) == 2 {
		localPart := parts[0]
		domain := parts[1]
		
		if len(localPart) > 64 {
			return ValidationError{
				Field:   field,
				Tag:     "email",
				Value:   value,
				Message: fmt.Sprintf("field '%s' email local part is too long", field),
			}
		}
		
		if len(domain) > 253 {
			return ValidationError{
				Field:   field,
				Tag:     "email",
				Value:   value,
				Message: fmt.Sprintf("field '%s' email domain is too long", field),
			}
		}
	}
	
	return nil
}

// Enhanced URL validation
func ValidateURL(field string, value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return ValidationError{
			Field:   field,
			Tag:     "url",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid URL", field),
		}
	}
	
	// Require scheme and host
	if u.Scheme == "" {
		return ValidationError{
			Field:   field,
			Tag:     "url",
			Value:   value,
			Message: fmt.Sprintf("field '%s' URL must have a scheme (http, https, etc.)", field),
		}
	}
	
	if u.Host == "" {
		return ValidationError{
			Field:   field,
			Tag:     "url",
			Value:   value,
			Message: fmt.Sprintf("field '%s' URL must have a host", field),
		}
	}
	
	return nil
}

// HTTP URL validation (http or https only)
func ValidateHTTPURL(field string, value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return ValidationError{
			Field:   field,
			Tag:     "http_url",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid HTTP URL", field),
		}
	}
	
	if u.Scheme != "http" && u.Scheme != "https" {
		return ValidationError{
			Field:   field,
			Tag:     "http_url",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be an HTTP or HTTPS URL", field),
		}
	}
	
	if u.Host == "" {
		return ValidationError{
			Field:   field,
			Tag:     "http_url",
			Value:   value,
			Message: fmt.Sprintf("field '%s' URL must have a host", field),
		}
	}
	
	return nil
}

// Hostname validation (RFC 1123)
var hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

func ValidateHostname(field string, value string) error {
	if len(value) > 253 {
		return ValidationError{
			Field:   field,
			Tag:     "hostname",
			Value:   value,
			Message: fmt.Sprintf("field '%s' hostname is too long", field),
		}
	}
	
	if !hostnameRegex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Tag:     "hostname",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid hostname", field),
		}
	}
	
	return nil
}

// Credit card validation using Luhn algorithm
func ValidateCreditCard(field string, value string) error {
	// Remove spaces and dashes
	cleaned := strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "-", "")
	
	// Check if all characters are digits
	for _, r := range cleaned {
		if !unicode.IsDigit(r) {
			return ValidationError{
				Field:   field,
				Tag:     "creditcard",
				Value:   value,
				Message: fmt.Sprintf("field '%s' must contain only digits, spaces, and dashes", field),
			}
		}
	}
	
	// Check length
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return ValidationError{
			Field:   field,
			Tag:     "creditcard",
			Value:   value,
			Message: fmt.Sprintf("field '%s' credit card number must be between 13 and 19 digits", field),
		}
	}
	
	// Luhn algorithm
	if !luhnCheck(cleaned) {
		return ValidationError{
			Field:   field,
			Tag:     "creditcard",
			Value:   value,
			Message: fmt.Sprintf("field '%s' is not a valid credit card number", field),
		}
	}
	
	return nil
}

// luhnCheck implements the Luhn algorithm for credit card validation
func luhnCheck(cardNumber string) bool {
	var sum int
	alternate := false
	
	// Process digits from right to left
	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(cardNumber[i]))
		
		if alternate {
			digit *= 2
			if digit > 9 {
				digit = digit%10 + digit/10
			}
		}
		
		sum += digit
		alternate = !alternate
	}
	
	return sum%10 == 0
}

// Phone number validation (E.164 format)
var phoneE164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

func ValidatePhone(field string, value string) error {
	if !phoneE164Regex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Tag:     "phone",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid phone number in E.164 format (+1234567890)", field),
		}
	}
	return nil
}

// US phone number validation
var phoneUSRegex = regexp.MustCompile(`^(\+1|1)?[-.\s]?\(?([0-9]{3})\)?[-.\s]?([0-9]{3})[-.\s]?([0-9]{4})$`)

func ValidatePhoneUS(field string, value string) error {
	if !phoneUSRegex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Tag:     "phone_us",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid US phone number", field),
		}
	}
	return nil
}

// Date validation (ISO 8601 format)
func ValidateDate(field string, value string) error {
	_, err := time.Parse("2006-01-02", value)
	if err != nil {
		return ValidationError{
			Field:   field,
			Tag:     "date",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be a valid date in YYYY-MM-DD format", field),
		}
	}
	return nil
}

// DateTime validation (ISO 8601 format)
func ValidateDateTime(field string, value string) error {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	
	for _, format := range formats {
		if _, err := time.Parse(format, value); err == nil {
			return nil
		}
	}
	
	return ValidationError{
		Field:   field,
		Tag:     "datetime",
		Value:   value,
		Message: fmt.Sprintf("field '%s' must be a valid datetime in ISO 8601 format", field),
	}
}

// Time validation (HH:MM:SS format)
func ValidateTime(field string, value string) error {
	_, err := time.Parse("15:04:05", value)
	if err != nil {
		_, err = time.Parse("15:04", value)
		if err != nil {
			return ValidationError{
				Field:   field,
				Tag:     "time",
				Value:   value,
				Message: fmt.Sprintf("field '%s' must be a valid time in HH:MM or HH:MM:SS format", field),
			}
		}
	}
	return nil
}

// Postal code validation by country
func ValidatePostalCode(field string, value string, country string) error {
	patterns := map[string]*regexp.Regexp{
		"US": regexp.MustCompile(`^\d{5}(-\d{4})?$`),
		"CA": regexp.MustCompile(`^[A-Za-z]\d[A-Za-z][ -]?\d[A-Za-z]\d$`),
		"UK": regexp.MustCompile(`^[A-Z]{1,2}\d[A-Z\d]? \d[A-Z]{2}$`),
		"DE": regexp.MustCompile(`^\d{5}$`),
		"FR": regexp.MustCompile(`^\d{5}$`),
		"JP": regexp.MustCompile(`^\d{3}-\d{4}$`),
		"AU": regexp.MustCompile(`^\d{4}$`),
	}
	
	pattern, exists := patterns[strings.ToUpper(country)]
	if !exists {
		return ValidationError{
			Field:   field,
			Tag:     "postal_code",
			Value:   value,
			Message: fmt.Sprintf("field '%s' postal code validation not supported for country '%s'", field, country),
		}
	}
	
	if !pattern.MatchString(value) {
		return ValidationError{
			Field:   field,
			Tag:     "postal_code",
			Value:   value,
			Param:   country,
			Message: fmt.Sprintf("field '%s' must be a valid postal code for %s", field, country),
		}
	}
	
	return nil
}

// JSON validation
func ValidateJSON(field string, value string) error {
	// Try to parse as JSON to validate structure
	var js interface{}
	if err := json.Unmarshal([]byte(value), &js); err != nil {
		return ValidationError{
			Field:   field,
			Tag:     "json",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be valid JSON", field),
		}
	}
	return nil
}

// Base64 validation
var base64Regex = regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)

func ValidateBase64(field string, value string) error {
	if len(value)%4 != 0 {
		return ValidationError{
			Field:   field,
			Tag:     "base64",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be valid base64", field),
		}
	}
	
	if !base64Regex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Tag:     "base64",
			Value:   value,
			Message: fmt.Sprintf("field '%s' must be valid base64", field),
		}
	}
	
	return nil
}

// Password strength validation
func ValidatePasswordStrength(field string, value string, minLength int, requireUpper, requireLower, requireDigit, requireSpecial bool) error {
	if len(value) < minLength {
		return ValidationError{
			Field:   field,
			Tag:     "password",
			Value:   "[REDACTED]",
			Param:   fmt.Sprintf("min=%d", minLength),
			Message: fmt.Sprintf("field '%s' must be at least %d characters long", field, minLength),
		}
	}
	
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	
	for _, r := range value {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	
	if requireUpper && !hasUpper {
		return ValidationError{
			Field:   field,
			Tag:     "password",
			Value:   "[REDACTED]",
			Message: fmt.Sprintf("field '%s' must contain at least one uppercase letter", field),
		}
	}
	
	if requireLower && !hasLower {
		return ValidationError{
			Field:   field,
			Tag:     "password",
			Value:   "[REDACTED]",
			Message: fmt.Sprintf("field '%s' must contain at least one lowercase letter", field),
		}
	}
	
	if requireDigit && !hasDigit {
		return ValidationError{
			Field:   field,
			Tag:     "password",
			Value:   "[REDACTED]",
			Message: fmt.Sprintf("field '%s' must contain at least one digit", field),
		}
	}
	
	if requireSpecial && !hasSpecial {
		return ValidationError{
			Field:   field,
			Tag:     "password",
			Value:   "[REDACTED]",
			Message: fmt.Sprintf("field '%s' must contain at least one special character", field),
		}
	}
	
	return nil
}

// File extension validation
func ValidateFileExtension(field string, filename string, allowedExts []string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ValidationError{
			Field:   field,
			Tag:     "file_ext",
			Value:   filename,
			Message: fmt.Sprintf("field '%s' file must have an extension", field),
		}
	}
	
	// Remove the dot from extension for comparison
	ext = ext[1:]
	
	for _, allowed := range allowedExts {
		if strings.ToLower(allowed) == ext {
			return nil
		}
	}
	
	return ValidationError{
		Field:   field,
		Tag:     "file_ext",
		Value:   filename,
		Param:   strings.Join(allowedExts, ","),
		Message: fmt.Sprintf("field '%s' file extension must be one of: %s", field, strings.Join(allowedExts, ", ")),
	}
}