package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/mateothegreat/go-config/errors"
)

// UnifiedValidator provides validation using multiple strategies
type UnifiedValidator struct {
	config             ValidatorConfig
	detector           *ValidationDetector
	generatedRegistry  GeneratedValidatorRegistry
	reflectionRegistry ValidatorRegistry
	fastValidator      TypedValidator
}

// NewUnifiedValidator creates a new unified validator
func NewUnifiedValidator(config ValidatorConfig) *UnifiedValidator {
	detector := NewValidationDetector(config)
	registry := NewValidatorRegistry()
	registerBuiltInValidators(registry)

	return &UnifiedValidator{
		config:             config,
		detector:           detector,
		generatedRegistry:  globalGeneratedRegistry,
		reflectionRegistry: registry,
		fastValidator:      NewFastValidator(),
	}
}

// Validate validates the given data using the most appropriate strategy
func (uv *UnifiedValidator) Validate(data any) error {
	strategy := uv.detector.DetectStrategy(data)

	switch strategy {
	case StrategyGenerated:
		return uv.validateWithGenerated(data)
	case StrategyReflection:
		return uv.validateWithReflection(data)
	case StrategyFast:
		return uv.validateWithFast(data)
	default:
		return uv.validateWithAuto(data)
	}
}

// validateWithGenerated uses generated validation code
func (uv *UnifiedValidator) validateWithGenerated(data any) error {
	// Try interface first
	if gv, ok := data.(GeneratedValidator); ok {
		return gv.Validate()
	}

	// Try registry
	return uv.generatedRegistry.ValidateWithGenerated(data)
}

// validateWithReflection uses reflection-based validation
func (uv *UnifiedValidator) validateWithReflection(data any) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("validation target must be a struct, got %v", val.Kind())
	}

	typ := val.Type()
	collector := NewErrorCollector()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		validateTag := field.Tag.Get("validate")
		if validateTag == "" {
			// For nested structs (including pointer to struct), recursively validate
			if field.Type.Kind() == reflect.Struct {
				if err := uv.validateWithReflection(fieldValue.Interface()); err != nil {
					collector.Add(field.Name, err.Error())
				}
			} else if field.Type.Kind() == reflect.Ptr && !fieldValue.IsNil() && field.Type.Elem().Kind() == reflect.Struct {
				if err := uv.validateWithReflection(fieldValue.Interface()); err != nil {
					collector.Add(field.Name, err.Error())
				}
			}
			continue
		}

		rules := parseValidationRules(validateTag)
		uv.validateFieldWithReflection(field.Name, fieldValue, rules, collector)

		// For nested structs (including pointer to struct), recursively validate
		if field.Type.Kind() == reflect.Struct {
			if err := uv.validateWithReflection(fieldValue.Interface()); err != nil {
				collector.Add(field.Name, err.Error())
			}
		} else if field.Type.Kind() == reflect.Ptr && !fieldValue.IsNil() && field.Type.Elem().Kind() == reflect.Struct {
			if err := uv.validateWithReflection(fieldValue.Interface()); err != nil {
				collector.Add(field.Name, err.Error())
			}
		}

		if uv.config.FailFast && collector.HasErrors() {
			break
		}
	}

	if collector.HasErrors() {
		return collector.Errors()
	}
	return nil
}

// validateWithFast uses type-specific fast validation
func (uv *UnifiedValidator) validateWithFast(data any) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("validation target must be a struct, got %v", val.Kind())
	}

	typ := val.Type()
	var allErrors []string

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		if !fieldValue.CanInterface() {
			continue
		}

		validateTag := field.Tag.Get("validate")
		if validateTag == "" {
			// For nested structs (including pointer to struct), recursively validate
			if field.Type.Kind() == reflect.Struct {
				if err := uv.validateWithFast(fieldValue.Interface()); err != nil {
					allErrors = append(allErrors, fmt.Sprintf("%s: %s", field.Name, err.Error()))
				}
			} else if field.Type.Kind() == reflect.Ptr && !fieldValue.IsNil() && field.Type.Elem().Kind() == reflect.Struct {
				if err := uv.validateWithFast(fieldValue.Interface()); err != nil {
					allErrors = append(allErrors, fmt.Sprintf("%s: %s", field.Name, err.Error()))
				}
			}
			continue
		}

		rules := parseValidationRules(validateTag)
		fieldErrors := uv.validateFieldByType(field.Name, fieldValue, rules)
		allErrors = append(allErrors, fieldErrors...)

		// For nested structs (including pointer to struct), recursively validate
		if field.Type.Kind() == reflect.Struct {
			if err := uv.validateWithFast(fieldValue.Interface()); err != nil {
				allErrors = append(allErrors, fmt.Sprintf("%s: %s", field.Name, err.Error()))
			}
		} else if field.Type.Kind() == reflect.Ptr && !fieldValue.IsNil() && field.Type.Elem().Kind() == reflect.Struct {
			if err := uv.validateWithFast(fieldValue.Interface()); err != nil {
				allErrors = append(allErrors, fmt.Sprintf("%s: %s", field.Name, err.Error()))
			}
		}

		if uv.config.FailFast && len(allErrors) > 0 {
			break
		}
	}

	if len(allErrors) > 0 {
		var validationErrors errors.ValidationErrors
		for _, errMsg := range allErrors {
			validationErrors = append(validationErrors, errors.ValidationError{
				Message: errMsg,
			})
		}
		return validationErrors
	}
	return nil
}

