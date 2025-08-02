package rules

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// RuleFactory creates a rule instance from a rule string
type RuleFactory[T any] func(ruleString string) (Validator[T], error)

// Validator represents a typed validation rule
type Validator[T any] interface {
	Validate(field string, value T) error
	String() string // For debugging/caching
}

// RuleRegistry manages rule factories with lazy loading and caching
type RuleRegistry struct {
	factories   sync.Map // map[string]RuleFactory[any]
	ruleCache   sync.Map // map[string]Validator[any] - parsed rule cache
	ruleGroups  sync.Map // map[RuleGroup][]string - rule name groupings
}

// Global registry instance
var GlobalRegistry = &RuleRegistry{}

// RegisterRule registers a rule factory for a specific type
func RegisterRule[T any](name string, factory RuleFactory[T]) {
	// Type-erase the factory to store in sync.Map
	GlobalRegistry.factories.Store(name, func(ruleString string) (any, error) {
		return factory(ruleString)
	})
}

// GetRule retrieves and caches a parsed rule instance
func GetRule[T any](name, ruleString string) (Validator[T], error) {
	cacheKey := name + ":" + ruleString
	
	// Check cache first
	if cached, exists := GlobalRegistry.ruleCache.Load(cacheKey); exists {
		if validator, ok := cached.(Validator[T]); ok {
			return validator, nil
		}
	}
	
	// Get factory
	factoryRaw, exists := GlobalRegistry.factories.Load(name)
	if !exists {
		return nil, fmt.Errorf("rule '%s' not registered", name)
	}
	
	factory := factoryRaw.(func(string) (any, error))
	
	// Create rule instance
	validatorRaw, err := factory(ruleString)
	if err != nil {
		return nil, err
	}
	
	validator, ok := validatorRaw.(Validator[T])
	if !ok {
		return nil, fmt.Errorf("type mismatch for rule '%s'", name)
	}
	
	// Cache the result
	GlobalRegistry.ruleCache.Store(cacheKey, validator)
	
	return validator, nil
}

// ParseRuleString efficiently parses rule strings like "range=1:20" or "minlen=5"
func ParseRuleString(ruleString string) (name string, params string, err error) {
	if ruleString == "" {
		return "", "", fmt.Errorf("empty rule string")
	}
	
	// Efficient parsing without regex
	parts := strings.SplitN(ruleString, "=", 2)
	if len(parts) == 1 {
		return parts[0], "", nil // Rules without parameters like "required"
	}
	
	return parts[0], parts[1], nil
}

// ParseRangeParams parses range parameters like "1:20" or "1,20"
func ParseRangeParams(params string) (min, max int64, err error) {
	if params == "" {
		return 0, 0, fmt.Errorf("missing range parameters")
	}
	
	var parts []string
	if strings.Contains(params, ":") {
		parts = strings.SplitN(params, ":", 2)
	} else if strings.Contains(params, ",") {
		parts = strings.SplitN(params, ",", 2)
	} else {
		return 0, 0, fmt.Errorf("invalid range format: %s", params)
	}
	
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format: %s", params)
	}
	
	min, err = strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid min value: %s", parts[0])
	}
	
	max, err = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid max value: %s", parts[1])
	}
	
	if min > max {
		return 0, 0, fmt.Errorf("min (%d) cannot be greater than max (%d)", min, max)
	}
	
	return min, max, nil
}

// RuleGroup defines logical groupings of related rules
type RuleGroup string

const (
	GroupNumeric     RuleGroup = "numeric"
	GroupString      RuleGroup = "string" 
	GroupFormat      RuleGroup = "format"
	GroupComparison  RuleGroup = "comparison"
	GroupCollection  RuleGroup = "collection"
)

// RegisterRuleGroup associates a rule with a logical group
func RegisterRuleGroup(group RuleGroup, ruleName string) {
	if rules, exists := GlobalRegistry.ruleGroups.Load(group); exists {
		ruleList := rules.([]string)
		GlobalRegistry.ruleGroups.Store(group, append(ruleList, ruleName))
	} else {
		GlobalRegistry.ruleGroups.Store(group, []string{ruleName})
	}
}

// GetRuleGroup returns all rules in a specific group
func GetRuleGroup(group RuleGroup) []string {
	if rules, exists := GlobalRegistry.ruleGroups.Load(group); exists {
		return rules.([]string)
	}
	return nil
}