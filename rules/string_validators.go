package rules

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// StringLengthValidator validates string length constraints
type StringLengthValidator struct {
	min int
	max int
	op  string // "min", "max", "len", "range"
}

// NewStringLength creates parameterized string length validators
func NewStringLength(op string, min, max int) *StringLengthValidator {
	return &StringLengthValidator{min: min, max: max, op: op}
}

func (v *StringLengthValidator) Validate(field string, value string) error {
	length := len(value)
	
	switch v.op {
	case "minlen":
		if length < v.min {
			return fmt.Errorf("field '%s' length %d is less than minimum %d", field, length, v.min)
		}
	case "maxlen":
		if length > v.max {
			return fmt.Errorf("field '%s' length %d exceeds maximum %d", field, length, v.max)
		}
	case "len":
		if length != v.min {
			return fmt.Errorf("field '%s' length %d does not equal required %d", field, length, v.min)
		}
	case "range":
		if length < v.min || length > v.max {
			return fmt.Errorf("field '%s' length %d is not within range [%d, %d]", field, length, v.min, v.max)
		}
	}
	return nil
}

func (v *StringLengthValidator) String() string {
	return fmt.Sprintf("%s[%d:%d]", v.op, v.min, v.max)
}

// StringLengthFactory creates string length validators from rule strings
func StringLengthFactory(ruleString string) (Validator[string], error) {
	name, params, err := ParseRuleString(ruleString)
	if err != nil {
		return nil, err
	}
	
	switch name {
	case "minlen", "maxlen":
		val, err := strconv.Atoi(params)
		if err != nil {
			return nil, fmt.Errorf("invalid %s parameter: %s", name, params)
		}
		if name == "minlen" {
			return NewStringLength("minlen", val, 0), nil
		}
		return NewStringLength("maxlen", 0, val), nil
		
	case "len":
		val, err := strconv.Atoi(params)
		if err != nil {
			return nil, fmt.Errorf("invalid len parameter: %s", params)
		}
		return NewStringLength("len", val, 0), nil
		
	case "lenrange":
		min, max, err := ParseRangeParams(params)
		if err != nil {
			return nil, err
		}
		return NewStringLength("range", int(min), int(max)), nil
	}
	
	return nil, fmt.Errorf("unknown string length rule: %s", name)
}

// CharacteristicValidator validates string characteristics
type CharacteristicValidator struct {
	characteristic RuleCharacteristic
}

func NewCharacteristic(characteristic RuleCharacteristic) *CharacteristicValidator {
	return &CharacteristicValidator{characteristic: characteristic}
}

func (v *CharacteristicValidator) Validate(field string, value string) error {
	switch v.characteristic {
	case RuleStaticAlpha:
		for _, r := range value {
			if !unicode.IsLetter(r) {
				return fmt.Errorf("field '%s' contains non-alphabetic character: %c", field, r)
			}
		}
	case RuleStaticAlphaNumeric:
		for _, r := range value {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return fmt.Errorf("field '%s' contains non-alphanumeric character: %c", field, r)
			}
		}
	case RuleStaticNumeric:
		for _, r := range value {
			if !unicode.IsDigit(r) {
				return fmt.Errorf("field '%s' contains non-numeric character: %c", field, r)
			}
		}
	}
	return nil
}

func (v *CharacteristicValidator) String() string {
	return string(v.characteristic)
}

// CharacteristicFactory creates characteristic validators
func CharacteristicFactory(ruleString string) (Validator[string], error) {
	name, _, err := ParseRuleString(ruleString)
	if err != nil {
		return nil, err
	}
	
	characteristic := RuleCharacteristic(name)
	switch characteristic {
	case RuleStaticAlpha, RuleStaticAlphaNumeric, RuleStaticNumeric:
		return NewCharacteristic(characteristic), nil
	default:
		return nil, fmt.Errorf("unknown characteristic: %s", name)
	}
}

// OneOfValidator validates that a value is one of the allowed values
type OneOfValidator struct {
	allowedValues []string
}

func NewOneOf(values []string) *OneOfValidator {
	return &OneOfValidator{allowedValues: values}
}

func (v *OneOfValidator) Validate(field string, value string) error {
	for _, allowed := range v.allowedValues {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("field '%s' value '%s' is not one of allowed values: %v", 
		field, value, v.allowedValues)
}

func (v *OneOfValidator) String() string {
	return fmt.Sprintf("oneof[%s]", strings.Join(v.allowedValues, ","))
}

// OneOfFactory creates oneof validators from rule strings like "oneof=red,green,blue"
func OneOfFactory(ruleString string) (Validator[string], error) {
	_, params, err := ParseRuleString(ruleString)
	if err != nil {
		return nil, err
	}
	
	if params == "" {
		return nil, fmt.Errorf("oneof rule requires parameters")
	}
	
	values := strings.Split(params, ",")
	for i, v := range values {
		values[i] = strings.TrimSpace(v)
	}
	
	return NewOneOf(values), nil
}