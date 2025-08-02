package validation

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/mateothegreat/go-config/errors"
)

// generatedValidatorRegistry implements GeneratedValidatorRegistry
type generatedValidatorRegistry struct {
	mu         sync.RWMutex
	validators map[string]func(any) error
}

// Global generated validator registry
var globalGeneratedRegistry = &generatedValidatorRegistry{
	validators: make(map[string]func(any) error),
}

// GetGeneratedValidator retrieves a generated validator globally
func GetGeneratedValidator(structName string) (func(any) error, bool) {
	return globalGeneratedRegistry.GetGeneratedValidator(structName)
}

func (r *generatedValidatorRegistry) RegisterGeneratedValidator(structName string, validator func(any) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validators[structName] = validator
}

func (r *generatedValidatorRegistry) GetGeneratedValidator(structName string) (func(any) error, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	validator, exists := r.validators[structName]
	return validator, exists
}

func (r *generatedValidatorRegistry) HasGeneratedValidator(data any) bool {
	// First check if it implements GeneratedValidator interface
	if _, ok := data.(GeneratedValidator); ok {
		return true
	}

	// Then check registry
	structName := getStructName(data)
	_, exists := r.GetGeneratedValidator(structName)
	return exists
}

func (r *generatedValidatorRegistry) ValidateWithGenerated(data any) error {
	// Try interface first
	if gv, ok := data.(GeneratedValidator); ok {
		return gv.Validate()
	}

	// Try registry
	structName := getStructName(data)
	if validator, exists := r.GetGeneratedValidator(structName); exists {
		return validator(data)
	}

	return fmt.Errorf("no generated validator found for type %s", structName)
}

// reflectionValidatorRegistry implements ValidatorRegistry
type reflectionValidatorRegistry struct {
	mu         sync.RWMutex
	validators map[string]FieldValidator
}

// NewValidatorRegistry creates a new reflection-based validator registry
func NewValidatorRegistry() ValidatorRegistry {
	return &reflectionValidatorRegistry{
		validators: make(map[string]FieldValidator),
	}
}

func (r *reflectionValidatorRegistry) RegisterValidator(name string, validator FieldValidator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validators[name] = validator
}

func (r *reflectionValidatorRegistry) GetValidator(name string) (FieldValidator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	validator, exists := r.validators[name]
	return validator, exists
}

func (r *reflectionValidatorRegistry) ListValidators() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.validators))
	for name := range r.validators {
		names = append(names, name)
	}
	return names
}

// errorCollector implements ErrorCollector
type errorCollector struct {
	errors errors.ValidationErrors
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() ErrorCollector {
	return &errorCollector{
		errors: make(errors.ValidationErrors, 0),
	}
}

func (ec *errorCollector) Add(field, message string) {
	ec.errors = append(ec.errors, errors.ValidationError{
		Field:   field,
		Message: message,
	})
}

func (ec *errorCollector) AddWithValue(field, message string, value any) {
	ec.errors = append(ec.errors, errors.ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

func (ec *errorCollector) AddWithCode(field, message, code string) {
	ec.errors = append(ec.errors, errors.ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

func (ec *errorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

func (ec *errorCollector) Errors() errors.ValidationErrors {
	return ec.errors
}

// Helper functions

func getStructName(data any) string {
	t := reflect.TypeOf(data)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t.Name()
	}
	return ""
}

func getFullTypeName(data any) string {
	t := reflect.TypeOf(data)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.PkgPath() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	return t.Name()
}
