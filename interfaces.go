// Package validate provides validation interfaces and implementations
package validation

import (
	"reflect"

	"github.com/mateothegreat/go-config/errors"
)

// Validator represents the main validation interface
type Validator[T any] interface {
	Validate(field string, value T, rule string) error
}

// GeneratedValidator represents structs with generated validation methods
type GeneratedValidator interface {
	Validate() error
}

// FieldValidator represents a single validation rule for a field
type FieldValidator func(fieldName string, value reflect.Value, rule string) error

// TypedValidator provides type-specific validation methods without reflection
type TypedValidator interface {
	ValidateString(fieldName string, value string, rules map[string]string) []string
	ValidateInt(fieldName string, value int, rules map[string]string) []string
	ValidateBool(fieldName string, value bool, rules map[string]string) []string
	ValidateFloat(fieldName string, value float64, rules map[string]string) []string
}

// ValidatorRegistry manages validation rule registration
type ValidatorRegistry interface {
	RegisterValidator(name string, validator FieldValidator)
	GetValidator(name string) (FieldValidator, bool)
	ListValidators() []string
}

// GeneratedValidatorRegistry manages generated validation functions
type GeneratedValidatorRegistry interface {
	RegisterGeneratedValidator(structName string, validator func(any) error)
	GetGeneratedValidator(structName string) (func(any) error, bool)
	HasGeneratedValidator(data any) bool
	ValidateWithGenerated(data any) error
}

// Strategy determines which validation approach to use
type Strategy string

const (
	// StrategyAuto automatically detects the best validation method
	StrategyAuto Strategy = "auto"
	// StrategyGenerated forces use of generated validation
	StrategyGenerated Strategy = "generated"
	// StrategyReflection forces use of reflection-based validation
	StrategyReflection Strategy = "reflection"
	// StrategyFast uses optimized type-specific validation
	StrategyFast Strategy = "fast"
)

// ValidatorConfig configures validation behavior
type ValidatorConfig struct {
	Strategy         Strategy
	AllowUnknownTags bool
	FailFast         bool
	CollectAllErrors bool
}

// ErrorCollector collects validation errors during validation
type ErrorCollector interface {
	Add(field, message string)
	AddWithValue(field, message string, value any)
	AddWithCode(field, message, code string)
	HasErrors() bool
	Errors() errors.ValidationErrors
}

// ContextualValidator provides validation with additional context
type ContextualValidator interface {
	ValidateWithContext(data any, ctx ValidationContext) error
}

// ValidationContext provides additional context for validation
type ValidationContext struct {
	Path     string            // Current field path (e.g., "user.address.street")
	Tags     map[string]string // Struct tags
	Metadata map[string]any    // Additional metadata
}
