package validation

import (
	"reflect"
	"strings"
)

// ValidationDetector detects the best validation strategy for a given struct
type ValidationDetector struct {
	generatedRegistry GeneratedValidatorRegistry
	config            ValidatorConfig
}

// NewValidationDetector creates a new validation detector
func NewValidationDetector(config ValidatorConfig) *ValidationDetector {
	return &ValidationDetector{
		generatedRegistry: globalGeneratedRegistry,
		config:            config,
	}
}

// DetectStrategy determines the best validation strategy for the given data
func (vd *ValidationDetector) DetectStrategy(data any) Strategy {
	if vd.config.Strategy != StrategyAuto {
		return vd.config.Strategy
	}

	// Check for generated validation first (highest priority)
	if vd.hasGeneratedValidation(data) {
		return StrategyGenerated
	}

	// Check for struct tags
	if vd.hasValidationTags(data) {
		return StrategyReflection
	}

	// Default to fast validation
	return StrategyFast
}

// hasGeneratedValidation checks if the struct has generated validation
func (vd *ValidationDetector) hasGeneratedValidation(data any) bool {
	// Check if it implements GeneratedValidator interface
	if _, ok := data.(GeneratedValidator); ok {
		return true
	}

	// Check if it's registered in the generated validator registry
	return vd.generatedRegistry.HasGeneratedValidator(data)
}

// hasValidationTags checks if the struct has validation tags
func (vd *ValidationDetector) hasValidationTags(data any) bool {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return false
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if !val.Field(i).CanInterface() {
			continue
		}

		validateTag := field.Tag.Get("validate")
		if validateTag != "" {
			return true
		}
	}

	return false
}

// CanValidate checks if the data can be validated using any strategy
func (vd *ValidationDetector) CanValidate(data any) bool {
	strategy := vd.DetectStrategy(data)
	return strategy != StrategyFast || vd.hasValidationTags(data) || vd.hasGeneratedValidation(data)
}

// GetValidationInfo returns detailed information about validation capabilities
func (vd *ValidationDetector) GetValidationInfo(data any) ValidationInfo {
	return ValidationInfo{
		StructName:          getStructName(data),
		FullTypeName:        getFullTypeName(data),
		HasGeneratedCode:    vd.hasGeneratedValidation(data),
		HasValidationTags:   vd.hasValidationTags(data),
		RecommendedStrategy: vd.DetectStrategy(data),
		ImplementsInterface: func() bool {
			_, ok := data.(GeneratedValidator)
			return ok
		}(),
	}
}

// ValidationInfo contains information about a struct's validation capabilities
type ValidationInfo struct {
	StructName          string
	FullTypeName        string
	HasGeneratedCode    bool
	HasValidationTags   bool
	RecommendedStrategy Strategy
	ImplementsInterface bool
}

// String returns a human-readable description of the validation info
func (vi ValidationInfo) String() string {
	strategy := ""
	switch vi.RecommendedStrategy {
	case StrategyGenerated:
		strategy = "generated"
	case StrategyReflection:
		strategy = "reflection"
	case StrategyFast:
		strategy = "fast"
	default:
		strategy = "auto"
	}

	features := []string{}
	if vi.HasGeneratedCode {
		features = append(features, "generated-code")
	}
	if vi.HasValidationTags {
		features = append(features, "validation-tags")
	}
	if vi.ImplementsInterface {
		features = append(features, "interface")
	}

	if len(features) == 0 {
		return vi.StructName + " (strategy: " + strategy + ", no validation features)"
	}

	return vi.StructName + " (strategy: " + strategy + ", features: " + strings.Join(features, ",") + ")"
}
