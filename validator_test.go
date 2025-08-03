package validation

import (
	"strings"
	"testing"
)

// Test structures
type User struct {
	Name     string `validate:"required,min=2,max=50"`
	Email    string `validate:"required,email"`
	Age      int    `validate:"required,min=18,max=120"`
	Password string `validate:"required,min=8"`
	Website  string `validate:"omitempty,url"`
	Phone    string `validate:"omitempty,phone"`
}

type Address struct {
	Street   string `validate:"required,min=5"`
	City     string `validate:"required,min=2"`
	PostCode string `validate:"required,len=5"`
	Country  string `validate:"required,oneof=US CA UK"`
}

type UserWithAddress struct {
	User     User              `validate:"required"`
	Address  Address           `validate:"required"`
	Tags     []string          `validate:"dive,required,min=2"`
	Metadata map[string]string `validate:"dive,keys,alpha,endkeys,required"`
}

type CrossFieldTest struct {
	Password        string `validate:"required,min=8"`
	ConfirmPassword string `validate:"required,eqfield=Password"`
	StartDate       string `validate:"required,date"`
	EndDate         string `validate:"required,date,gtfield=StartDate"`
	Age             int    `validate:"required,min=18"`
	ParentEmail     string `validate:"required_if=Age 17,omitempty,email"`
}

