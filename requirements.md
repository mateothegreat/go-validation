# Requirements

## Problem

for a global/shared validation library package using golang that is goign to be used in highly available and scaled and fault tolerant applications, what is the most idiomatic and efficient way as possible to handle not having to have multiple copies of a validation method like Range(1, 20) and Range(1000,99999) and so on? is it more efficient to declare a single method that accepts arguments to tune the method based on the caller and tghe value or?

## The Baseline Requirements

Core Design: Generic, Parameterized, Lazy-Loaded Validators
minimal requirements (you need to add your own on tiop of these):

- Avoid static duplication — it’s brittle and unscalable.
- Use reflection sparingly and generics where possible (remove reflection at all costs).
- Use parameterized constructors like NewRange(min, max).
- Keep the Rule interface lean. Efficient Rule Parsing (No Regex).
- rules are defined as strings like "range=1:20" and should be parsed using the most efficient means available.
- Keeps Memory Footprint Low (instantiate only what is needed during/throughout runtime dynamically and No static registry of every possible range.)
- provide a common interface for error handling and collection so that the caller can decide how to handle the errors - all validations should run and collect errors and return them to the caller as a slice of errors.
- Use a factory pattern to instantiate rules only when needed
- ensure future proofing for new rules and characteristics is taken in to account at all times. think of how to handle new rules and characteristics without breaking the existing code as the library evolves.
- organize rules by groupings of rules and characteristics such as "alpha" + "alphanumeric" + "numeric" would be in a `RuleGroup` called "AlphaNumeric" and "AlphaNumeric" would be in a `RuleGroup` called "AlphaNumeric".

## Your Instructions

identify optimizations like lazy loading, etc. and include examples that can be benchmarked.

hypothetical example of a rule definition:

```go
package rules

type RuleFactory func(rule string) (Rule, error)

var ruleRegistry = map[string]RuleFactory{}

func RegisterRule(name string, factory RuleFactory) {
    ruleRegistry[name] = factory
}

func GetRule(name, rule string) (Rule, error) {
    factory, ok := ruleRegistry[name]
    if !ok {
        return nil, fmt.Errorf("rule '%s' not registered", name)
    }
    return factory(rule)
}

```

Use a factory pattern to instantiate rules only when needed:

```go
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
```

Register a rule:

```go
func init() {
    RegisterRule("range", func(rule string) (Rule, error) {
        min, max, err := ParseRangeRule(rule)
        if err != nil {
            return nil, err
        }
        return NewRange[int64](min, max), nil
    })
}
```

Benchmark the rule parsing and instantiation:

```go
func BenchmarkGenericRange(b *testing.B) {
    r := NewRange[int64](1, 100)
    for i := 0; i < b.N; i++ {
        _ = r.Validate("age", int64(50), "")
    }
}
```
