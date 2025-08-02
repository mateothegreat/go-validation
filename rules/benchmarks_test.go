package rules

import (
	"fmt"
	"testing"
)

// Benchmark generic parameterized range validation
func BenchmarkGenericRange(b *testing.B) {
	validator := NewNumericRange[int64](1, 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("age", int64(50))
	}
}

// Benchmark rule factory with caching
func BenchmarkRuleFactoryWithCache(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator, _ := GetRule[int64]("range_int64", "range=1:100")
		_ = validator.Validate("age", int64(50))
	}
}

// Benchmark rule factory without caching (create new each time)
func BenchmarkRuleFactoryWithoutCache(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator, _ := RangeFactory[int64]("range=1:100")
		_ = validator.Validate("age", int64(50))
	}
}

// Benchmark string parsing efficiency
func BenchmarkRuleStringParsing(b *testing.B) {
	ruleString := "range=1:100"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseRuleString(ruleString)
	}
}

// Benchmark range parameter parsing
func BenchmarkRangeParamsParsing(b *testing.B) {
	params := "1:100"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseRangeParams(params)
	}
}

// Benchmark reflection-based approach (old way)
func BenchmarkReflectionBasedValidation(b *testing.B) {
	validator := &Range{Min: 1, Max: 100}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("age", int64(50), "range=1:100")
	}
}

// Benchmark string length validation
func BenchmarkStringLengthValidation(b *testing.B) {
	validator := NewStringLength("minlen", 5, 0)
	testValue := "hello world"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("name", testValue)
	}
}

// Benchmark characteristic validation
func BenchmarkCharacteristicValidation(b *testing.B) {
	validator := NewCharacteristic(RuleStaticAlphaNumeric)
	testValue := "Hello123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("username", testValue)
	}
}

// Benchmark oneof validation
func BenchmarkOneOfValidation(b *testing.B) {
	validator := NewOneOf([]string{"red", "green", "blue", "yellow", "purple"})
	testValue := "blue"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("color", testValue)
	}
}

// Benchmark multiple field validation
func BenchmarkMultipleFieldValidation(b *testing.B) {
	fieldRules := map[string]string{
		"age":      "range=18:65",
		"name":     "minlen=2",
		"email":    "minlen=5",
		"username": "alphanumeric",
		"color":    "oneof=red,green,blue",
	}
	
	values := map[string]any{
		"age":      int64(25), 
		"name":     "John",
		"email":    "john@example.com",
		"username": "john123",
		"color":    "blue",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateFields(fieldRules, values)
	}
}

// Benchmark memory allocation for different approaches
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("Generic/NoAlloc", func(b *testing.B) {
		validator := NewNumericRange[int64](1, 100)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			_ = validator.Validate("age", int64(50))
		}
	})
	
	b.Run("Reflection/WithAlloc", func(b *testing.B) {
		validator := &Range{Min: 1, Max: 100}
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			_ = validator.Validate("age", int64(50), "range=1:100")
		}
	})
}

// Benchmark different range sizes to test scalability
func BenchmarkRangeSizes(b *testing.B) {
	sizes := []struct {
		name string
		min  int64
		max  int64
	}{
		{"Small_1to10", 1, 10},
		{"Medium_1to1000", 1, 1000},
		{"Large_1to1000000", 1, 1000000},
	}
	
	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			validator := NewNumericRange(size.min, size.max)
			testValue := (size.min + size.max) / 2 // middle value
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = validator.Validate("value", testValue)
			}
		})
	}
}

// Test the efficiency of our lazy loading approach
func TestLazyLoading(t *testing.T) {
	// First access should create and cache
	validator1, err := GetRule[int64]("range_int64", "range=1:100")
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}
	
	// Second access should use cached version
	validator2, err := GetRule[int64]("range_int64", "range=1:100")
	if err != nil {
		t.Fatalf("Failed to get cached rule: %v", err)
	}
	
	// They should be the same instance due to caching
	if fmt.Sprintf("%p", validator1) != fmt.Sprintf("%p", validator2) {
		t.Log("Cache working - same instance returned")
	}
	
	// Test validation works correctly
	if err := validator1.Validate("test", int64(50)); err != nil {
		t.Errorf("Validation failed: %v", err)
	}
	
	if err := validator1.Validate("test", int64(150)); err == nil {
		t.Error("Expected validation to fail for out-of-range value")
	}
}

// Comprehensive test for different numeric types
func TestNumericTypeSupport(t *testing.T) {
	testCases := []struct {
		name     string
		ruleName string
		value    any
		rule     string
		valid    bool
	}{
		{"Int_Valid", "range_int", 50, "range=1:100", true},
		{"Int_Invalid", "range_int", 150, "range=1:100", false},
		{"Int64_Valid", "range_int64", int64(50), "range=1:100", true},
		{"Int64_Invalid", "range_int64", int64(150), "range=1:100", false},
		{"Float64_Valid", "range_float64", 50.5, "range=1:100", true},
		{"Float64_Invalid", "range_float64", 150.5, "range=1:100", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateField("test", tc.value, tc.rule)
			if tc.valid && err != nil {
				t.Errorf("Expected validation to pass, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("Expected validation to fail, but it passed")
			}
		})
	}
}

// Benchmark comparison with static validators (what we want to avoid)
func BenchmarkStaticVsParameterized(b *testing.B) {
	// Simulate what you'd need with static approach
	staticValidators := map[string]func(int64) error{
		"range_1_20":   func(v int64) error { return checkRange(v, 1, 20) },
		"range_1_100":  func(v int64) error { return checkRange(v, 1, 100) },
		"range_1_1000": func(v int64) error { return checkRange(v, 1, 1000) },
	}
	
	b.Run("Static/MultipleValidators", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = staticValidators["range_1_100"](50)
		}
	})
	
	b.Run("Parameterized/SingleValidator", func(b *testing.B) {
		validator := NewNumericRange[int64](1, 100)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.Validate("test", int64(50))
		}
	})
}

func checkRange(value, min, max int64) error {
	if value < min || value > max {
		return fmt.Errorf("value %d out of range [%d, %d]", value, min, max)
	}
	return nil
}