func TestValidatorBasicValidation(t *testing.T) {
	validator := New()

	tests := []struct {
		name      string
		input     interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid user",
			input: User{
				Name:     "John Doe",
				Email:    "john@example.com",
				Age:      25,
				Password: "password123",
				Website:  "https://example.com",
				Phone:    "+1234567890",
			},
			wantError: false,
		},
		{
			name: "missing required fields",
			input: User{
				Age: 25,
			},
			wantError: true,
			errorMsg:  "Name",
		},
		{
			name: "invalid email",
			input: User{
				Name:     "John Doe",
				Email:    "invalid-email",
				Age:      25,
				Password: "password123",
			},
			wantError: true,
			errorMsg:  "email",
		},
		{
			name: "age below minimum",
			input: User{
				Name:     "John Doe",
				Email:    "john@example.com",
				Age:      17,
				Password: "password123",
			},
			wantError: true,
			errorMsg:  "Age",
		},
		{
			name: "password too short",
			input: User{
				Name:     "John Doe",
				Email:    "john@example.com",
				Age:      25,
				Password: "123",
			},
			wantError: true,
			errorMsg:  "Password",
		},
		{
			name: "invalid optional URL",
			input: User{
				Name:     "John Doe",
				Email:    "john@example.com",
				Age:      25,
				Password: "password123",
				Website:  "not-a-url",
				Phone:    "", // Make sure phone is empty to avoid additional errors
			},
			wantError: true,
			errorMsg:  "URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Struct(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}

				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidatorVar(t *testing.T) {
	validator := New()

	tests := []struct {
		name      string
		value     interface{}
		tag       string
		wantError bool
	}{
		{"valid email", "test@example.com", "email", false},
		{"invalid email", "invalid-email", "email", true},
		{"valid required string", "hello", "required", false},
		{"invalid required string", "", "required", true},
		{"valid min length", "hello", "min=3", false},
		{"invalid min length", "hi", "min=3", true},
		{"valid max length", "hello", "max=10", false},
		{"invalid max length", "hello world test", "max=10", true},
		{"valid exact length", "hello", "len=5", false},
		{"invalid exact length", "hello world", "len=5", true},
		{"valid oneof", "red", "oneof=red green blue", false},
		{"invalid oneof", "yellow", "oneof=red green blue", true},
		{"valid numeric", "12345", "numeric", false},
		{"invalid numeric", "123abc", "numeric", true},
		{"valid alpha", "hello", "alpha", false},
		{"invalid alpha", "hello123", "alpha", true},
		{"valid alphanum", "hello123", "alphanum", false},
		{"invalid alphanum", "hello@123", "alphanum", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Var(tt.value, tt.tag)

			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			} else if !tt.wantError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestValidatorNestedStructs(t *testing.T) {
	validator := New()

	validUser := UserWithAddress{
		User: User{
			Name:     "John Doe",
			Email:    "john@example.com",
			Age:      25,
			Password: "password123",
		},
		Address: Address{
			Street:   "123 Main Street",
			City:     "Anytown",
			PostCode: "12345",
			Country:  "US",
		},
		Tags: []string{"tag1", "tag2"},
		Metadata: map[string]string{
			"key": "value",
		},
	}

	err := validator.Struct(validUser)
	if err != nil {
		t.Errorf("expected valid nested struct to pass, got: %v", err)
	}

	// Test invalid nested struct
	invalidUser := validUser
	invalidUser.Address.Country = "INVALID"

	err = validator.Struct(invalidUser)
	if err == nil {
		t.Error("expected error for invalid nested struct")
	}

	// Test dive validation failure
	invalidUser2 := validUser
	invalidUser2.Tags = []string{"a", "valid_tag"} // First tag too short

	err = validator.Struct(invalidUser2)
	if err == nil {
		t.Error("expected error for dive validation failure")
	}
}

func TestValidatorCrossFieldValidation(t *testing.T) {
	validator := New()

	validCrossField := CrossFieldTest{
		Password:        "password123",
		ConfirmPassword: "password123",
		StartDate:       "2023-01-01",
		EndDate:         "2023-12-31",
		Age:             25,
		ParentEmail:     "", // Not required since age >= 18
	}

	err := validator.Struct(validCrossField)
	if err != nil {
		t.Errorf("expected valid cross-field validation to pass, got: %v", err)
	}

	// Test password mismatch
	invalidCrossField := validCrossField
	invalidCrossField.ConfirmPassword = "different"

	err = validator.Struct(invalidCrossField)
	if err == nil {
		t.Error("expected error for password mismatch")
	}

	// Test conditional required field
	invalidCrossField2 := validCrossField
	invalidCrossField2.Age = 17
	// ParentEmail should now be required but is empty

	err = validator.Struct(invalidCrossField2)
	if err == nil {
		t.Error("expected error for missing required conditional field")
	}
}

func TestValidatorCustomRules(t *testing.T) {
	validator := New()

	// Register custom validation rule
	err := validator.RegisterValidation("isawesome", func(fl FieldLevel) bool {
		return fl.Field().String() == "awesome"
	})
	if err != nil {
		t.Fatalf("failed to register custom validation: %v", err)
	}

	// Test custom rule
	err = validator.Var("awesome", "isawesome")
	if err != nil {
		t.Errorf("expected custom rule to pass, got: %v", err)
	}

	err = validator.Var("not awesome", "isawesome")
	if err == nil {
		t.Error("expected custom rule to fail")
	}
}

func TestValidatorStructLevelValidation(t *testing.T) {
	validator := New()

	// Register struct-level validation
	validator.RegisterStructValidation(func(sl StructLevel) {
		user := sl.Current().Interface().(User)

		// Custom business logic: if user is under 21, website should not be provided
		if user.Age < 21 && user.Website != "" {
			sl.ReportError("Website", "Website", "no_website_under_21",
				"users under 21 cannot have a website")
		}
	}, User{})

	// Test struct-level validation failure
	user := User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      20, // Under 21
		Password: "password123",
		Website:  "https://example.com", // Should not be allowed
	}

	err := validator.Struct(user)
	if err == nil {
		t.Error("expected struct-level validation to fail")
	}

	// Test struct-level validation pass
	user.Website = ""
	err = validator.Struct(user)
	if err != nil {
		t.Errorf("expected struct-level validation to pass, got: %v", err)
	}
}

func TestValidatorErrorCollection(t *testing.T) {
	validator := New()

	// Create user with multiple validation errors
	user := User{
		Name:     "J",             // Too short
		Email:    "invalid-email", // Invalid format
		Age:      15,              // Below minimum
		Password: "123",           // Too short
		Website:  "not-a-url",     // Invalid URL
	}

	err := validator.Struct(user)
	if err == nil {
		t.Fatal("expected multiple validation errors")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Fatal("expected ValidationErrors type")
	}

	// Should have errors for all invalid fields
	expectedFields := []string{"Name", "Email", "Age", "Password", "Website"}
	actualFields := validationErrors.Fields()

	if len(actualFields) < len(expectedFields) {
		t.Errorf("expected at least %d error fields, got %d: %v",
			len(expectedFields), len(actualFields), actualFields)
	}
}

func TestValidatorErrorMethods(t *testing.T) {
	validator := New()

	user := User{
		Email: "invalid-email",
	}

	err := validator.Struct(user)
	if err == nil {
		t.Fatal("expected validation errors")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Fatal("expected ValidationErrors type")
	}

	// Test error methods
	if !validationErrors.HasErrors() {
		t.Error("expected HasErrors() to return true")
	}

	emailErrors := validationErrors.FilterByField("Email")
	if len(emailErrors) == 0 {
		t.Error("expected email field errors")
	}

	requiredErrors := validationErrors.FilterByTag("required")
	if len(requiredErrors) == 0 {
		t.Error("expected required tag errors")
	}

	errorMap := validationErrors.AsMap()
	if len(errorMap) == 0 {
		t.Error("expected error map to have entries")
	}

	// Test JSON serialization
	jsonBytes, err := validationErrors.JSON()
	if err != nil {
		t.Errorf("failed to serialize errors to JSON: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("expected JSON output")
	}
}

func TestValidatorConfiguration(t *testing.T) {
	config := ValidatorConfig{
		TagName:  "validation",
		FailFast: true,
	}

	validator := NewWithConfig(config)

	type TestStruct struct {
		Field1 string `validation:"required"`
		Field2 string `validation:"required"`
	}

	test := TestStruct{} // Both fields empty

	err := validator.Struct(test)
	if err == nil {
		t.Fatal("expected validation error")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Fatal("expected ValidationErrors type")
	}

	// With FailFast=true, should only have one error
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 error with FailFast=true, got %d", len(validationErrors))
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Test package-level Struct function
	user := User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Password: "password123",
	}

	err := Struct(user)
	if err != nil {
		t.Errorf("expected package-level Struct to pass, got: %v", err)
	}

	// Test package-level Var function
	err = Var("test@example.com", "email")
	if err != nil {
		t.Errorf("expected package-level Var to pass, got: %v", err)
	}

	err = Var("invalid-email", "email")
	if err == nil {
		t.Error("expected package-level Var to fail")
	}

	// Test package-level RegisterValidation
	err = RegisterValidation("custom", func(fl FieldLevel) bool {
		return true
	})
	if err != nil {
		t.Errorf("failed to register validation: %v", err)
	}
}

// Benchmark tests
func BenchmarkValidatorStruct(b *testing.B) {
	validator := New()
	user := User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(user)
	}
}

func BenchmarkValidatorVar(b *testing.B) {
	validator := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Var("test@example.com", "email")
	}
}

func BenchmarkValidatorComplexStruct(b *testing.B) {
	validator := New()
	complex := UserWithAddress{
		User: User{
			Name:     "John Doe",
			Email:    "john@example.com",
			Age:      25,
			Password: "password123",
		},
		Address: Address{
			Street:   "123 Main Street",
			City:     "Anytown",
			PostCode: "12345",
			Country:  "US",
		},
		Tags: []string{"tag1", "tag2", "tag3"},
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(complex)
	}
}
