package validation

import (
	"reflect"
	"strings"
	"testing"
)

// =============================================================================
// CRITICAL MISSING BENCHMARKS IDENTIFIED FROM REFLECTION ANALYSIS
// These benchmarks address the specific performance bottlenecks identified
// in the reflection usage analysis v2.0
// =============================================================================

// =============================================================================
// BENCHMARK STRUCTURES FOR TESTING
// =============================================================================

// Small struct for testing field count scaling
type SmallBenchStruct struct {
	Field1 string `validate:"required"`
	Field2 int    `validate:"min=1"`
}

// Medium struct for testing field count scaling
type MediumBenchStruct struct {
	Field1  string  `validate:"required,min=2"`
	Field2  int     `validate:"min=1,max=100"`
	Field3  string  `validate:"email"`
	Field4  string  `validate:"url"`
	Field5  bool    `validate:"required"`
	Field6  string  `validate:"oneof=red green blue"`
	Field7  int     `validate:"required"`
	Field8  string  `validate:"alphanum"`
	Field9  float64 `validate:"min=0.1"`
	Field10 string  `validate:"len=5"`
}

// Large struct for testing field count scaling  
type LargeBenchStruct struct {
	F1, F2, F3, F4, F5          string `validate:"required"`
	F6, F7, F8, F9, F10         int    `validate:"min=1"`
	F11, F12, F13, F14, F15     string `validate:"email"`
	F16, F17, F18, F19, F20     string `validate:"url"`
	F21, F22, F23, F24, F25     bool
	F26, F27, F28, F29, F30     string `validate:"oneof=a b c"`
	F31, F32, F33, F34, F35     int    `validate:"max=100"`
	F36, F37, F38, F39, F40     string `validate:"alphanum"`
	F41, F42, F43, F44, F45     float64 `validate:"min=0"`
	F46, F47, F48, F49, F50     string `validate:"len=10"`
}

// Cross-field validation structs
type CrossFieldBenchStruct struct {
	Password        string `validate:"required,min=8"`
	ConfirmPassword string `validate:"required,eqfield=Password"`
	StartDate       string `validate:"required,date"`
	EndDate         string `validate:"required,date,gtfield=StartDate"`
	Age             int    `validate:"required,min=18"`
	ParentEmail     string `validate:"required_if=Age 17,omitempty,email"`
}

// OmitEmpty testing structs
type OmitEmptyBenchStruct struct {
	RequiredField string `validate:"required"`
	OptionalURL   string `validate:"omitempty,url"`
	OptionalEmail string `validate:"omitempty,email"`
	OptionalPhone string `validate:"omitempty,phone"`
	OptionalUUID  string `validate:"omitempty,uuid"`
}

// Nested struct for enhanced testing
type NestedBenchStructWithTags struct {
	BasicInfo ContactBenchInfo `validate:"required"`
	Address   AddressBenchInfo `validate:"required"`
}

type NestedBenchStructWithoutTags struct {
	BasicInfo ContactBenchInfo
	Address   AddressBenchInfo
}

type ContactBenchInfo struct {
	Name  string `validate:"required,min=2"`
	Email string `validate:"required,email"`
	Phone string `validate:"omitempty,phone"`
}

type AddressBenchInfo struct {
	Street  string `validate:"required,min=5"`
	City    string `validate:"required,min=2"`
	Country string `validate:"required,oneof=US CA UK DE FR"`
}

// =============================================================================
// 🔴 CRITICAL MISSING BENCHMARKS - CROSS-FIELD VALIDATION
// These were identified as the primary performance bottleneck (25-70ns per op)
// =============================================================================

func BenchmarkCrossFieldValidation(b *testing.B) {
	validator := New()
	
	// Test data with valid cross-field relationships
	validStruct := CrossFieldBenchStruct{
		Password:        "password123",
		ConfirmPassword: "password123",
		StartDate:       "2023-01-01",
		EndDate:         "2023-12-31",
		Age:             25,
		ParentEmail:     "", // Not required since age >= 18
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(validStruct)
	}
}

