package rules

import (
	"fmt"
	"reflect"
)

// init registers all built-in rules with their factories and groups
func init() {
	// Register numeric range rules
	RegisterRule("range_int", func(ruleString string) (Validator[int], error) {
		return RangeFactory[int](ruleString)
	})
	RegisterRule("range_int64", func(ruleString string) (Validator[int64], error) {
		return RangeFactory[int64](ruleString)
	})
	RegisterRule("range_float64", func(ruleString string) (Validator[float64], error) {
		return RangeFactory[float64](ruleString)
	})
	
	// Register string length rules
	RegisterRule("minlen", StringLengthFactory)
	RegisterRule("maxlen", StringLengthFactory)
	RegisterRule("len", StringLengthFactory)
	RegisterRule("lenrange", StringLengthFactory)
	
	// Register string characteristic rules
	RegisterRule("alpha", CharacteristicFactory)
	RegisterRule("alphanumeric", CharacteristicFactory)
	RegisterRule("numeric", CharacteristicFactory)
	
	// Register string format rules
	RegisterRule("oneof", OneOfFactory)
	
	// Register rule groups
	RegisterRuleGroup(GroupNumeric, "range_int")
	RegisterRuleGroup(GroupNumeric, "range_int64")
	RegisterRuleGroup(GroupNumeric, "range_float64")
	
	RegisterRuleGroup(GroupString, "minlen")
	RegisterRuleGroup(GroupString, "maxlen")
	RegisterRuleGroup(GroupString, "len")
	RegisterRuleGroup(GroupString, "lenrange")
	RegisterRuleGroup(GroupString, "alpha")
	RegisterRuleGroup(GroupString, "alphanumeric")
	RegisterRuleGroup(GroupString, "numeric")
	
	RegisterRuleGroup(GroupFormat, "oneof")
}

// GetRuleForType automatically selects the appropriate rule factory based on the value type
func GetRuleForType(ruleName string, value any) (string, error) {
	valueType := reflect.TypeOf(value)
	
	switch ruleName {
	case "range":
		switch valueType.Kind() {
		case reflect.Int:
			return "range_int", nil
		case reflect.Int64:
			return "range_int64", nil
		case reflect.Float64:
			return "range_float64", nil
		default:
			return "", fmt.Errorf("range rule not supported for type %s", valueType.String())
		}
	default:
		return ruleName, nil // Use rule as-is for non-polymorphic rules
	}
}

// ValidateField validates a field using the appropriate rule for its type
func ValidateField(field string, value any, ruleString string) error {
	name, _, err := ParseRuleString(ruleString)
	if err != nil {
		return err
	}
	
	// Automatically select the right rule for the type
	actualRuleName, err := GetRuleForType(name, value)
	if err != nil {
		return err
	}
	
	// Type-specific validation
	switch v := value.(type) {
	case int:
		if validator, err := GetRule[int](actualRuleName, ruleString); err == nil {
			return validator.Validate(field, v)
		}
	case int64:
		if validator, err := GetRule[int64](actualRuleName, ruleString); err == nil {
			return validator.Validate(field, v)
		}
	case float64:
		if validator, err := GetRule[float64](actualRuleName, ruleString); err == nil {
			return validator.Validate(field, v)
		}
	case string:
		if validator, err := GetRule[string](actualRuleName, ruleString); err == nil {
			return validator.Validate(field, v)
		}
	}
	
	return fmt.Errorf("no validator found for field '%s' with type %T and rule '%s'", 
		field, value, ruleString)
}

// ValidateFields validates multiple fields with their respective rules
func ValidateFields(fieldRules map[string]string, values map[string]any) []error {
	var errors []error
	
	for field, ruleString := range fieldRules {
		if value, exists := values[field]; exists {
			if err := ValidateField(field, value, ruleString); err != nil {
				errors = append(errors, err)
			}
		}
	}
	
	return errors
}