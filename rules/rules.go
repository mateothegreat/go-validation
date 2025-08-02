// Package rules - This file contains the main rule configurations.
package rules

// RuleOperator defines the operator of the rule.
type RuleOperator string

// RuleOperator defines the operation that the rule takes.
const (
	RuleFunctionRequired RuleOperator = "required"
	RuleKeywordMin       RuleOperator = "min"
	RuleKeywordMax       RuleOperator = "max"
	RuleFunctionMinLen   RuleOperator = "minlen"
	RuleFunctionMaxLen   RuleOperator = "maxlen"
	RuleFunctionLen      RuleOperator = "len"
	RuleFunctionRegex    RuleOperator = "regex"
	RuleFunctionOneOf    RuleOperator = "oneof"
	RuleFunctionRange    RuleOperator = "range"
)

// RuleCharacteristic is the characteristic of the rule.
type RuleCharacteristic string

// RuleCharacteristic defines what kind of value the rule is applied against.
const (
	RuleStaticAlpha        RuleCharacteristic = "alpha"
	RuleStaticAlphaNumeric RuleCharacteristic = "alphanumeric"
	RuleStaticNumeric      RuleCharacteristic = "numeric"
	RuleStaticEmail        RuleCharacteristic = "email"
	RuleStaticURL          RuleCharacteristic = "url"
	RuleStaticIP           RuleCharacteristic = "ip"
	RuleStaticUUID         RuleCharacteristic = "uuid"
	RuleStaticCreditCard   RuleCharacteristic = "creditcard"
	RuleStaticPhone        RuleCharacteristic = "phone"
)

// RuleDefinition is the definition of the rule.
type RuleDefinition[T any] struct {
	DataTypes       []DataType
	Operators       []RuleOperator
	Characteristics []RuleCharacteristic
}

func NewRuleDefinition[T any](dataTypes []DataType, operators []RuleOperator, characteristics []RuleCharacteristic) RuleDefinition[T] {
	return RuleDefinition[T]{
		DataTypes:       dataTypes,
		Operators:       operators,
		Characteristics: characteristics,
	}
}

// Rule is the interface that wraps the Validate method.
type Rule interface {
	Validate(field string, value any, rule string) error
}