// validateWithAuto automatically chooses the best strategy
func (uv *UnifiedValidator) validateWithAuto(data any) error {
	strategy := uv.detector.DetectStrategy(data)

	// Use the detected strategy
	switch strategy {
	case StrategyGenerated:
		return uv.validateWithGenerated(data)
	case StrategyReflection:
		return uv.validateWithReflection(data)
	case StrategyFast:
		return uv.validateWithFast(data)
	default:
		return uv.validateWithReflection(data)
	}
}

// validateFieldWithReflection validates a field using reflection-based validators
func (uv *UnifiedValidator) validateFieldWithReflection(fieldName string, fieldValue reflect.Value, rules map[string]string, collector ErrorCollector) {
	for ruleName, ruleValue := range rules {
		if validator, exists := uv.reflectionRegistry.GetValidator(ruleName); exists {
			if err := validator(fieldName, fieldValue, ruleValue); err != nil {
				collector.Add(fieldName, err.Error())
				if uv.config.FailFast {
					return
				}
			}
		} else if !uv.config.AllowUnknownTags {
			collector.Add(fieldName, fmt.Sprintf("unknown validation rule: %s", ruleName))
		}
	}
}

// validateFieldByType validates a field using type-specific fast validation
func (uv *UnifiedValidator) validateFieldByType(fieldName string, fieldValue reflect.Value, rules map[string]string) []string {
	switch fieldValue.Kind() {
	case reflect.String:
		return uv.fastValidator.ValidateString(fieldName, fieldValue.String(), rules)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uv.fastValidator.ValidateInt(fieldName, int(fieldValue.Int()), rules)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uv.fastValidator.ValidateInt(fieldName, int(fieldValue.Uint()), rules)
	case reflect.Float32, reflect.Float64:
		return uv.fastValidator.ValidateFloat(fieldName, fieldValue.Float(), rules)
	case reflect.Bool:
		return uv.fastValidator.ValidateBool(fieldName, fieldValue.Bool(), rules)
	default:
		// For unsupported types, fall back to string representation
		return uv.fastValidator.ValidateString(fieldName, fmt.Sprintf("%v", fieldValue.Interface()), rules)
	}
}

// RegisterValidator registers a new reflection-based validator
func (uv *UnifiedValidator) RegisterValidator(name string, validator FieldValidator) {
	uv.reflectionRegistry.RegisterValidator(name, validator)
}

// GetValidationInfo returns information about validation capabilities for the given data
func (uv *UnifiedValidator) GetValidationInfo(data any) ValidationInfo {
	return uv.detector.GetValidationInfo(data)
}

// parseValidationRules parses validation tag string into rules map
func parseValidationRules(tag string) map[string]string {
	rules := make(map[string]string)
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			rules[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		} else {
			rules[part] = ""
		}
	}

	return rules
}

