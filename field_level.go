package validation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// FieldLevel provides access to the field being validated and its context
type FieldLevel interface {
	// Top returns the top level struct being validated
	Top() reflect.Value
	
	// Parent returns the parent struct of the field being validated
	Parent() reflect.Value
	
	// Field returns the field being validated
	Field() reflect.Value
	
	// FieldName returns the name of the field being validated
	FieldName() string
	
	// StructFieldName returns the struct field name
	StructFieldName() string
	
	// Param returns the parameter passed to the validation function
	Param() string
	
	// GetTag returns the validation tag being processed
	GetTag() string
	
	// ExtractType returns the field type, handling pointers
	ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool)
	
	// GetStructFieldOK returns a field from the parent struct
	GetStructFieldOK() (reflect.Value, reflect.Kind, bool)
	
	// GetStructFieldOK2 returns a field from the current struct by name
	GetStructFieldOK2() (reflect.Value, reflect.Kind, bool)
}

// fieldLevel implements FieldLevel interface
type fieldLevel struct {
	validator     *Validator
	top           reflect.Value
	parent        reflect.Value
	field         reflect.Value
	fieldName     string
	structField   string
	param         string
	tag           string
}

// Top returns the top level struct being validated
func (fl *fieldLevel) Top() reflect.Value {
	return fl.top
}

// Parent returns the parent struct of the field being validated
func (fl *fieldLevel) Parent() reflect.Value {
	return fl.parent
}

// Field returns the field being validated
func (fl *fieldLevel) Field() reflect.Value {
	return fl.field
}

// FieldName returns the name of the field being validated
func (fl *fieldLevel) FieldName() string {
	return fl.fieldName
}

// StructFieldName returns the struct field name
func (fl *fieldLevel) StructFieldName() string {
	return fl.structField
}

// Param returns the parameter passed to the validation function
func (fl *fieldLevel) Param() string {
	return fl.param
}

// GetTag returns the validation tag being processed
func (fl *fieldLevel) GetTag() string {
	return fl.tag
}

// ExtractType returns the field type, handling pointers and interfaces
func (fl *fieldLevel) ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool) {
	switch field.Kind() {
	case reflect.Ptr:
		if field.IsNil() {
			return field, field.Kind(), false
		}
		return fl.ExtractType(field.Elem())
	case reflect.Interface:
		if field.IsNil() {
			return field, field.Kind(), false
		}
		return fl.ExtractType(field.Elem())
	default:
		return field, field.Kind(), true
	}
}

// GetStructFieldOK returns a field from the parent struct
func (fl *fieldLevel) GetStructFieldOK() (reflect.Value, reflect.Kind, bool) {
	return fl.getStructFieldOK(fl.parent, fl.param)
}

// GetStructFieldOK2 returns a field from the current struct by name
func (fl *fieldLevel) GetStructFieldOK2() (reflect.Value, reflect.Kind, bool) {
	return fl.getStructFieldOK(fl.field, fl.param)
}


// StructLevel provides context for struct-level validation
type StructLevel interface {
	// Validator returns the validator instance
	Validator() *Validator
	
	// Top returns the top level struct being validated
	Top() reflect.Value
	
	// Current returns the current struct being validated
	Current() reflect.Value
	
	// ExtractType returns the field type, handling pointers
	ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool)
	
	// ReportError reports an error for struct level validation
	ReportError(field, structField, tag, message string)
	
	// ReportValidationErrors reports validation errors
	ReportValidationErrors(field, structField, tag string, errs ValidationErrors)
}

// structLevel implements StructLevel interface
type structLevel struct {
	validator *Validator
	top       reflect.Value
	current   reflect.Value
	namespace string
	errors    ValidationErrors
}

// Validator returns the validator instance
func (sl *structLevel) Validator() *Validator {
	return sl.validator
}

// Top returns the top level struct being validated
func (sl *structLevel) Top() reflect.Value {
	return sl.top
}

// Current returns the current struct being validated
func (sl *structLevel) Current() reflect.Value {
	return sl.current
}

// ExtractType returns the field type, handling pointers and interfaces
func (sl *structLevel) ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool) {
	switch field.Kind() {
	case reflect.Ptr:
		if field.IsNil() {
			return field, field.Kind(), false
		}
		return sl.ExtractType(field.Elem())
	case reflect.Interface:
		if field.IsNil() {
			return field, field.Kind(), false
		}
		return sl.ExtractType(field.Elem())
	default:
		return field, field.Kind(), true
	}
}

// ReportError reports an error for struct level validation
func (sl *structLevel) ReportError(field, structField, tag, message string) {
	namespace := field
	if sl.namespace != "" {
		namespace = sl.namespace + "." + field
	}
	
	sl.errors.Add(ValidationError{
		Field:       field,
		Tag:         tag,
		Message:     message,
		Namespace:   namespace,
		StructField: structField,
	})
}

// ReportValidationErrors reports multiple validation errors
func (sl *structLevel) ReportValidationErrors(field, structField, tag string, errs ValidationErrors) {
	for _, err := range errs {
		if err.Namespace == "" {
			namespace := field
			if sl.namespace != "" {
				namespace = sl.namespace + "." + field
			}
			err.Namespace = namespace
		}
		if err.StructField == "" {
			err.StructField = structField
		}
		sl.errors.Add(err)
	}
}

// Utility functions for common validation tasks

// ParseParam parses validation parameters
func ParseParam(param string) ([]string, error) {
	if param == "" {
		return nil, nil
	}
	
	// Handle different parameter formats
	// Simple list: "red,green,blue"
	// Range: "1:10"
	// Key-value: "min=1,max=10"
	
	if strings.Contains(param, ":") && len(strings.Split(param, ":")) == 2 {
		// Range format
		return strings.Split(param, ":"), nil
	}
	
	// Comma-separated format
	parts := strings.Split(param, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	
	return parts, nil
}

// ParseIntParam parses integer parameter
func ParseIntParam(param string) (int64, error) {
	if param == "" {
		return 0, nil
	}
	return strconv.ParseInt(param, 10, 64)
}

// ParseFloatParam parses float parameter
func ParseFloatParam(param string) (float64, error) {
	if param == "" {
		return 0, nil
	}
	return strconv.ParseFloat(param, 64)
}

// ParseRangeParam parses range parameters like "1:10"
func ParseRangeParam(param string) (min, max int64, err error) {
	parts := strings.Split(param, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format, expected 'min:max'")
	}
	
	min, err = strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid min value: %v", err)
	}
	
	max, err = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid max value: %v", err)
	}
	
	if min > max {
		return 0, 0, fmt.Errorf("min value cannot be greater than max value")
	}
	
	return min, max, nil
}

// IsEmpty checks if a value is considered empty for validation purposes
func IsEmpty(fl FieldLevel) bool {
	field := fl.Field()
	
	switch field.Kind() {
	case reflect.String:
		return field.Len() == 0
	case reflect.Slice, reflect.Map, reflect.Array:
		return field.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return field.IsNil()
	case reflect.Invalid:
		return true
	default:
		return false
	}
}

// HasValue checks if a field has a non-zero value
func HasValue(fl FieldLevel) bool {
	return !IsEmpty(fl) && !fl.Field().IsZero()
}