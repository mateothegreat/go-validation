package validation

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Validator provides high-level validation functionality
type Validator struct {
	tagName       string
	rules         map[string][]ValidationFunc
	customRules   map[string]ValidationFunc
	structRules   map[reflect.Type]StructLevelValidationFunc
	fieldNameFunc FieldNameFunc
	errorCollector *ErrorCollector
	config        ValidatorConfig
	mu            sync.RWMutex
}

// ValidationFunc defines a validation function signature
type ValidationFunc func(fl FieldLevel) bool

// StructLevelValidationFunc defines a struct-level validation function
type StructLevelValidationFunc func(sl StructLevel)

// FieldNameFunc defines a function to get field names for errors
type FieldNameFunc func(fld reflect.StructField) string

// ValidatorConfig holds configuration for the validator
type ValidatorConfig struct {
	TagName      string // Default: "validate"
	FailFast     bool   // Stop on first error
	IgnoreFields []string // Fields to ignore during validation
}

// DefaultValidatorConfig returns default configuration
func DefaultValidatorConfig() ValidatorConfig {
	return ValidatorConfig{
		TagName:  "validate",
		FailFast: false,
	}
}

// New creates a new validator with default configuration
func New() *Validator {
	return NewWithConfig(DefaultValidatorConfig())
}

// NewWithConfig creates a new validator with custom configuration
func NewWithConfig(config ValidatorConfig) *Validator {
	v := &Validator{
		tagName:       config.TagName,
		rules:         make(map[string][]ValidationFunc),
		customRules:   make(map[string]ValidationFunc),
		structRules:   make(map[reflect.Type]StructLevelValidationFunc),
		config:        config,
		fieldNameFunc: defaultFieldNameFunc,
	}
	
	// Register built-in validation rules
	v.registerBuiltInRules()
	
	return v
}

// Global validator instance for package-level functions
var defaultValidator = New()

// SetTagName sets the tag name for validation (default: "validate")
func (v *Validator) SetTagName(name string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.tagName = name
}

// SetFieldNameFunc sets the function to use for getting field names
func (v *Validator) SetFieldNameFunc(fn FieldNameFunc) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.fieldNameFunc = fn
}

// RegisterValidation registers a custom validation function
func (v *Validator) RegisterValidation(tag string, fn ValidationFunc) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	if tag == "" {
		return fmt.Errorf("validation tag cannot be empty")
	}
	
	v.customRules[tag] = fn
	return nil
}

// RegisterStructValidation registers a struct-level validation function
func (v *Validator) RegisterStructValidation(fn StructLevelValidationFunc, types ...interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	for _, t := range types {
		v.structRules[reflect.TypeOf(t)] = fn
	}
}

// Struct validates a struct based on its tags
func (v *Validator) Struct(s interface{}) error {
	if s == nil {
		return nil
	}
	
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}
	
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("validation can only be performed on structs, got %s", val.Kind())
	}
	
	collector := NewErrorCollector()
	collector.SetFailFast(v.config.FailFast)
	
	v.validateStruct(val, val.Type(), "", collector)
	
	if collector.HasErrors() {
		return collector.Errors()
	}
	
	return nil
}

// Var validates a single variable against a validation tag
func (v *Validator) Var(field interface{}, tag string) error {
	if tag == "" {
		return nil
	}
	
	val := reflect.ValueOf(field)
	collector := NewErrorCollector()
	
	v.validateField(val, reflect.Value{}, "field", tag, collector)
	
	if collector.HasErrors() {
		return collector.Errors()
	}
	
	return nil
}

// VarWithValue validates a field with another value for comparison
func (v *Validator) VarWithValue(field interface{}, other interface{}, tag string) error {
	// Implementation for cross-field validation
	// This would be used for rules like "eqfield", "nefield", etc.
	return v.Var(field, tag)
}

