// Package goconfig provides a unified interface for configuration management and validation
package validation

import (
	"github.com/mateothegreat/go-config/errors"
	"github.com/mateothegreat/go-config/plugins"
)

// Core type aliases for convenience
type (
	// Configuration types - Note: These create import cycles, use config package directly
	// Loader       = config.Loader
	// Builder      = config.Builder
	// LoaderConfig = config.LoaderConfig
	// PluginLoader = config.PluginLoader

	// // Validation types
	// Validator          = validate.Validator
	// GeneratedValidator = validate.GeneratedValidator
	// ValidationStrategy = validate.ValidationStrategy
	// ValidatorConfig    = validate.ValidatorConfig
	// ValidationInfo     = validate.ValidationInfo

	// Error types
	ValidationError    = errors.ValidationError
	ValidationErrors   = errors.ValidationErrors
	ConfigurationError = errors.ConfigurationError
	MultiError         = errors.MultiError
	FluentError        = errors.FluentError
	ErrorCorrelator    = errors.ErrorCorrelator
	CorrelatedError    = errors.CorrelatedError

	// Plugin types
	Plugin          = plugins.Plugin
	ValidatorPlugin = plugins.ValidatorPlugin
	PluginMetadata  = plugins.PluginMetadata
)

// Convenience constants for validation rules
const (
	Required     = "required"
	Min          = "min"
	Max          = "max"
	MinLen       = "minlen"
	MaxLen       = "maxlen"
	Len          = "len"
	Email        = "email"
	URL          = "url"
	Alpha        = "alpha"
	AlphaNumeric = "alphanumeric"
	Numeric      = "numeric"
	Regex        = "regex"
	OneOf        = "oneof"
	Range        = "range"
	IP           = "ip"
	UUID         = "uuid"
	CreditCard   = "creditcard"
	Phone        = "phone"
)

// NewValidator creates a new unified validator
func NewValidator() Validator {
	return NewUnifiedValidator(ValidatorConfig{})
}

// NewValidatorWithConfig creates a validator with custom configuration
func NewValidatorWithConfig(cfg ValidatorConfig) Validator {
	return NewUnifiedValidator(cfg)
}

// DefaultValidatorConfig returns the default validator configuration
func DefaultValidatorConfig() ValidatorConfig {
	return ValidatorConfig{
		Strategy:         StrategyAuto,
		AllowUnknownTags: false,
		FailFast:         false,
		CollectAllErrors: true,
	}
}

// NewDetector creates a new validation detector
// This is an alias for NewValidationDetector for backward compatibility
// Use NewValidationDetector directly instead

// RegisterGeneratedValidator registers a generated validation function
func RegisterGeneratedValidator(structName string, validator func(any) error) {
	globalGeneratedRegistry.RegisterGeneratedValidator(structName, validator)
}

// HasGeneratedValidator checks if a struct has generated validation
func HasGeneratedValidator(data any) bool {
	return globalGeneratedRegistry.HasGeneratedValidator(data)
}

// ValidateWithGenerated validates using generated validation
func ValidateWithGenerated(data any) error {
	return globalGeneratedRegistry.ValidateWithGenerated(data)
}

// Error handling functions

// NewError creates a new fluent error builder
func NewError() *FluentError {
	return errors.NewError()
}

// NewMultiError creates a new multi-error
func NewMultiError() *MultiError {
	return errors.NewMultiError()
}

// NewErrorCorrelator creates a new error correlator
func NewErrorCorrelator() *ErrorCorrelator {
	return errors.NewErrorCorrelator()
}

// AsValidationErrors attempts to convert an error to ValidationErrors
func AsValidationErrors(err error) (ValidationErrors, bool) {
	return errors.AsValidationErrors(err)
}

// AsConfigurationError attempts to convert an error to ConfigurationError
func AsConfigurationError(err error) (ConfigurationError, bool) {
	return errors.AsConfigurationError(err)
}

// Plugin management functions

// RegisterSourceFactory registers a source plugin factory
func RegisterSourceFactory(name string, factory plugins.PluginFactory) {
	plugins.RegisterSourceFactory(name, factory)
}

// RegisterValidatorFactory registers a validator plugin factory
func RegisterValidatorFactory(name string, factory plugins.ValidatorFactory) {
	plugins.RegisterValidatorFactory(name, factory)
}

// CreateSourcePlugin creates a source plugin
func CreateSourcePlugin(name string, opts any) (Plugin, error) {
	return plugins.CreateSourcePlugin(name, opts)
}

// CreateValidatorPlugin creates a validator plugin
func CreateValidatorPlugin(name string) (ValidatorPlugin, error) {
	return plugins.CreateValidatorPlugin(name)
}

// ListSourcePlugins returns all available source plugins
func ListSourcePlugins() []string {
	return plugins.ListSourcePlugins()
}

// ListValidatorPlugins returns all available validator plugins
func ListValidatorPlugins() []string {
	return plugins.ListValidatorPlugins()
}
