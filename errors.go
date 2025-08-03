package validation

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationError represents a single validation error with structured information
type ValidationError struct {
	Field       string      `json:"field"`              // Field name (e.g., "User.Email")
	Tag         string      `json:"tag"`                // Validation tag (e.g., "required", "email")
	Value       interface{} `json:"value,omitempty"`    // The value that failed validation
	Param       string      `json:"param,omitempty"`    // Rule parameter (e.g., "5" for min=5)
	Message     string      `json:"message"`            // Human-readable error message
	Code        string      `json:"code,omitempty"`     // Error code for programmatic handling
	Namespace   string      `json:"namespace,omitempty"` // Full namespace path (e.g., "User.Address.Street")
	StructField string      `json:"struct_field,omitempty"` // Original struct field name
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	if ve.Message != "" {
		return ve.Message
	}
	return fmt.Sprintf("Field '%s' failed validation '%s'", ve.Field, ve.Tag)
}

// ValidationErrors represents a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	
	if len(ve) == 1 {
		return ve[0].Error()
	}
	
	var messages []string
	for _, err := range ve {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(messages, "; "))
}

// HasErrors returns true if there are any validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// FilterByField returns all errors for a specific field
func (ve ValidationErrors) FilterByField(field string) ValidationErrors {
	var filtered ValidationErrors
	for _, err := range ve {
		if err.Field == field || err.Namespace == field {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// FilterByTag returns all errors for a specific validation tag
func (ve ValidationErrors) FilterByTag(tag string) ValidationErrors {
	var filtered ValidationErrors
	for _, err := range ve {
		if err.Tag == tag {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// Fields returns a list of all field names that have errors
func (ve ValidationErrors) Fields() []string {
	fieldMap := make(map[string]bool)
	var fields []string
	
	for _, err := range ve {
		field := err.Field
		if err.Namespace != "" {
			field = err.Namespace
		}
		if !fieldMap[field] {
			fieldMap[field] = true
			fields = append(fields, field)
		}
	}
	return fields
}

// AsMap returns errors grouped by field name
func (ve ValidationErrors) AsMap() map[string][]ValidationError {
	result := make(map[string][]ValidationError)
	for _, err := range ve {
		field := err.Field
		if err.Namespace != "" {
			field = err.Namespace
		}
		result[field] = append(result[field], err)
	}
	return result
}

// JSON returns the errors as JSON bytes
func (ve ValidationErrors) JSON() ([]byte, error) {
	return json.Marshal(ve)
}

// Add appends a new validation error
func (ve *ValidationErrors) Add(err ValidationError) {
	*ve = append(*ve, err)
}

// AddFieldError adds a simple field error with message
func (ve *ValidationErrors) AddFieldError(field, tag, message string) {
	ve.Add(ValidationError{
		Field:   field,
		Tag:     tag,
		Message: message,
	})
}

// AddFieldErrorWithValue adds a field error with the failing value
func (ve *ValidationErrors) AddFieldErrorWithValue(field, tag, message string, value interface{}) {
	ve.Add(ValidationError{
		Field:   field,
		Tag:     tag,
		Message: message,
		Value:   value,
	})
}

// AddFieldErrorWithParam adds a field error with validation parameter
func (ve *ValidationErrors) AddFieldErrorWithParam(field, tag, param, message string, value interface{}) {
	ve.Add(ValidationError{
		Field:   field,
		Tag:     tag,
		Param:   param,
		Message: message,
		Value:   value,
	})
}

// Merge combines multiple ValidationErrors into one
func (ve *ValidationErrors) Merge(other ValidationErrors) {
	*ve = append(*ve, other...)
}

// Merge combines multiple ValidationErrors into one (for ErrorCollector)
func (ec *ErrorCollector) Merge(other ValidationErrors) {
	ec.errors.Merge(other)
}

// ErrorCollector provides a convenient way to collect validation errors
type ErrorCollector struct {
	errors    ValidationErrors
	namespace string
	failFast  bool
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make(ValidationErrors, 0),
	}
}

// NewErrorCollectorWithNamespace creates a new error collector with a namespace
func NewErrorCollectorWithNamespace(namespace string) *ErrorCollector {
	return &ErrorCollector{
		errors:    make(ValidationErrors, 0),
		namespace: namespace,
	}
}

// SetFailFast configures whether to stop on first error
func (ec *ErrorCollector) SetFailFast(failFast bool) {
	ec.failFast = failFast
}

// SetNamespace sets the namespace for collected errors
func (ec *ErrorCollector) SetNamespace(namespace string) {
	ec.namespace = namespace
}

// Add adds a validation error
func (ec *ErrorCollector) Add(err ValidationError) {
	// Add namespace if not already present
	if ec.namespace != "" && err.Namespace == "" {
		if err.Field != "" {
			err.Namespace = ec.namespace + "." + err.Field
		} else {
			err.Namespace = ec.namespace
		}
	}
	ec.errors.Add(err)
}

// AddFieldError adds a simple field error
func (ec *ErrorCollector) AddFieldError(field, tag, message string) {
	ec.Add(ValidationError{
		Field:   field,
		Tag:     tag,
		Message: message,
	})
}

// AddFieldErrorWithValue adds a field error with value
func (ec *ErrorCollector) AddFieldErrorWithValue(field, tag, message string, value interface{}) {
	ec.Add(ValidationError{
		Field:   field,
		Tag:     tag,
		Message: message,
		Value:   value,
	})
}

// AddFieldErrorWithParam adds a field error with parameter
func (ec *ErrorCollector) AddFieldErrorWithParam(field, tag, param, message string, value interface{}) {
	ec.Add(ValidationError{
		Field:   field,
		Tag:     tag,
		Param:   param,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if any errors were collected
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// ShouldStop returns true if collection should stop (fail fast mode and has errors)
func (ec *ErrorCollector) ShouldStop() bool {
	return ec.failFast && ec.HasErrors()
}

// Errors returns the collected validation errors
func (ec *ErrorCollector) Errors() ValidationErrors {
	return ec.errors
}

// Count returns the number of errors collected
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// Clear removes all collected errors
func (ec *ErrorCollector) Clear() {
	ec.errors = make(ValidationErrors, 0)
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Valid    bool              `json:"valid"`              // Whether validation passed
	Errors   ValidationErrors  `json:"errors,omitempty"`   // Validation errors if any
	Warnings ValidationErrors  `json:"warnings,omitempty"` // Non-fatal validation warnings
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Additional validation metadata
}

// NewValidationResult creates a new validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:    true,
		Errors:   make(ValidationErrors, 0),
		Warnings: make(ValidationErrors, 0),
		Metadata: make(map[string]interface{}),
	}
}

// AddError adds an error and marks the result as invalid
func (vr *ValidationResult) AddError(err ValidationError) {
	vr.Valid = false
	vr.Errors.Add(err)
}

// AddErrors adds multiple errors and marks the result as invalid
func (vr *ValidationResult) AddErrors(errors ValidationErrors) {
	if len(errors) > 0 {
		vr.Valid = false
		vr.Errors.Merge(errors)
	}
}

// AddWarning adds a warning (doesn't affect validity)
func (vr *ValidationResult) AddWarning(warning ValidationError) {
	vr.Warnings.Add(warning)
}

// SetMetadata sets metadata for the validation result
func (vr *ValidationResult) SetMetadata(key string, value interface{}) {
	vr.Metadata[key] = value
}

// Error implements the error interface for ValidationResult
func (vr *ValidationResult) Error() string {
	if vr.Valid {
		return ""
	}
	return vr.Errors.Error()
}

// JSON returns the result as JSON bytes
func (vr *ValidationResult) JSON() ([]byte, error) {
	return json.Marshal(vr)
}

// Common error message templates
var (
	// ErrorMsgRequired is used when a required field is missing
	ErrorMsgRequired = "field '%s' is required"
	
	// ErrorMsgMin is used when a value is below minimum
	ErrorMsgMin = "field '%s' must be at least %s"
	
	// ErrorMsgMax is used when a value exceeds maximum
	ErrorMsgMax = "field '%s' must be at most %s"
	
	// ErrorMsgRange is used when a value is outside range
	ErrorMsgRange = "field '%s' must be between %s and %s"
	
	// ErrorMsgLength is used when length doesn't match
	ErrorMsgLength = "field '%s' must be exactly %s"
	
	// ErrorMsgMinLength is used when length is below minimum
	ErrorMsgMinLength = "field '%s' must be at least %s characters"
	
	// ErrorMsgMaxLength is used when length exceeds maximum
	ErrorMsgMaxLength = "field '%s' must be at most %s characters"
	
	// ErrorMsgEmail is used for invalid email format
	ErrorMsgEmail = "field '%s' must be a valid email address"
	
	// ErrorMsgURL is used for invalid URL format
	ErrorMsgURL = "field '%s' must be a valid URL"
	
	// ErrorMsgOneOf is used when value is not in allowed list
	ErrorMsgOneOf = "field '%s' must be one of [%s]"
	
	// ErrorMsgRegex is used when value doesn't match pattern
	ErrorMsgRegex = "field '%s' does not match required pattern"
	
	// ErrorMsgAlpha is used when value contains non-alphabetic characters
	ErrorMsgAlpha = "field '%s' must contain only alphabetic characters"
	
	// ErrorMsgAlphaNumeric is used when value contains non-alphanumeric characters
	ErrorMsgAlphaNumeric = "field '%s' must contain only alphanumeric characters"
	
	// ErrorMsgNumeric is used when value contains non-numeric characters
	ErrorMsgNumeric = "field '%s' must contain only numeric characters"
)