// registerBuiltInValidators registers the standard validation rules
func registerBuiltInValidators(registry ValidatorRegistry) {
	// Required validator
	registry.RegisterValidator("required", func(fieldName string, value reflect.Value, rule string) error {
		if !value.IsValid() {
			return fmt.Errorf("field '%s' is required", fieldName)
		}

		switch value.Kind() {
		case reflect.String:
			if value.String() == "" {
				return fmt.Errorf("field '%s' is required", fieldName)
			}
		case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan:
			if value.IsNil() {
				return fmt.Errorf("field '%s' is required", fieldName)
			}
			// For pointers to structs, we also need to validate that the struct itself is valid
			if value.Kind() == reflect.Ptr && !value.IsNil() && value.Elem().Kind() == reflect.Struct {
				// The pointer is not nil, which satisfies the "required" constraint
				// The actual struct validation will be handled by recursive validation
				return nil
			}
		case reflect.Array:
			// Arrays are never nil, but we can check if all elements are zero
			if value.Len() == 0 {
				return fmt.Errorf("field '%s' is required", fieldName)
			}
		}
		return nil
	})

	// MinLen validator
	registry.RegisterValidator("minlen", func(fieldName string, value reflect.Value, rule string) error {
		minLen, err := strconv.Atoi(rule)
		if err != nil {
			return fmt.Errorf("invalid minlen rule '%s' for field '%s'", rule, fieldName)
		}

		var length int
		switch value.Kind() {
		case reflect.String:
			length = len(value.String())
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
			length = value.Len()
		default:
			return fmt.Errorf("minlen validation not supported for type %v", value.Kind())
		}

		if length < minLen {
			return fmt.Errorf("field '%s' must be at least %d characters/elements long", fieldName, minLen)
		}
		return nil
	})

	// MaxLen validator
	registry.RegisterValidator("maxlen", func(fieldName string, value reflect.Value, rule string) error {
		maxLen, err := strconv.Atoi(rule)
		if err != nil {
			return fmt.Errorf("invalid maxlen rule '%s' for field '%s'", rule, fieldName)
		}

		var length int
		switch value.Kind() {
		case reflect.String:
			length = len(value.String())
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
			length = value.Len()
		default:
			return fmt.Errorf("maxlen validation not supported for type %v", value.Kind())
		}

		if length > maxLen {
			return fmt.Errorf("field '%s' must be at most %d characters/elements long", fieldName, maxLen)
		}
		return nil
	})

	// Email validator
	registry.RegisterValidator("email", func(fieldName string, value reflect.Value, rule string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("email validation only supports string fields")
		}

		email := value.String()
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(email) {
			return fmt.Errorf("field '%s' must be a valid email address", fieldName)
		}
		return nil
	})

	// Min validator for numeric fields
	registry.RegisterValidator("min", func(fieldName string, value reflect.Value, rule string) error {
		minVal, err := strconv.ParseFloat(rule, 64)
		if err != nil {
			return fmt.Errorf("invalid min rule '%s' for field '%s'", rule, fieldName)
		}

		var numVal float64
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			numVal = float64(value.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			numVal = float64(value.Uint())
		case reflect.Float32, reflect.Float64:
			numVal = value.Float()
		default:
			return fmt.Errorf("min validation not supported for type %v", value.Kind())
		}

		if numVal < minVal {
			return fmt.Errorf("field '%s' must be at least %g", fieldName, minVal)
		}
		return nil
	})

	// Max validator for numeric fields
	registry.RegisterValidator("max", func(fieldName string, value reflect.Value, rule string) error {
		maxVal, err := strconv.ParseFloat(rule, 64)
		if err != nil {
			return fmt.Errorf("invalid max rule '%s' for field '%s'", rule, fieldName)
		}

		var numVal float64
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			numVal = float64(value.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			numVal = float64(value.Uint())
		case reflect.Float32, reflect.Float64:
			numVal = value.Float()
		default:
			return fmt.Errorf("max validation not supported for type %v", value.Kind())
		}

		if numVal > maxVal {
			return fmt.Errorf("field '%s' must be at most %g", fieldName, maxVal)
		}
		return nil
	})

	// OneOf validator
	registry.RegisterValidator("oneof", func(fieldName string, value reflect.Value, rule string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("oneof validation only supports string fields")
		}

		stringVal := value.String()
		options := strings.Split(rule, "|")
		
		for _, option := range options {
			option = strings.TrimSpace(option)
			if stringVal == option {
				return nil
			}
		}

		return fmt.Errorf("field '%s' must be one of [%s]", fieldName, strings.Join(options, ", "))
	})

	// Regex validator
	registry.RegisterValidator("regex", func(fieldName string, value reflect.Value, rule string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("regex validation only supports string fields")
		}

		stringVal := value.String()
		regex, err := regexp.Compile(rule)
		if err != nil {
			return fmt.Errorf("invalid regex pattern '%s' for field '%s': %v", rule, fieldName, err)
		}

		if !regex.MatchString(stringVal) {
			return fmt.Errorf("field '%s' does not match pattern '%s'", fieldName, rule)
		}
		return nil
	})
}