// validateStruct validates a struct recursively
func (v *Validator) validateStruct(val reflect.Value, typ reflect.Type, namespace string, collector *ErrorCollector) {
	// Check for struct-level validation
	if structFn, exists := v.structRules[typ]; exists {
		sl := &structLevel{
			validator: v,
			top:       val,
			current:   val,
			namespace: namespace,
		}
		structFn(sl)
		if sl.errors.HasErrors() {
			collector.Merge(sl.errors)
		}
	}
	
	// Validate individual fields
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)
		
		// Skip unexported fields
		if !fieldVal.CanInterface() {
			continue
		}
		
		// Skip ignored fields
		if v.isIgnoredField(fieldType.Name) {
			continue
		}
		
		fieldName := v.fieldNameFunc(fieldType)
		fullPath := fieldName
		if namespace != "" {
			fullPath = namespace + "." + fieldName
		}
		
		// Get validation tag
		tag := fieldType.Tag.Get(v.tagName)
		if tag == "" || tag == "-" {
			// Handle nested structs even without validation tags
			if fieldVal.Kind() == reflect.Struct || (fieldVal.Kind() == reflect.Ptr && fieldVal.Type().Elem().Kind() == reflect.Struct) {
				v.validateNestedStruct(fieldVal, fullPath, collector)
			}
			continue
		}
		
		// Set namespace for error collection
		collector.SetNamespace(namespace)
		
		// Handle nested struct validation
		if strings.Contains(tag, "dive") {
			v.validateDive(fieldVal, fullPath, tag, collector)
		} else {
			v.validateField(fieldVal, val, fieldName, tag, collector)
			
			// Also validate nested struct if field is a struct type
			if fieldVal.Kind() == reflect.Struct || (fieldVal.Kind() == reflect.Ptr && fieldVal.Type().Elem().Kind() == reflect.Struct) {
				v.validateNestedStruct(fieldVal, fullPath, collector)
			}
		}
		
		if collector.ShouldStop() {
			return
		}
	}
}

// validateField validates a single field with its validation rules
func (v *Validator) validateField(val reflect.Value, parent reflect.Value, fieldName, tag string, collector *ErrorCollector) {
	rules := strings.Split(tag, ",")
	
	// Check if omitempty is present
	hasOmitEmpty := false
	for _, rule := range rules {
		if strings.TrimSpace(rule) == "omitempty" {
			hasOmitEmpty = true
			break
		}
	}
	
	// If omitempty is present and field has no value, only validate required-like rules
	if hasOmitEmpty && !HasValue(&fieldLevel{
		validator: v,
		top:       parent,
		parent:    parent,
		field:     val,
		fieldName: fieldName,
	}) {
		// Only process required-like rules for empty fields with omitempty
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}
			
			parts := strings.SplitN(rule, "=", 2)
			ruleName := parts[0]
			
			// Only validate required-like rules
			if strings.HasPrefix(ruleName, "required") {
				var param string
				if len(parts) > 1 {
					param = parts[1]
				}
				
				fl := &fieldLevel{
					validator:   v,
					top:         parent,
					parent:      parent,
					field:       val,
					fieldName:   fieldName,
					param:       param,
					tag:         ruleName,
				}
				
				if customFn, exists := v.customRules[ruleName]; exists {
					if !customFn(fl) {
						collector.AddFieldErrorWithParam(fieldName, ruleName, param, 
							v.getErrorMessage(ruleName, fieldName, param), val.Interface())
					}
				}
			}
		}
		return
	}

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" || rule == "omitempty" {
			continue
		}
		
		// Parse rule and parameters
		parts := strings.SplitN(rule, "=", 2)
		ruleName := parts[0]
		var param string
		if len(parts) > 1 {
			param = parts[1]
		}
		
		// Skip validation if field is nil and rule is not "required"
		if !val.IsValid() || (val.Kind() == reflect.Ptr && val.IsNil()) {
			if ruleName != "required" {
				continue
			}
		}
		
		// Create field level context
		fl := &fieldLevel{
			validator:   v,
			top:         parent,
			parent:      parent,
			field:       val,
			fieldName:   fieldName,
			param:       param,
			tag:         ruleName,
		}
		
		// Check custom rules first
		if customFn, exists := v.customRules[ruleName]; exists {
			if !customFn(fl) {
				collector.AddFieldErrorWithParam(fieldName, ruleName, param, 
					v.getErrorMessage(ruleName, fieldName, param), val.Interface())
			}
			continue
		}
		
		// Check built-in rules
		if err := v.validateBuiltInRule(fl); err != nil {
			if validationErr, ok := err.(ValidationError); ok {
				collector.Add(validationErr)
			} else {
				collector.AddFieldError(fieldName, ruleName, err.Error())
			}
		}
		
		if collector.ShouldStop() {
			return
		}
	}
}

