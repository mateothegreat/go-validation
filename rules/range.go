package rules

import (
	"fmt"
	"reflect"
)

// NumericRange is a generic, parameterized range validator that works with any numeric type
type NumericRange[T NumericType] struct {
	min T
	max T
}

// NumericType constraint for all numeric types
type NumericType interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
	~float32 | ~float64
}

// NewNumericRange creates a parameterized range validator
func NewNumericRange[T NumericType](min, max T) *NumericRange[T] {
	return &NumericRange[T]{min: min, max: max}
}

// Validate validates that the value is within the specified range
func (r *NumericRange[T]) Validate(field string, value T) error {
	if value < r.min || value > r.max {
		return fmt.Errorf("field '%s' with value %v is not within range [%v, %v]", 
			field, value, r.min, r.max)
	}
	return nil
}

// String returns a string representation for caching and debugging
func (r *NumericRange[T]) String() string {
	return fmt.Sprintf("range[%v:%v]", r.min, r.max)
}

// RangeFactory creates range validators from rule strings like "range=1:20"
func RangeFactory[T NumericType](ruleString string) (Validator[T], error) {
	_, params, err := ParseRuleString(ruleString)
	if err != nil {
		return nil, err
	}
	
	minInt64, maxInt64, err := ParseRangeParams(params)
	if err != nil {
		return nil, err
	}
	
	// Convert to the target numeric type
	min := T(minInt64)
	max := T(maxInt64)
	
	// Validate the conversion didn't overflow
	if int64(min) != minInt64 || int64(max) != maxInt64 {
		return nil, fmt.Errorf("range values %d:%d don't fit in type %s", 
			minInt64, maxInt64, reflect.TypeOf(min).Name())
	}
	
	return NewNumericRange(min, max), nil
}

// Legacy Range struct for backward compatibility
type Range struct {
	Min int
	Max int64
}

func (r *Range) Validate(field string, value any, rule string) error {
	// Convert any to numeric and validate
	switch v := value.(type) {
	case int:
		return NewNumericRange(r.Min, int(r.Max)).Validate(field, v)
	case int64:
		return NewNumericRange(int64(r.Min), r.Max).Validate(field, v)
	case float64:
		return NewNumericRange(float64(r.Min), float64(r.Max)).Validate(field, v)
	default:
		return fmt.Errorf("field '%s': unsupported type %T for range validation", field, value)
	}
}

func NewRange(min int, max int64) *Range {
	return &Range{Min: min, Max: max}
}
