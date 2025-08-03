package validation

import (
	"fmt"
	"testing"

	bench "github.com/mateothegreat/go-bench"
	"github.com/mateothegreat/go-validation/rules"
)

// Comprehensive validation-specific benchmark suite using the abstractable framework

// BenchmarkValidationSuite executes all validation benchmarks using the framework
func BenchmarkValidationSuite(b *testing.B) {
	// Create comprehensive validation benchmark suite
	suite := createValidationBenchmarkSuite()
	runner := bench.NewBenchmarkRunner(suite)

	b.Run("Standard", runner.RunStandardBenchmarks)
	b.Run("Scaling", runner.RunScalingBenchmarks)
	b.Run("Concurrency", runner.RunConcurrencyBenchmarks)
	b.Run("Memory", runner.RunMemoryProfilingBenchmarks)
}

// createValidationBenchmarkSuite builds a comprehensive benchmark suite for validation
func createValidationBenchmarkSuite() *bench.BenchmarkSuite {
	suite := bench.NewBenchmarkTable("ValidationLibrary").
		// Numeric Range Validators
		WithCase("Range_Int_Valid", wrapRangeIntValidator(1, 100), 50).
		WithCaseExpectingError("Range_Int_Invalid", wrapRangeIntValidator(1, 100), 150).
		WithCase("Range_Int64_Valid", wrapRangeInt64Validator(1, 100), int64(50)).
		WithCaseExpectingError("Range_Int64_Invalid", wrapRangeInt64Validator(1, 100), int64(150)).
		WithCase("Range_Float64_Valid", wrapRangeFloat64Validator(0.0, 100.0), 50.5).
		WithCaseExpectingError("Range_Float64_Invalid", wrapRangeFloat64Validator(0.0, 100.0), 150.5).

		// String Length Validators
		WithCase("StringLength_MinLen_Valid", wrapStringLengthValidator("minlen", 5, 0), "hello world").
		WithCaseExpectingError("StringLength_MinLen_Invalid", wrapStringLengthValidator("minlen", 5, 0), "hi").
		WithCase("StringLength_MaxLen_Valid", wrapStringLengthValidator("maxlen", 0, 10), "hello").
		WithCaseExpectingError("StringLength_MaxLen_Invalid", wrapStringLengthValidator("maxlen", 0, 10), "this is too long").

		// String Characteristic Validators
		WithCase("Characteristic_Alpha_Valid", wrapCharacteristicValidator(rules.RuleStaticAlpha), "hello").
		WithCaseExpectingError("Characteristic_Alpha_Invalid", wrapCharacteristicValidator(rules.RuleStaticAlpha), "hello123").
		WithCase("Characteristic_AlphaNumeric_Valid", wrapCharacteristicValidator(rules.RuleStaticAlphaNumeric), "hello123").
		WithCaseExpectingError("Characteristic_AlphaNumeric_Invalid", wrapCharacteristicValidator(rules.RuleStaticAlphaNumeric), "hello@123").
		WithCase("Characteristic_Numeric_Valid", wrapCharacteristicValidator(rules.RuleStaticNumeric), "12345").
		WithCaseExpectingError("Characteristic_Numeric_Invalid", wrapCharacteristicValidator(rules.RuleStaticNumeric), "123abc").

		// OneOf Validators
		WithCase("OneOf_Valid", wrapOneOfValidator([]string{"red", "green", "blue"}), "blue").
		WithCaseExpectingError("OneOf_Invalid", wrapOneOfValidator([]string{"red", "green", "blue"}), "yellow").

		// Factory Pattern Benchmarks
		WithCase("Factory_Range_Cached", wrapFactoryRangeValidator("range=1:100"), int64(50)).
		WithCase("Factory_StringLen_Cached", wrapFactoryStringValidator("minlen=5"), "hello world").

		// Multi-field validation
		WithCase("MultiField_5Fields", wrapMultiFieldValidator(5), generateFieldData(5)).
		WithCase("MultiField_10Fields", wrapMultiFieldValidator(10), generateFieldData(10)).
		WithCase("MultiField_20Fields", wrapMultiFieldValidator(20), generateFieldData(20)).

		// Add input size scaling for performance analysis
		WithInputSizeScaling().

		// Add concurrency testing
		WithConcurrency(1, 2, 4, 8).

		// Set performance thresholds for regression detection
		WithThreshold("Range_Int_Valid", 5.0, 0). // 5ns max, 0 allocs
		WithThreshold("Range_Int64_Valid", 5.0, 0).
		WithThreshold("Range_Float64_Valid", 5.0, 0).
		WithThreshold("StringLength_MinLen_Valid", 10.0, 0).  // 10ns max for string ops
		WithThreshold("Characteristic_Alpha_Valid", 50.0, 0). // Character iteration is slower
		WithThreshold("OneOf_Valid", 20.0, 0).
		WithThreshold("Factory_Range_Cached", 50.0, 1). // Factory allows 1 alloc for lookup
		WithThreshold("MultiField_5Fields", 500.0, 10). // Multi-field validation is more complex

		Build()

	// Add scaling test data for different input sizes
	suite.AddScalingDimension(bench.ScalingDimension{
		Name:   "StringLength",
		Values: []interface{}{10},
	})

	suite.AddScalingDimension(bench.ScalingDimension{
		Name:   "OneOfChoices",
		Values: []interface{}{5},
	})

	return suite
}

