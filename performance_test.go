package validation

import (
	"testing"
)

// Simple performance benchmarks
func BenchmarkSimpleValidation(b *testing.B) {
	type SimpleUser struct {
		Name  string `validate:"required,min=2"`
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min=18"`
	}
	
	user := SimpleUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}
	
	validator := New()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Struct(user)
	}
}

func BenchmarkEmailValidation(b *testing.B) {
	validator := New()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Var("john@example.com", "email")
	}
}

func BenchmarkRequiredValidation(b *testing.B) {
	validator := New()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Var("hello", "required")
	}
}

func BenchmarkNumericRangeValidation(b *testing.B) {
	validator := New()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Var(25, "min=18,max=65")
	}
}