// validateNestedStruct handles validation of nested structs
func (v *Validator) validateNestedStruct(val reflect.Value, namespace string, collector *ErrorCollector) {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	
	if val.Kind() == reflect.Struct {
		v.validateStruct(val, val.Type(), namespace, collector)
	}
}

// validateDive handles "dive" validation for slices, arrays, and maps
func (v *Validator) validateDive(val reflect.Value, namespace, tag string, collector *ErrorCollector) {
	// Remove "dive" from tag to get rules for elements
	tag = strings.ReplaceAll(tag, "dive", "")
	tag = strings.TrimSpace(strings.Trim(tag, ","))
	
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			elemVal := val.Index(i)
			elemPath := fmt.Sprintf("%s[%d]", namespace, i)
			
			if tag != "" {
				v.validateField(elemVal, reflect.Value{}, elemPath, tag, collector)
			} else if elemVal.Kind() == reflect.Struct {
				v.validateNestedStruct(elemVal, elemPath, collector)
			}
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			elemVal := val.MapIndex(key)
			elemPath := fmt.Sprintf("%s[%v]", namespace, key.Interface())
			
			if tag != "" {
				v.validateField(elemVal, reflect.Value{}, elemPath, tag, collector)
			} else if elemVal.Kind() == reflect.Struct {
				v.validateNestedStruct(elemVal, elemPath, collector)
			}
		}
	}
}

// isIgnoredField checks if a field should be ignored
func (v *Validator) isIgnoredField(fieldName string) bool {
	for _, ignored := range v.config.IgnoreFields {
		if fieldName == ignored {
			return true
		}
	}
	return false
}

// getErrorMessage returns an appropriate error message for a validation rule
func (v *Validator) getErrorMessage(rule, field, param string) string {
	switch rule {
	case "required":
		return fmt.Sprintf(ErrorMsgRequired, field)
	case "min":
		return fmt.Sprintf(ErrorMsgMin, field, param)
	case "max":
		return fmt.Sprintf(ErrorMsgMax, field, param)
	case "len":
		return fmt.Sprintf(ErrorMsgLength, field, param)
	case "email":
		return fmt.Sprintf(ErrorMsgEmail, field)
	case "url":
		return fmt.Sprintf(ErrorMsgURL, field)
	case "oneof":
		return fmt.Sprintf(ErrorMsgOneOf, field, param)
	default:
		return fmt.Sprintf("field '%s' failed validation '%s'", field, rule)
	}
}

// defaultFieldNameFunc returns the field name from struct field
func defaultFieldNameFunc(fld reflect.StructField) string {
	// Check for json tag first
	if jsonTag := fld.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
		if name := strings.Split(jsonTag, ",")[0]; name != "" {
			return name
		}
	}
	
	// Use field name
	return fld.Name
}

// Package-level convenience functions

// Struct validates a struct using the default validator
func Struct(s interface{}) error {
	return defaultValidator.Struct(s)
}

// Var validates a variable using the default validator
func Var(field interface{}, tag string) error {
	return defaultValidator.Var(field, tag)
}

// RegisterValidation registers a validation function on the default validator
func RegisterValidation(tag string, fn ValidationFunc) error {
	return defaultValidator.RegisterValidation(tag, fn)
}

// RegisterStructValidation registers a struct validation function on the default validator
func RegisterStructValidation(fn StructLevelValidationFunc, types ...interface{}) {
	defaultValidator.RegisterStructValidation(fn, types...)
}