// Wrapper functions to adapt validation functions to TestableFunction interface

func wrapRangeIntValidator(min, max int) bench.TestableFunction {
	validator := rules.NewNumericRange[int](min, max)
	return func(args ...interface{}) error {
		if len(args) == 0 {
			return fmt.Errorf("missing argument")
		}
		value, ok := args[0].(int)
		if !ok {
			return fmt.Errorf("invalid argument type")
		}
		return validator.Validate("test", value)
	}
}

func wrapRangeInt64Validator(min, max int64) bench.TestableFunction {
	validator := rules.NewNumericRange[int64](min, max)
	return func(args ...interface{}) error {
		if len(args) == 0 {
			return fmt.Errorf("missing argument")
		}
		value, ok := args[0].(int64)
		if !ok {
			return fmt.Errorf("invalid argument type")
		}
		return validator.Validate("test", value)
	}
}

func wrapRangeFloat64Validator(min, max float64) bench.TestableFunction {
	validator := rules.NewNumericRange[float64](min, max)
	return func(args ...interface{}) error {
		if len(args) == 0 {
			return fmt.Errorf("missing argument")
		}
		value, ok := args[0].(float64)
		if !ok {
			return fmt.Errorf("invalid argument type")
		}
		return validator.Validate("test", value)
	}
}

func wrapStringLengthValidator(op string, min, max int) bench.TestableFunction {
	validator := rules.NewStringLength(op, min, max)
	return func(args ...interface{}) error {
		if len(args) == 0 {
			return fmt.Errorf("missing argument")
		}
		value, ok := args[0].(string)
		if !ok {
			return fmt.Errorf("invalid argument type")
		}
		return validator.Validate("test", value)
	}
}

func wrapCharacteristicValidator(characteristic rules.RuleCharacteristic) bench.TestableFunction {
	validator := rules.NewCharacteristic(characteristic)
	return func(args ...interface{}) error {
		if len(args) == 0 {
			return fmt.Errorf("missing argument")
		}
		value, ok := args[0].(string)
		if !ok {
			return fmt.Errorf("invalid argument type")
		}
		return validator.Validate("test", value)
	}
}

func wrapOneOfValidator(choices []string) bench.TestableFunction {
	validator := rules.NewOneOf(choices)
	return func(args ...interface{}) error {
		if len(args) == 0 {
			return fmt.Errorf("missing argument")
		}
		value, ok := args[0].(string)
		if !ok {
			return fmt.Errorf("invalid argument type")
		}
		return validator.Validate("test", value)
	}
}

func wrapFactoryRangeValidator(ruleString string) bench.TestableFunction {
	return func(args ...interface{}) error {
		validator, err := rules.GetRule[int64]("range_int64", ruleString)
		if err != nil {
			return err
		}
		if len(args) == 0 {
			return fmt.Errorf("missing argument")
		}
		value, ok := args[0].(int64)
		if !ok {
			return fmt.Errorf("invalid argument type")
		}
		return validator.Validate("test", value)
	}
}

func wrapFactoryStringValidator(ruleString string) bench.TestableFunction {
	return func(args ...interface{}) error {
		validator, err := rules.GetRule[string]("minlen", ruleString)
		if err != nil {
			return err
		}
		if len(args) == 0 {
			return fmt.Errorf("missing argument")
		}
		value, ok := args[0].(string)
		if !ok {
			return fmt.Errorf("invalid argument type")
		}
		return validator.Validate("test", value)
	}
}

func wrapMultiFieldValidator(fieldCount int) bench.TestableFunction {
	return func(args ...interface{}) error {
		if len(args) == 0 {
			return fmt.Errorf("missing argument")
		}
		data, ok := args[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid argument type")
		}

		// Generate field rules
		fieldRules := make(map[string]string)
		for i := 0; i < fieldCount; i++ {
			fieldRules[fmt.Sprintf("field%d", i)] = "range=1:100"
		}

		errs := rules.ValidateFields(fieldRules, data)
		if len(errs) > 0 {
			return errs[0] // Return first error for consistency
		}
		return nil
	}
}

