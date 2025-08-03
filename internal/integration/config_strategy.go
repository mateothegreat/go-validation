// Package integration provides go-config integration for zero-reflection validation.
package integration

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/mateothegreat/go-validation"
	"github.com/mateothegreat/go-validation/internal/analyzer"
)

// ConfigValidationStrategy provides go-config compatible validation strategy
type ConfigValidationStrategy interface {
	// Validate validates a configuration struct
	Validate(ctx context.Context, config interface{}) error

	// ValidateWithPath validates a configuration struct with YAML path context
	ValidateWithPath(ctx context.Context, config interface{}, yamlPath string) error

	// GetValidationErrors returns detailed validation errors with context
	GetValidationErrors() []EnhancedValidationError

	// SetFailFast configures fail-fast behavior
	SetFailFast(enabled bool)
}

// EnhancedValidationError provides enhanced error information for configuration validation
type EnhancedValidationError struct {
	validation.ValidationError
	YAMLPath     string            `json:"yaml_path"`
	ConfigSource string            `json:"config_source"`
	Suggestions  []string          `json:"suggestions,omitempty"`
	Context      map[string]string `json:"context,omitempty"`
}

// GeneratedStrategy implements ConfigValidationStrategy using generated validators
type GeneratedStrategy struct {
	validators     map[string]ValidatorInterface
	analysisResult *analyzer.AnalysisResult
	errors         []EnhancedValidationError
	failFast       bool
	debugMode      bool
}

// ValidatorInterface defines the interface that generated validators must implement
type ValidatorInterface interface {
	Validate(config interface{}) error
	SetFailFast(enabled bool)
	GetFieldPath(fieldName string) string
}

// NewGeneratedStrategy creates a new generated validation strategy
func NewGeneratedStrategy(analysisResult *analyzer.AnalysisResult) *GeneratedStrategy {
	return &GeneratedStrategy{
		validators:     make(map[string]ValidatorInterface),
		analysisResult: analysisResult,
		errors:         make([]EnhancedValidationError, 0),
		failFast:       false,
		debugMode:      false,
	}
}

// RegisterValidator registers a generated validator for a specific config type
func (gs *GeneratedStrategy) RegisterValidator(typeName string, validator ValidatorInterface) {
	gs.validators[typeName] = validator
}

// Validate validates a configuration struct using the appropriate generated validator
func (gs *GeneratedStrategy) Validate(ctx context.Context, config interface{}) error {
	return gs.ValidateWithPath(ctx, config, "")
}

