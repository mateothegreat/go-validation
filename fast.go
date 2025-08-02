package validation

import (
	"fmt"
)

// FastValidator implements TypedValidator with minimal reflection
type FastValidator struct {
	stringValidators map[string]func(string, string, string) error
	intValidators    map[string]func(string, int, string) error
	boolValidators   map[string]func(string, bool, string) error
	floatValidators  map[string]func(string, float64, string) error
}

// NewFastValidator creates a new fast validator with type-specific functions
func NewFastValidator() TypedValidator {
	fv := &FastValidator{
		stringValidators: make(map[string]func(string, string, string) error),
		intValidators:    make(map[string]func(string, int, string) error),
		boolValidators:   make(map[string]func(string, bool, string) error),
		floatValidators:  make(map[string]func(string, float64, string) error),
	}

	// Register string validators
	fv.stringValidators["required"] = validateStringRequired
	fv.stringValidators["minlen"] = validateStringMinLen
	fv.stringValidators["maxlen"] = validateStringMaxLen
	fv.stringValidators["len"] = validateStringLen
	fv.stringValidators["regex"] = validateStringRegex
	fv.stringValidators["oneof"] = validateStringOneOf
	fv.stringValidators["email"] = validateStringEmail
	fv.stringValidators["url"] = validateStringURL
	fv.stringValidators["alpha"] = validateStringAlpha
	fv.stringValidators["alphanumeric"] = validateStringAlphaNumeric
	fv.stringValidators["numeric"] = validateStringNumeric

	// Register int validators
	fv.intValidators["required"] = validateIntRequired
	fv.intValidators["min"] = validateIntMin
	fv.intValidators["max"] = validateIntMax
	fv.intValidators["range"] = validateIntRange

	// Register float validators
	fv.floatValidators["required"] = validateFloatRequired
	fv.floatValidators["min"] = validateFloatMin
	fv.floatValidators["max"] = validateFloatMax
	fv.floatValidators["range"] = validateFloatRange

	return fv
}

// ValidateString validates a string value using registered string validators
func (fv *FastValidator) ValidateString(fieldName string, value string, rules map[string]string) []string {
	var errors []string
	for ruleName, ruleValue := range rules {
		if validator, exists := fv.stringValidators[ruleName]; exists {
			if err := validator(fieldName, value, ruleValue); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}
	return errors
}

// ValidateInt validates an int value using registered int validators
func (fv *FastValidator) ValidateInt(fieldName string, value int, rules map[string]string) []string {
	var errors []string
	for ruleName, ruleValue := range rules {
		if validator, exists := fv.intValidators[ruleName]; exists {
			if err := validator(fieldName, value, ruleValue); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}
	return errors
}

// ValidateBool validates a bool value (minimal validation needed)
func (fv *FastValidator) ValidateBool(fieldName string, value bool, rules map[string]string) []string {
	var errors []string
	// Bool validation is typically limited, but we can check for required
	if _, required := rules["required"]; required && !value {
		errors = append(errors, fmt.Sprintf("field '%s' is required", fieldName))
	}
	return errors
}

// ValidateFloat validates a float64 value using registered float validators
func (fv *FastValidator) ValidateFloat(fieldName string, value float64, rules map[string]string) []string {
	var errors []string
	for ruleName, ruleValue := range rules {
		if validator, exists := fv.floatValidators[ruleName]; exists {
			if err := validator(fieldName, value, ruleValue); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}
	return errors
}

// String validators - now using primitives.go helpers
// All string validation functions are now defined in primitives.go

// Int validators - now using primitives.go helpers
// All int validation functions are now defined in primitives.go

// Float validators - now using primitives.go helpers
// All float validation functions are now defined in primitives.go