// generateFieldData creates test data for multi-field validation benchmarks
func generateFieldData(fieldCount int) map[string]interface{} {
	data := make(map[string]interface{})
	for i := 0; i < fieldCount; i++ {
		data[fmt.Sprintf("field%d", i)] = int64(50) // Valid value within range 1:100
	}
	return data
}

// BenchmarkScalingAnalysis demonstrates scaling analysis across input sizes
func BenchmarkScalingAnalysis(b *testing.B) {
	inputSizes := []int{10, 100, 1000, 10000}
	generator := &bench.DataGenerator{}

	for _, size := range inputSizes {
		// Test string validation scaling
		b.Run(fmt.Sprintf("StringValidation_Size%d", size), func(b *testing.B) {
			validator := rules.NewCharacteristic(rules.RuleStaticAlphaNumeric)
			testStrings := generator.GenerateStrings(size, 20)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for _, str := range testStrings {
					_ = validator.Validate("test", str)
				}
			}
		})

		// Test numeric validation scaling
		b.Run(fmt.Sprintf("NumericValidation_Size%d", size), func(b *testing.B) {
			validator := rules.NewNumericRange[int64](1, 1000000)
			testNumbers := generator.GenerateInts(size, 1, 1000000)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for _, num := range testNumbers {
					_ = validator.Validate("test", num)
				}
			}
		})
	}
}

// BenchmarkMemoryProfile focuses specifically on memory allocation patterns
func BenchmarkMemoryProfile(b *testing.B) {
	b.Run("ZeroAlloc_DirectValidation", func(b *testing.B) {
		validator := rules.NewNumericRange[int64](1, 100)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = validator.Validate("test", int64(50))
		}
	})

	b.Run("MinimalAlloc_FactoryWithCache", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			validator, _ := rules.GetRule[int64]("range_int64", "range=1:100")
			_ = validator.Validate("test", int64(50))
		}
	})

	b.Run("HighAlloc_FactoryWithoutCache", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			validator, _ := rules.RangeFactory[int64]("range=1:100")
			_ = validator.Validate("test", int64(50))
		}
	})
}

// BenchmarkRegressionDetection demonstrates the regression detection capabilities
func BenchmarkRegressionDetection(b *testing.B) {
	// This would typically load baseline results from a file
	// For demo purposes, we'll create mock baseline data
	baseline := []bench.BenchmarkResult{
		{
			Name:        "Range_Int_Valid",
			NsPerOp:     2.0,
			AllocsPerOp: 0,
		},
		{
			Name:        "StringLength_MinLen_Valid",
			NsPerOp:     8.0,
			AllocsPerOp: 0,
		},
	}

	// Simulate current benchmark results
	current := []bench.BenchmarkResult{
		{
			Name:        "Range_Int_Valid",
			NsPerOp:     2.5, // 25% slower - should trigger regression
			AllocsPerOp: 0,
		},
		{
			Name:        "StringLength_MinLen_Valid",
			NsPerOp:     8.2, // 2.5% slower - within tolerance
			AllocsPerOp: 0,
		},
	}

	// Analyze for regressions with 10% tolerance
	regressions := bench.CompareResults(baseline, current, 10.0)

	if len(regressions) > 0 {
		for _, regression := range regressions {
			b.Logf("Regression detected in %s: %.1f%% slower",
				regression.Name, regression.TimeDiffPercent)
		}
	}
}

// Example of how other libraries could use this framework
func BenchmarkOtherLibraryExample(b *testing.B) {
	// Example: HTTP client library benchmarks
	suite := bench.NewBenchmarkTable("HTTPClient").
		WithCase("GET_Request", mockHTTPGet, "https://api.example.com/users").
		WithCase("POST_Request", mockHTTPPost, "https://api.example.com/users", "{\"name\":\"test\"}").
		WithScaling("PayloadSize", 1024, 10240, 102400, 1048576). // 1KB to 1MB
		WithConcurrency(1, 10, 100).
		WithThreshold("GET_Request", 50000000.0, 5). // 50ms max, 5 allocs max
		Build()

	runner := bench.NewBenchmarkRunner(suite)
	runner.RunStandardBenchmarks(b)
}

// Mock functions for example
func mockHTTPGet(args ...interface{}) error {
	// Simulate HTTP GET operation
	return nil
}

func mockHTTPPost(args ...interface{}) error {
	// Simulate HTTP POST operation
	return nil
}