// ValidateWithPath validates a configuration struct with YAML path context
func (gs *GeneratedStrategy) ValidateWithPath(ctx context.Context, config interface{}, yamlPath string) error {
	// Clear previous errors
	gs.errors = gs.errors[:0]

	// Get the type name of the config
	configType := gs.getConfigTypeName(config)

	// Find the appropriate validator
	validator, exists := gs.validators[configType]
	if !exists {
		return gs.handleUnregisteredType(configType, config, yamlPath)
	}

	// Configure fail-fast for the validator
	validator.SetFailFast(gs.failFast)

	// Perform validation
	if err := validator.Validate(config); err != nil {
		return gs.enhanceValidationErrors(err, yamlPath, "generated")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// GetValidationErrors returns detailed validation errors with context
func (gs *GeneratedStrategy) GetValidationErrors() []EnhancedValidationError {
	return gs.errors
}

// SetFailFast configures fail-fast behavior
func (gs *GeneratedStrategy) SetFailFast(enabled bool) {
	gs.failFast = enabled

	// Propagate to all registered validators
	for _, validator := range gs.validators {
		validator.SetFailFast(enabled)
	}
}

// SetDebugMode enables or disables debug mode for enhanced error reporting
func (gs *GeneratedStrategy) SetDebugMode(enabled bool) {
	gs.debugMode = enabled
}

// getConfigTypeName extracts the type name from a config interface
func (gs *GeneratedStrategy) getConfigTypeName(config interface{}) string {
	configType := reflect.TypeOf(config)

	// Handle pointers
	if configType.Kind() == reflect.Ptr {
		configType = configType.Elem()
	}

	return configType.Name()
}

// handleUnregisteredType handles validation for types without generated validators
func (gs *GeneratedStrategy) handleUnregisteredType(typeName string, config interface{}, yamlPath string) error {
	// Check if we have analysis information for this type
	if structInfo, exists := gs.analysisResult.Structs[typeName]; exists {
		return gs.validateUsingAnalysis(structInfo, config, yamlPath)
	}

	// Fallback to reflection-based validation
	return gs.validateUsingReflection(config, yamlPath)
}

// validateUsingAnalysis validates using analysis information without generated code
func (gs *GeneratedStrategy) validateUsingAnalysis(structInfo *analyzer.StructInfo, config interface{}, yamlPath string) error {
	configValue := reflect.ValueOf(config)
	if configValue.Kind() == reflect.Ptr {
		if configValue.IsNil() {
			gs.addError("", "required", "", "config is nil", yamlPath, "analysis")
			return gs.buildError()
		}
		configValue = configValue.Elem()
	}

	// Validate each field according to analysis
	for _, fieldInfo := range structInfo.Fields {
		fieldValue := configValue.FieldByName(fieldInfo.Name)
		if !fieldValue.IsValid() {
			continue
		}

		fieldYAMLPath := gs.buildFieldYAMLPath(yamlPath, &fieldInfo)

		if err := gs.validateFieldUsingAnalysis(&fieldInfo, fieldValue, fieldYAMLPath); err != nil {
			if gs.failFast {
				return err
			}
		}
	}

	return gs.buildError()
}

// validateFieldUsingAnalysis validates a single field using analysis information
func (gs *GeneratedStrategy) validateFieldUsingAnalysis(fieldInfo *analyzer.FieldInfo, fieldValue reflect.Value, yamlPath string) error {
	// Skip validation if field is a pointer and nil (and not required)
	if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
		if gs.isFieldRequired(fieldInfo) {
			gs.addError(fieldInfo.Name, "required", "", "field is required but is nil", yamlPath, "analysis")
			return gs.buildError()
		}
		return nil
	}

	// Dereference pointer if needed
	if fieldValue.Kind() == reflect.Ptr {
		fieldValue = fieldValue.Elem()
	}

	// Validate each rule
	for _, rule := range fieldInfo.ValidationRules {
		if err := gs.validateRule(fieldInfo, rule, fieldValue, yamlPath); err != nil {
			if gs.failFast {
				return err
			}
		}
	}

	// Handle nested struct validation
	if fieldInfo.IsNested && fieldValue.Kind() == reflect.Struct {
		nestedConfig := fieldValue.Interface()
		if err := gs.ValidateWithPath(context.Background(), nestedConfig, yamlPath); err != nil {
			if gs.failFast {
				return err
			}
		}
	}

	return nil
}

// validateRule validates a single validation rule
func (gs *GeneratedStrategy) validateRule(fieldInfo *analyzer.FieldInfo, rule analyzer.ValidationRule, fieldValue reflect.Value, yamlPath string) error {
	// Build validation tag
	var tag string
	if rule.Parameter != "" {
		tag = rule.Name + "=" + rule.Parameter
	} else {
		tag = rule.Name
	}

	// Use the validation library for the actual validation
	err := validation.Var(fieldValue.Interface(), tag)
	if err != nil {
		gs.addValidationError(err.(validation.ValidationError), yamlPath, "analysis")
		return gs.buildError()
	}

	return nil
}

// validateUsingReflection provides fallback validation using reflection
func (gs *GeneratedStrategy) validateUsingReflection(config interface{}, yamlPath string) error {
	// Use the validation library's reflection-based validation as fallback
	err := validation.Struct(config)
	if err != nil {
		return gs.enhanceValidationErrors(err, yamlPath, "reflection")
	}
	return nil
}

// enhanceValidationErrors converts validation errors to enhanced errors with context
func (gs *GeneratedStrategy) enhanceValidationErrors(err error, yamlPath, source string) error {
	if validationErrors, ok := err.(validation.ValidationErrors); ok {
		for _, valErr := range validationErrors {
			gs.addValidationError(valErr, yamlPath, source)
		}
	} else if valErr, ok := err.(validation.ValidationError); ok {
		gs.addValidationError(valErr, yamlPath, source)
	} else {
		// Handle generic errors
		gs.addError("", "validation", "", err.Error(), yamlPath, source)
	}

	return gs.buildError()
}

// addValidationError adds a validation error to the enhanced error list
func (gs *GeneratedStrategy) addValidationError(valErr validation.ValidationError, yamlPath, source string) {
	fieldYAMLPath := gs.buildFieldYAMLPath(yamlPath, &analyzer.FieldInfo{
		Name:    valErr.Field,
		YAMLTag: strings.ToLower(valErr.Field), // Default YAML name
	})

	enhancedErr := EnhancedValidationError{
		ValidationError: valErr,
		YAMLPath:        fieldYAMLPath,
		ConfigSource:    source,
		Suggestions:     gs.generateSuggestions(valErr),
		Context:         gs.generateContext(valErr, yamlPath),
	}

	gs.errors = append(gs.errors, enhancedErr)
}

// addError adds a custom validation error
func (gs *GeneratedStrategy) addError(field, tag, param, message, yamlPath, source string) {
	valErr := validation.ValidationError{
		Field:   field,
		Tag:     tag,
		Param:   param,
		Message: message,
	}

	enhancedErr := EnhancedValidationError{
		ValidationError: valErr,
		YAMLPath:        yamlPath,
		ConfigSource:    source,
		Suggestions:     gs.generateSuggestions(valErr),
		Context:         gs.generateContext(valErr, yamlPath),
	}

	gs.errors = append(gs.errors, enhancedErr)
}

// buildFieldYAMLPath constructs the full YAML path for a field
func (gs *GeneratedStrategy) buildFieldYAMLPath(basePath string, fieldInfo *analyzer.FieldInfo) string {
	fieldName := fieldInfo.YAMLTag
	if fieldName == "" {
		fieldName = strings.ToLower(fieldInfo.Name)
	}

	if basePath == "" {
		return fieldName
	}
	return basePath + "." + fieldName
}

// isFieldRequired checks if a field is required based on validation rules
func (gs *GeneratedStrategy) isFieldRequired(fieldInfo *analyzer.FieldInfo) bool {
	for _, rule := range fieldInfo.ValidationRules {
		if rule.Name == "required" {
			return true
		}
	}
	return false
}

// generateSuggestions generates helpful suggestions for validation errors
func (gs *GeneratedStrategy) generateSuggestions(valErr validation.ValidationError) []string {
	var suggestions []string

	switch valErr.Tag {
	case "required":
		suggestions = append(suggestions, fmt.Sprintf("Ensure the '%s' field is provided in your configuration", valErr.Field))
		suggestions = append(suggestions, "Check that the field name in your config file matches the expected name")

	case "email":
		suggestions = append(suggestions, "Ensure the email address follows the format: user@domain.com")
		suggestions = append(suggestions, "Check for typos in the email address")

	case "url":
		suggestions = append(suggestions, "Ensure the URL includes a scheme (http:// or https://)")
		suggestions = append(suggestions, "Check that the URL is properly formatted")

	case "min":
		suggestions = append(suggestions, fmt.Sprintf("Ensure the value is at least %s", valErr.Param))
		if strings.Contains(valErr.Field, "port") {
			suggestions = append(suggestions, "Port numbers must be between 1 and 65535")
		}

	case "max":
		suggestions = append(suggestions, fmt.Sprintf("Ensure the value is at most %s", valErr.Param))
		if strings.Contains(valErr.Field, "port") {
			suggestions = append(suggestions, "Port numbers must be between 1 and 65535")
		}

	case "oneof":
		suggestions = append(suggestions, fmt.Sprintf("Valid values are: %s", valErr.Param))
		suggestions = append(suggestions, "Check for typos in the configuration value")

	default:
		suggestions = append(suggestions, fmt.Sprintf("Check the documentation for the '%s' validation rule", valErr.Tag))
	}

	return suggestions
}

// generateContext generates contextual information for validation errors
func (gs *GeneratedStrategy) generateContext(valErr validation.ValidationError, yamlPath string) map[string]string {
	context := make(map[string]string)

	context["validation_rule"] = valErr.Tag
	if valErr.Param != "" {
		context["rule_parameter"] = valErr.Param
	}

	if yamlPath != "" {
		context["yaml_path"] = yamlPath

		// Add section information
		pathParts := strings.Split(yamlPath, ".")
		if len(pathParts) > 1 {
			context["config_section"] = pathParts[0]
		}
	}

	// Add field type information if available
	if gs.analysisResult != nil {
		for _, structInfo := range gs.analysisResult.Structs {
			for _, fieldInfo := range structInfo.Fields {
				if fieldInfo.Name == valErr.Field {
					context["field_type"] = fieldInfo.Type
					if fieldInfo.DefaultValue != "" {
						context["default_value"] = fieldInfo.DefaultValue
					}
					break
				}
			}
		}
	}

	return context
}

// buildError builds the final error from collected validation errors
func (gs *GeneratedStrategy) buildError() error {
	if len(gs.errors) == 0 {
		return nil
	}

	// Convert enhanced errors back to validation errors for compatibility
	var valErrors []validation.ValidationError
	for _, enhancedErr := range gs.errors {
		valErrors = append(valErrors, enhancedErr.ValidationError)
	}

	return validation.ValidationErrors(valErrors)
}

// ConfigStrategyFactory creates validation strategies for configuration management
type ConfigStrategyFactory struct {
	analysisResult *analyzer.AnalysisResult
	strategies     map[string]ConfigValidationStrategy
}

// NewConfigStrategyFactory creates a new strategy factory
func NewConfigStrategyFactory(analysisResult *analyzer.AnalysisResult) *ConfigStrategyFactory {
	return &ConfigStrategyFactory{
		analysisResult: analysisResult,
		strategies:     make(map[string]ConfigValidationStrategy),
	}
}

// CreateGeneratedStrategy creates a generated validation strategy
func (csf *ConfigStrategyFactory) CreateGeneratedStrategy() ConfigValidationStrategy {
	strategy := NewGeneratedStrategy(csf.analysisResult)
	csf.strategies["generated"] = strategy
	return strategy
}

// CreateReflectionStrategy creates a reflection-based validation strategy (fallback)
func (csf *ConfigStrategyFactory) CreateReflectionStrategy() ConfigValidationStrategy {
	strategy := &ReflectionStrategy{
		analysisResult: csf.analysisResult,
		errors:         make([]EnhancedValidationError, 0),
	}
	csf.strategies["reflection"] = strategy
	return strategy
}

// GetStrategy returns a strategy by name
func (csf *ConfigStrategyFactory) GetStrategy(name string) (ConfigValidationStrategy, bool) {
	strategy, exists := csf.strategies[name]
	return strategy, exists
}

// ReflectionStrategy provides a reflection-based validation strategy as fallback
type ReflectionStrategy struct {
	analysisResult *analyzer.AnalysisResult
	errors         []EnhancedValidationError
	failFast       bool
}

// Validate validates using reflection-based validation
func (rs *ReflectionStrategy) Validate(ctx context.Context, config interface{}) error {
	return rs.ValidateWithPath(ctx, config, "")
}

// ValidateWithPath validates using reflection with path context
func (rs *ReflectionStrategy) ValidateWithPath(ctx context.Context, config interface{}, yamlPath string) error {
	rs.errors = rs.errors[:0]

	err := validation.Struct(config)
	if err != nil {
		if validationErrors, ok := err.(validation.ValidationErrors); ok {
			for _, valErr := range validationErrors {
				enhancedErr := EnhancedValidationError{
					ValidationError: valErr,
					YAMLPath:        yamlPath + "." + strings.ToLower(valErr.Field),
					ConfigSource:    "reflection",
					Suggestions:     []string{"Consider using generated validation for better performance"},
				}
				rs.errors = append(rs.errors, enhancedErr)
			}
		}
		return err
	}

	return nil
}

// GetValidationErrors returns validation errors
func (rs *ReflectionStrategy) GetValidationErrors() []EnhancedValidationError {
	return rs.errors
}

// SetFailFast configures fail-fast behavior
func (rs *ReflectionStrategy) SetFailFast(enabled bool) {
	rs.failFast = enabled
}