func BenchmarkCrossFieldValidation_EqField(b *testing.B) {
	validator := New()
	
	type EqFieldTest struct {
		Password        string `validate:"required"`
		ConfirmPassword string `validate:"eqfield=Password"`
	}
	
	test := EqFieldTest{
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkCrossFieldValidation_GtField(b *testing.B) {
	validator := New()
	
	type GtFieldTest struct {
		StartDate string `validate:"required"`
		EndDate   string `validate:"gtfield=StartDate"`
	}
	
	test := GtFieldTest{
		StartDate: "2023-01-01",
		EndDate:   "2023-12-31",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkCrossFieldValidation_RequiredIf(b *testing.B) {
	validator := New()
	
	type RequiredIfTest struct {
		Age         int    `validate:"required"`
		ParentEmail string `validate:"required_if=Age 17,omitempty,email"`
	}
	
	test := RequiredIfTest{
		Age:         25, // Should not require ParentEmail
		ParentEmail: "",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

// Test with failing cross-field validation to measure error path performance
func BenchmarkCrossFieldValidation_Failures(b *testing.B) {
	validator := New()
	
	// Invalid data that will trigger cross-field validation failures
	invalidStruct := CrossFieldBenchStruct{
		Password:        "password123",
		ConfirmPassword: "different",      // Fails eqfield
		StartDate:       "2023-12-31",
		EndDate:         "2023-01-01",     // Fails gtfield
		Age:             17,               // Triggers required_if
		ParentEmail:     "invalid-email",  // Fails email validation
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(invalidStruct)
	}
}

// =============================================================================
// 🔴 CRITICAL MISSING BENCHMARKS - PARENT CONTEXT OVERHEAD
// New validateField signature affects ALL field validation (5-10% regression)
// =============================================================================

func BenchmarkParentContextOverhead(b *testing.B) {
	validator := New()
	
	// Simple struct to isolate parent context overhead
	type SimpleTest struct {
		Field1 string `validate:"required"`
		Field2 string `validate:"required"`
		Field3 string `validate:"required"`
	}
	
	test := SimpleTest{
		Field1: "value1",
		Field2: "value2",
		Field3: "value3",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkFieldLevelCreation(b *testing.B) {
	validator := New()
	parentValue := reflect.ValueOf(struct{}{})
	fieldValue := reflect.ValueOf("test")
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		fl := &fieldLevel{
			validator: validator,
			top:       parentValue,
			parent:    parentValue,
			field:     fieldValue,
			fieldName: "test",
			param:     "required",
			tag:       "required",
		}
		_ = fl
	}
}

// =============================================================================
// 🔴 CRITICAL MISSING BENCHMARKS - FIELD LOOKUP BY NAME
// Identified as "MOST EXPENSIVE" operation (15-50ns per call)
// =============================================================================

func BenchmarkFieldLookupByName(b *testing.B) {
	// Create a struct with multiple fields to simulate realistic lookup costs
	type TestStruct struct {
		Field1, Field2, Field3, Field4, Field5     string
		Field6, Field7, Field8, Field9, Field10    string
		Field11, Field12, Field13, Field14, Field15 string
	}
	
	val := reflect.ValueOf(TestStruct{})
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Simulate field lookup by name (expensive operation)
		_ = val.FieldByName("Field10") // Middle field for average case
	}
}

func BenchmarkFieldByNameVsIndex(b *testing.B) {
	type TestStruct struct {
		Field1, Field2, Field3, Field4, Field5 string
	}
	
	val := reflect.ValueOf(TestStruct{})
	
	b.Run("ByName", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = val.FieldByName("Field3")
		}
	})
	
	b.Run("ByIndex", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = val.Field(2) // Field3 is at index 2
		}
	})
}

func BenchmarkGetStructFieldOK(b *testing.B) {
	validator := New()
	
	type TestStruct struct {
		TargetField string `validate:"required"`
		OtherField1 string
		OtherField2 string
		OtherField3 string
	}
	
	val := reflect.ValueOf(TestStruct{TargetField: "test"})
	fl := &fieldLevel{
		validator: validator,
		top:       val,
		parent:    val,
		field:     val.Field(0),
		fieldName: "TargetField",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _, _ = fl.getStructFieldOK(val, "TargetField")
	}
}

// =============================================================================
// 🟠 HIGH PRIORITY MISSING BENCHMARKS - ENHANCED NESTED STRUCT VALIDATION
// Enhanced detection now performs 2-3 Kind() calls (15-25% regression)
// =============================================================================

func BenchmarkNestedStructEnhanced(b *testing.B) {
	validator := New()
	
	nested := NestedBenchStructWithTags{
		BasicInfo: ContactBenchInfo{
			Name:  "John Doe",
			Email: "john@example.com",
			Phone: "+1234567890",
		},
		Address: AddressBenchInfo{
			Street:  "123 Main Street",
			City:    "Anytown",
			Country: "US",
		},
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(nested)
	}
}

func BenchmarkNestedStructWithTags(b *testing.B) {
	validator := New()
	
	nested := NestedBenchStructWithTags{
		BasicInfo: ContactBenchInfo{
			Name:  "John Doe",
			Email: "john@example.com",
		},
		Address: AddressBenchInfo{
			Street:  "123 Main St",
			City:    "City",
			Country: "US",
		},
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(nested)
	}
}

func BenchmarkNestedStructWithoutTags(b *testing.B) {
	validator := New()
	
	nested := NestedBenchStructWithoutTags{
		BasicInfo: ContactBenchInfo{
			Name:  "John Doe",
			Email: "john@example.com",
		},
		Address: AddressBenchInfo{
			Street:  "123 Main St",
			City:    "City",
			Country: "US",
		},
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(nested)
	}
}

func BenchmarkPointerToStructDetection(b *testing.B) {
	type PointerStruct struct {
		Info *ContactBenchInfo `validate:"required"`
	}
	
	validator := New()
	test := PointerStruct{
		Info: &ContactBenchInfo{
			Name:  "John Doe",
			Email: "john@example.com",
		},
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

// =============================================================================
// 🟠 HIGH PRIORITY MISSING BENCHMARKS - OMITEMPTY LOGIC
// New logic creates temporary fieldLevel instances (medium impact)
// =============================================================================

func BenchmarkOmitEmptyLogic(b *testing.B) {
	validator := New()
	
	// Test with empty optional fields (should skip validation)
	test := OmitEmptyBenchStruct{
		RequiredField: "present",
		OptionalURL:   "", // Empty - should skip URL validation
		OptionalEmail: "", // Empty - should skip email validation
		OptionalPhone: "", // Empty - should skip phone validation
		OptionalUUID:  "", // Empty - should skip UUID validation
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkOmitEmptyWithValues(b *testing.B) {
	validator := New()
	
	// Test with populated optional fields (should run validation)
	test := OmitEmptyBenchStruct{
		RequiredField: "present",
		OptionalURL:   "https://example.com",
		OptionalEmail: "test@example.com",
		OptionalPhone: "+1234567890",
		OptionalUUID:  "550e8400-e29b-41d4-a716-446655440000",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkHasValueCheck(b *testing.B) {
	validator := New()
	
	// Test HasValue function directly
	emptyValue := reflect.ValueOf("")
	nonEmptyValue := reflect.ValueOf("test")
	
	fl := &fieldLevel{
		validator: validator,
		field:     emptyValue,
		fieldName: "test",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			fl.field = emptyValue
		} else {
			fl.field = nonEmptyValue
		}
		_ = HasValue(fl)
	}
}

// =============================================================================
// 🟡 MEDIUM PRIORITY MISSING BENCHMARKS - BUILT-IN RULES PERFORMANCE
// Individual rule performance measurement
// =============================================================================

func BenchmarkBuiltinRules_Email(b *testing.B) {
	validator := New()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Var("user@example.com", "email")
	}
}

func BenchmarkBuiltinRules_URL(b *testing.B) {
	validator := New()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Var("https://www.example.com", "url")
	}
}

func BenchmarkBuiltinRules_Phone(b *testing.B) {
	validator := New()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Var("+1234567890", "phone")
	}
}

func BenchmarkBuiltinRules_UUID(b *testing.B) {
	validator := New()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Var("550e8400-e29b-41d4-a716-446655440000", "uuid")
	}
}

func BenchmarkBuiltinRules_DateTime(b *testing.B) {
	validator := New()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Var("2023-12-25T10:30:00Z", "datetime")
	}
}

func BenchmarkBuiltinRules_CreditCard(b *testing.B) {
	validator := New()
	
	b.ReportAllocs()  
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Var("4111111111111111", "creditcard")
	}
}

// =============================================================================
// 🟡 MEDIUM PRIORITY MISSING BENCHMARKS - STRUCT SIZE SCALING
// Field iteration is "CRITICAL HOTPATH" but scaling not measured
// =============================================================================

func BenchmarkSmallStruct(b *testing.B) {
	validator := New()
	
	test := SmallBenchStruct{
		Field1: "test",
		Field2: 42,
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkMediumStruct(b *testing.B) {
	validator := New()
	
	test := MediumBenchStruct{
		Field1:  "test",
		Field2:  42,
		Field3:  "user@example.com",
		Field4:  "https://example.com",
		Field5:  true,
		Field6:  "red",
		Field7:  100,
		Field8:  "abc123",
		Field9:  1.5,
		Field10: "12345",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkLargeStruct(b *testing.B) {
	validator := New()
	
	// Create a large struct with all fields populated
	test := LargeBenchStruct{}
	// Populate required string fields
	test.F1, test.F2, test.F3, test.F4, test.F5 = "a", "b", "c", "d", "e"
	// Populate int fields with valid values
	test.F6, test.F7, test.F8, test.F9, test.F10 = 1, 2, 3, 4, 5
	// Populate email fields
	test.F11, test.F12, test.F13, test.F14, test.F15 = "a@b.com", "c@d.com", "e@f.com", "g@h.com", "i@j.com"
	// Populate URL fields
	test.F16, test.F17, test.F18, test.F19, test.F20 = "http://a.com", "http://b.com", "http://c.com", "http://d.com", "http://e.com"
	// Populate oneof fields
	test.F26, test.F27, test.F28, test.F29, test.F30 = "a", "b", "c", "a", "b"
	// Populate max int fields
	test.F31, test.F32, test.F33, test.F34, test.F35 = 50, 60, 70, 80, 90
	// Populate alphanum fields
	test.F36, test.F37, test.F38, test.F39, test.F40 = "abc123", "def456", "ghi789", "jkl012", "mno345"
	// Populate float fields
	test.F41, test.F42, test.F43, test.F44, test.F45 = 1.0, 2.0, 3.0, 4.0, 5.0
	// Populate len fields
	test.F46, test.F47, test.F48, test.F49, test.F50 = "1234567890", "abcdefghij", "0987654321", "jihgfedcba", "qwertyuiop"
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkFieldCountScaling(b *testing.B) {
	validator := New()
	
	b.Run("SmallStruct_2Fields", func(b *testing.B) {
		test := SmallBenchStruct{Field1: "test", Field2: 42}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.Struct(test)
		}
	})
	
	b.Run("MediumStruct_10Fields", func(b *testing.B) {
		test := MediumBenchStruct{
			Field1: "test", Field2: 42, Field3: "user@example.com",
			Field4: "https://example.com", Field5: true, Field6: "red",
			Field7: 100, Field8: "abc123", Field9: 1.5, Field10: "12345",
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.Struct(test)
		}
	})
}

// =============================================================================
// 🟡 MEDIUM PRIORITY MISSING BENCHMARKS - ERROR COLLECTION AND REPORTING
// Error path performance measurement
// =============================================================================

func BenchmarkErrorCollection_Success(b *testing.B) {
	validator := New()
	
	// Valid struct that should not produce errors
	test := CrossFieldBenchStruct{
		Password:        "password123",
		ConfirmPassword: "password123",
		StartDate:       "2023-01-01",
		EndDate:         "2023-12-31",
		Age:             25,
		ParentEmail:     "",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkErrorCollection_Failure(b *testing.B) {
	validator := New()
	
	// Invalid struct that should produce multiple errors
	test := CrossFieldBenchStruct{
		Password:        "123",              // Too short
		ConfirmPassword: "different",        // Doesn't match
		StartDate:       "invalid-date",     // Invalid date
		EndDate:         "2020-01-01",      // Before start date
		Age:             17,                 // Requires ParentEmail
		ParentEmail:     "invalid-email",    // Invalid email
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkErrorCollector_FailFast(b *testing.B) {
	config := ValidatorConfig{
		FailFast: true,
	}
	validator := NewWithConfig(config)
	
	// Invalid struct that should stop at first error
	test := struct {
		Field1 string `validate:"required"`
		Field2 string `validate:"required"`
		Field3 string `validate:"required"`
	}{} // All fields empty
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

// =============================================================================
// 🟡 MEDIUM PRIORITY MISSING BENCHMARKS - MEMORY ALLOCATION PATTERNS
// 20-30% increase in memory usage analysis
// =============================================================================

func BenchmarkMemoryAllocation_CrossField(b *testing.B) {
	validator := New()
	
	test := CrossFieldBenchStruct{
		Password:        "password123",
		ConfirmPassword: "password123",
		StartDate:       "2023-01-01",
		EndDate:         "2023-12-31",
		Age:             25,
		ParentEmail:     "",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

func BenchmarkMemoryAllocation_NestedStruct(b *testing.B) {
	validator := New()
	
	test := NestedBenchStructWithTags{
		BasicInfo: ContactBenchInfo{
			Name:  "John Doe",
			Email: "john@example.com",
		},
		Address: AddressBenchInfo{
			Street:  "123 Main St",
			City:    "City",
			Country: "US",
		},
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

// =============================================================================
// CONSISTENCY IMPROVEMENTS - DATA VARIATION TESTING
// Test different data scenarios to avoid caching effects
// =============================================================================

func BenchmarkValidation_DataVariation(b *testing.B) {
	validator := New()
	
	// Test with different data each iteration to avoid caching effects
	testData := []User{
		{Name: "Alice", Email: "alice@example.com", Age: 25, Password: "password123"},
		{Name: "Bob", Email: "bob@test.com", Age: 30, Password: "securepass"},
		{Name: "Charlie", Email: "charlie@domain.org", Age: 35, Password: "mypassword"},
		{Name: "Diana", Email: "diana@company.net", Age: 28, Password: "strongpass"},
		{Name: "Eve", Email: "eve@site.co", Age: 32, Password: "complexpass"},
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		test := testData[i%len(testData)]
		_ = validator.Struct(test)
	}
}

func BenchmarkValidation_ValidVsInvalid(b *testing.B) {
	validator := New()
	
	validUser := User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Password: "password123",
	}
	
	invalidUser := User{
		Name:     "J",                // Too short
		Email:    "invalid-email",    // Invalid format
		Age:      15,                 // Below minimum
		Password: "123",              // Too short
	}
	
	b.Run("ValidData", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.Struct(validUser)
		}
	})
	
	b.Run("InvalidData", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.Struct(invalidUser)
		}
	})
}

// =============================================================================
// CONSISTENCY IMPROVEMENTS - STANDARDIZED BENCHMARK PATTERNS  
// Standardize validator creation patterns
// =============================================================================

func BenchmarkValidatorReuse_vs_Creation(b *testing.B) {
	test := User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Password: "password123",
	}
	
	b.Run("ReuseValidator", func(b *testing.B) {
		validator := New()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.Struct(test)
		}
	})
	
	b.Run("CreateValidator", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			validator := New()
			_ = validator.Struct(test)
		}
	})
}

// =============================================================================
// REGRESSION DETECTION BENCHMARKS
// Baseline benchmarks for performance regression detection
// =============================================================================

func BenchmarkRegression_SimpleValidation(b *testing.B) {
	// Baseline benchmark for regression detection
	validator := New()
	user := User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Password: "password123",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(user)
	}
}

func BenchmarkRegression_CrossFieldValidation(b *testing.B) {
	// Baseline benchmark for cross-field validation regression detection
	validator := New()
	test := CrossFieldBenchStruct{
		Password:        "password123",
		ConfirmPassword: "password123",
		StartDate:       "2023-01-01",
		EndDate:         "2023-12-31",
		Age:             25,
		ParentEmail:     "",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(test)
	}
}

// =============================================================================
// REFLECTION OPERATION ISOLATION BENCHMARKS
// Isolate individual reflection operations for optimization analysis
// =============================================================================

func BenchmarkReflection_ValueOf(b *testing.B) {
	test := User{Name: "test", Email: "test@example.com", Age: 25, Password: "password"}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = reflect.ValueOf(test)
	}
}

func BenchmarkReflection_KindChecking(b *testing.B) {
	val := reflect.ValueOf(struct{}{})
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = val.Kind() == reflect.Struct
	}
}

func BenchmarkReflection_FieldIteration(b *testing.B) {
	val := reflect.ValueOf(MediumBenchStruct{})
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		numFields := val.NumField()
		for j := 0; j < numFields; j++ {
			_ = val.Field(j)
		}
	}
}

func BenchmarkReflection_TagParsing(b *testing.B) {
	typ := reflect.TypeOf(User{})
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		for j := 0; j < typ.NumField(); j++ {
			field := typ.Field(j)
			_ = field.Tag.Get("validate")
		}
	}
}

// =============================================================================
// HELPER BENCHMARKS FOR OPTIMIZATION VALIDATION
// Test optimization strategies
// =============================================================================

func BenchmarkStringComparison_Performance(b *testing.B) {
	// Test string comparison performance for rule names
	rules := []string{"required", "email", "min", "max", "len", "oneof", "alpha"}
	target := "email"
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		for _, rule := range rules {
			if strings.TrimSpace(rule) == target {
				break
			}
		}
	}
}

func BenchmarkMapLookup_vs_StringComparison(b *testing.B) {
	rules := map[string]bool{
		"required": true,
		"email":    true,
		"min":      true,
		"max":      true,
		"len":      true,
		"oneof":    true,
		"alpha":    true,
	}
	
	b.Run("MapLookup", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = rules["email"]
		}
	})
	
	b.Run("StringComparison", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rule := "email"
			_ = rule == "email"
		}
	})
}