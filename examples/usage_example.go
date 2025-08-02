package main

import (
	"fmt"
	"log"

	"github.com/mateothegreat/go-validation/rules"
)

// User represents a user model with validation rules
type User struct {
	ID       int64   `validate:"range=1:999999"`
	Name     string  `validate:"minlen=2"`
	Email    string  `validate:"minlen=5"`
	Age      int     `validate:"range=18:120"`
	Score    float64 `validate:"range=0:100"`
	Role     string  `validate:"oneof=admin,user,guest"`
	Username string  `validate:"alphanumeric"`
}

func main() {
	// Example 1: Direct validator usage (most efficient)
	fmt.Println("=== Direct Validator Usage ===")
	ageValidator := rules.NewNumericRange[int](18, 65)
	if err := ageValidator.Validate("age", 25); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("Age validation passed")
	}

	// Example 2: Factory pattern with caching (recommended for dynamic rules)
	fmt.Println("\n=== Factory Pattern with Caching ===")
	rangeValidator, err := rules.GetRule[int64]("range_int64", "range=1000:99999")
	if err != nil {
		log.Fatal(err)
	}
	
	if err := rangeValidator.Validate("salary", int64(50000)); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("Salary validation passed")
	}

	// Example 3: Multiple field validation
	fmt.Println("\n=== Multiple Field Validation ===")
	user := User{
		ID:       12345,
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      30,
		Score:    85.5,
		Role:     "admin",
		Username: "johndoe123",
	}

	if errs := validateUser(user); len(errs) > 0 {
		fmt.Println("User validation errors:")
		for _, err := range errs {
			fmt.Printf("  - %v\n", err)
		}
	} else {
		fmt.Println("User validation passed")
	}

	// Example 4: Rule groups usage
	fmt.Println("\n=== Rule Groups ===")
	stringRules := rules.GetRuleGroup(rules.GroupString)
	fmt.Printf("String rules available: %v\n", stringRules)

	numericRules := rules.GetRuleGroup(rules.GroupNumeric)
	fmt.Printf("Numeric rules available: %v\n", numericRules)

	// Example 5: Performance demonstration
	fmt.Println("\n=== Performance Demonstration ===")
	demonstratePerformance()

	// Example 6: Extensibility - adding custom rules
	fmt.Println("\n=== Custom Rule Extension ===")
	demonstrateCustomRule()
}

func validateUser(user User) []error {
	fieldRules := map[string]string{
		"ID":       "range=1:999999",
		"Name":     "minlen=2",
		"Email":    "minlen=5", 
		"Age":      "range=18:120",
		"Score":    "range=0:100",
		"Role":     "oneof=admin,user,guest",
		"Username": "alphanumeric",
	}

	values := map[string]any{
		"ID":       int64(user.ID),
		"Name":     user.Name,
		"Email":    user.Email,
		"Age":      user.Age,
		"Score":    user.Score,
		"Role":     user.Role,
		"Username": user.Username,
	}

	return rules.ValidateFields(fieldRules, values)
}

func demonstratePerformance() {
	// Create validator once, reuse many times (most efficient)
	validator := rules.NewNumericRange[int](1, 100)
	
	fmt.Println("Validating 1000 values with single validator instance:")
	for i := 1; i <= 1000; i++ {
		if err := validator.Validate("test", i); err != nil && i <= 100 {
			fmt.Printf("Unexpected error for value %d: %v\n", i, err)
		}
	}
	fmt.Println("Performance test completed")
}

// EmailValidator is a custom email validator
type EmailValidator struct{}

func (e EmailValidator) Validate(field string, value string) error {
	if !containsAt(value) || !containsDot(value) {
		return fmt.Errorf("field '%s' is not a valid email format", field)
	}
	return nil
}

func (e EmailValidator) String() string {
	return "email"
}

func demonstrateCustomRule() {
	// Register the custom validator
	rules.RegisterRule("email", func(ruleString string) (rules.Validator[string], error) {
		return EmailValidator{}, nil
	})
	
	// Register with group
	rules.RegisterRuleGroup(rules.GroupFormat, "email")
	
	// Use the custom validator
	emailValidator, err := rules.GetRule[string]("email", "email")
	if err != nil {
		fmt.Printf("Failed to get email validator: %v\n", err)
		return
	}
	
	testEmails := []string{
		"valid@example.com",
		"invalid-email",
		"another@valid.org",
	}
	
	for _, email := range testEmails {
		if err := emailValidator.Validate("email", email); err != nil {
			fmt.Printf("Email validation failed for '%s': %v\n", email, err)
		} else {
			fmt.Printf("Email validation passed for '%s'\n", email)
		}
	}
}

// Simple helper functions for email validation
func containsAt(s string) bool {
	for _, r := range s {
		if r == '@' {
			return true
		}
	}
	return false
}

func containsDot(s string) bool {
	for _, r := range s {
		if r == '.' {
			return true
		}
	}
	return false
}