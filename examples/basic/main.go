package main

import (
	"fmt"

	"github.com/mateothegreat/go-validation"
)

// User represents a user with validation rules
type User struct {
	Name     string `validate:"required,min=2,max=50"`
	Email    string `validate:"required,email"`
	Age      int    `validate:"required,min=18,max=120"`
	Password string `validate:"required,min=8"`
	Website  string `validate:"omitempty,url"`
	Phone    string `validate:"omitempty,phone"`
}

func main() {
	fmt.Println("=== Basic Validation Examples ===\n")

	// Example 1: Valid user
	fmt.Println("1. Valid User:")
	validUser := User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      25,
		Password: "secretpassword",
		Website:  "https://johndoe.com",
		Phone:    "+1234567890",
	}

	err := validation.Struct(validUser)
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("✅ User is valid!")
	}

	// Example 2: Invalid user with multiple errors
	fmt.Println("\n2. Invalid User (Multiple Errors):")
	invalidUser := User{
		Name:     "J",                // Too short
		Email:    "invalid-email",    // Invalid format
		Age:      15,                 // Below minimum
		Password: "123",              // Too short
		Website:  "not-a-url",        // Invalid URL
		Phone:    "invalid-phone",    // Invalid phone
	}

	err = validation.Struct(invalidUser)
	if err != nil {
		fmt.Printf("❌ Validation failed:\n")
		
		// Cast to ValidationErrors for detailed information
		if validationErrors, ok := err.(validation.ValidationErrors); ok {
			for _, fieldErr := range validationErrors {
				fmt.Printf("  - %s: %s\n", fieldErr.Field, fieldErr.Message)
			}
			
			fmt.Printf("\n📊 Summary:\n")
			fmt.Printf("  - Total errors: %d\n", len(validationErrors))
			fmt.Printf("  - Fields with errors: %v\n", validationErrors.Fields())
		}
	}

	// Example 3: Individual variable validation
	fmt.Println("\n3. Individual Variable Validation:")
	
	// Valid email
	err = validation.Var("john@example.com", "email")
	if err == nil {
		fmt.Println("✅ Valid email")
	}
	
	// Invalid email
	err = validation.Var("invalid-email", "email")
	if err != nil {
		fmt.Printf("❌ Invalid email: %v\n", err)
	}
	
	// String length validation
	err = validation.Var("hello", "min=3,max=10")
	if err == nil {
		fmt.Println("✅ String length is valid")
	}
	
	// Numeric validation
	err = validation.Var(25, "min=18,max=65")
	if err == nil {
		fmt.Println("✅ Age is valid")
	}
	
	// One-of validation
	err = validation.Var("blue", "oneof=red green blue")
	if err == nil {
		fmt.Println("✅ Color is valid")
	}

	// Example 4: Optional fields
	fmt.Println("\n4. Optional Fields:")
	userWithOptionals := User{
		Name:     "Jane Doe",
		Email:    "jane@example.com",
		Age:      30,
		Password: "securepassword",
		// Website and Phone are omitted (optional)
	}

	err = validation.Struct(userWithOptionals)
	if err == nil {
		fmt.Println("✅ User with optional fields is valid!")
	}

	// Example 5: Error filtering and analysis
	fmt.Println("\n5. Error Analysis:")
	err = validation.Struct(invalidUser)
	if err != nil {
		if validationErrors, ok := err.(validation.ValidationErrors); ok {
			// Filter errors by field
			emailErrors := validationErrors.FilterByField("Email")
			if len(emailErrors) > 0 {
				fmt.Printf("📧 Email errors: %d\n", len(emailErrors))
			}
			
			// Filter errors by validation tag
			requiredErrors := validationErrors.FilterByTag("required")
			fmt.Printf("❗ Required field errors: %d\n", len(requiredErrors))
			
			minErrors := validationErrors.FilterByTag("min")
			fmt.Printf("📏 Minimum value/length errors: %d\n", len(minErrors))
			
			// Convert to map for easy access
			errorMap := validationErrors.AsMap()
			fmt.Printf("🗂️  Error map has %d fields\n", len(errorMap))
			
			// Serialize to JSON
			jsonBytes, _ := validationErrors.JSON()
			fmt.Printf("📄 JSON representation: %d bytes\n", len(jsonBytes))
		}
	}

	fmt.Println("\n=== Basic Examples Complete ===")
}