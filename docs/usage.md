# Usage Guide

This guide provides clear, concise examples of how to use the Go validation library in your own code, from simple validation to advanced patterns.

## Table of Contents

* [Usage Guide](#usage-guide)
  * [Table of Contents](#table-of-contents)
  * [Example 1: Super Simple Validation](#example-1-super-simple-validation)
  * [Example 2: Intermediate Validation](#example-2-intermediate-validation)
  * [Example 3: Advanced Validation](#example-3-advanced-validation)
  * [Zero Reflection Usage](#zero-reflection-usage)
  * [When to Use Reflection-Based Functionality](#when-to-use-reflection-based-functionality)
  * [Summary](#summary)

## Example 1: Super Simple Validation

```go
package main

import (
    "fmt"
    "github.com/mateothegreat/go-validation"
)

// Simple struct with validation tags
type User struct {
    Name  string `validate:"required,min=2,max=50"`
    Email string `validate:"required,email"`
    Age   int    `validate:"required,min=18"`
}

func main() {
    // Create a user
    user := User{
        Name:  "John Doe",
        Email: "john@example.com",
        Age:   25,
    }
    
    // Validate with one line
    err := validation.Struct(user)
    if err != nil {
        fmt.Printf("Validation failed: %v\n", err)
        return
    }
    
    fmt.Println("✅ User is valid!")
}
```

**Key Points:**

* ✅ **One line validation**: `validation.Struct(user)`
* ✅ **Standard validation tags**: `required`, `min`, `max`, `email`
* ✅ **Automatic error handling**: Detailed error messages included


## Example 2: Intermediate Validation

**Use Case**: E-commerce product validation with conditional rules and custom error handling.

```go
package main

import (
    "fmt"
    "github.com/mateothegreat/go-validation"
)

type Product struct {
    Name        string  `validate:"required,min=3,max=100"`
    Price       float64 `validate:"required,min=0.01"`
    Category    string  `validate:"required,oneof=electronics clothing books"`
    SKU         string  `validate:"required,len=8,alphanum"`
    Description string  `validate:"omitempty,min=10,max=500"`
    Weight      float64 `validate:"omitempty,min=0.1"`
    InStock     bool    `validate:"required"`
    // Conditional: Weight required for physical products
    IsDigital   bool    `validate:"required"`
}

type Store struct {
    Name     string    `validate:"required,min=2"`
    Products []Product `validate:"dive"`  // Validate each product in slice
}

func main() {
    store := Store{
        Name: "Tech Store",
        Products: []Product{
            {
                Name:        "Laptop",
                Price:       999.99,
                Category:    "electronics",
                SKU:         "TECH1234",
                Description: "High-performance laptop for professionals",
                Weight:      2.5,
                InStock:     true,
                IsDigital:   false,
            },
            {
                Name:      "E-book",
                Price:     9.99,
                Category:  "books",
                SKU:       "BOOK5678",
                InStock:   true,
                IsDigital: true,
                // Weight omitted for digital product
            },
        },
    }
    
    // Validate the entire store
    err := validation.Struct(store)
    if err != nil {
        // Advanced error handling
        if validationErrors, ok := err.(validation.ValidationErrors); ok {
            fmt.Println("❌ Validation Errors:")
            
            // Group errors by field
            errorMap := validationErrors.AsMap()
            for field, fieldErrors := range errorMap {
                fmt.Printf("  %s:\n", field)
                for _, fieldError := range fieldErrors {
                    fmt.Printf("    - %s\n", fieldError.Message)
                }
            }
            
            // Show summary
            fmt.Printf("\nSummary: %d errors across %d fields\n", 
                len(validationErrors), len(errorMap))
            return
        }
    }
    
    fmt.Printf("✅ Store validation passed! %d products validated.\n", len(store.Products))
}
```

**Key Points:**

* ✅ **Slice validation**: `dive` validates each element in Products slice
* ✅ **Optional fields**: `omitempty` allows empty values
* ✅ **Choice validation**: `oneof` restricts to specific values
* ✅ **Advanced error handling**: Group and analyze validation errors
* ✅ **Nested validation**: Store contains validated Products


## Example 3: Advanced Validation

**Use Case**: Complex user profile with cross-field validation, custom validators, and business logic.

```go
package main

import (
    "fmt"
    "strings"
    "time"
    "github.com/mateothegreat/go-validation"
)

type Address struct {
    Street   string `validate:"required,min=5"`
    City     string `validate:"required,min=2"`
    State    string `validate:"required,len=2,alpha"`
    ZipCode  string `validate:"required,len=5,numeric"`
    Country  string `validate:"required,iso3166_1_alpha2"`  // Custom validator
}

type UserProfile struct {
    // Basic info
    FirstName       string `validate:"required,min=2,max=50,alpha"`
    LastName        string `validate:"required,min=2,max=50,alpha"`
    Email           string `validate:"required,email"`
    Phone           string `validate:"omitempty,phone"`
    
    // Security
    Password        string `validate:"required,min=8,secure_password"`  // Custom validator
    ConfirmPassword string `validate:"required,eqfield=Password"`
    
    // Profile data
    DateOfBirth     string `validate:"required,date"`
    Age             int    `validate:"required,min=13,max=120"`
    
    // Address info
    PrimaryAddress  Address   `validate:"required"`
    BillingAddress  *Address  `validate:"omitempty"`  // Optional pointer
    
    // Preferences
    Newsletter      bool     `validate:"required"`
    MarketingEmails bool     `validate:"required_if=Newsletter true"`
    
    // Social
    Website         string   `validate:"omitempty,url"`
    SocialMedia     map[string]string `validate:"dive,keys,oneof=twitter facebook linkedin,endkeys,url"`
    
    // Metadata
    Tags            []string `validate:"dive,min=2,max=20"`
    Preferences     map[string]interface{} `validate:"dive,keys,alpha,endkeys,required"`
}

func main() {
    // Create validator with custom configuration
    validator := validation.NewWithConfig(validation.ValidatorConfig{
        TagName:  "validate",
        FailFast: false,  // Collect all errors
    })
    
    // Register custom validators
    registerCustomValidators(validator)
    
    // Register struct-level validation for business logic
    validator.RegisterStructValidation(validateUserProfileBusinessLogic, UserProfile{})
    
    // Test user profile
    profile := UserProfile{
        FirstName:       "John",
        LastName:        "Doe",
        Email:           "john.doe@example.com",
        Phone:           "+1234567890",
        Password:        "SecurePass123!",
        ConfirmPassword: "SecurePass123!",
        DateOfBirth:     "1990-05-15",
        Age:             33,
        PrimaryAddress: Address{
            Street:  "123 Main Street",
            City:    "San Francisco",
            State:   "CA",
            ZipCode: "94105",
            Country: "US",
        },
        Newsletter:      true,
        MarketingEmails: true,
        Website:         "https://johndoe.com",
        SocialMedia: map[string]string{
            "twitter":  "https://twitter.com/johndoe",
            "linkedin": "https://linkedin.com/in/johndoe",
        },
        Tags: []string{"developer", "golang", "tech"},
        Preferences: map[string]interface{}{
            "theme":       "dark",
            "language":    "en",
            "timezone":    "PST",
        },
    }
    
    // Validate the complex profile
    err := validator.Struct(profile)
    if err != nil {
        handleValidationErrors(err)
        return
    }
    
    fmt.Println("✅ Advanced user profile validation passed!")
    fmt.Printf("   User: %s %s <%s>\n", profile.FirstName, profile.LastName, profile.Email)
    fmt.Printf("   Address: %s, %s %s\n", profile.PrimaryAddress.City, 
        profile.PrimaryAddress.State, profile.PrimaryAddress.ZipCode)
    fmt.Printf("   Social: %d platforms, %d tags\n", 
        len(profile.SocialMedia), len(profile.Tags))
}

// Custom validator registration
func registerCustomValidators(validator *validation.Validator) {
    // ISO 3166-1 alpha-2 country code validator
    validator.RegisterValidation("iso3166_1_alpha2", func(fl validation.FieldLevel) bool {
        validCountries := map[string]bool{
            "US": true, "CA": true, "UK": true, "DE": true, "FR": true,
            "JP": true, "AU": true, "BR": true, "IN": true, "CN": true,
        }
        return validCountries[fl.Field().String()]
    })
    
    // Secure password validator
    validator.RegisterValidation("secure_password", func(fl validation.FieldLevel) bool {
        password := fl.Field().String()
        
        // Must have at least 8 characters
        if len(password) < 8 {
            return false
        }
        
        var hasUpper, hasLower, hasDigit, hasSpecial bool
        for _, char := range password {
            switch {
            case char >= 'A' && char <= 'Z':
                hasUpper = true
            case char >= 'a' && char <= 'z':
                hasLower = true
            case char >= '0' && char <= '9':
                hasDigit = true
            case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
                hasSpecial = true
            }
        }
        
        return hasUpper && hasLower && hasDigit && hasSpecial
    })
}

// Struct-level validation for complex business logic
func validateUserProfileBusinessLogic(sl validation.StructLevel) {
    profile := sl.Current().Interface().(UserProfile)
    
    // Business rule: Age must match date of birth
    if profile.DateOfBirth != "" {
        if birthDate, err := time.Parse("2006-01-02", profile.DateOfBirth); err == nil {
            expectedAge := time.Now().Year() - birthDate.Year()
            if time.Now().YearDay() < birthDate.YearDay() {
                expectedAge--
            }
            
            if profile.Age != expectedAge {
                sl.ReportError("Age", "Age", "age_mismatch", 
                    fmt.Sprintf("age %d does not match date of birth %s", 
                        profile.Age, profile.DateOfBirth))
            }
        }
    }
    
    // Business rule: Marketing emails require newsletter subscription
    if profile.MarketingEmails && !profile.Newsletter {
        sl.ReportError("MarketingEmails", "MarketingEmails", "requires_newsletter",
            "marketing emails require newsletter subscription")
    }
    
    // Business rule: Social media URLs must match platform
    for platform, url := range profile.SocialMedia {
        if !strings.Contains(url, platform) {
            sl.ReportError("SocialMedia", "SocialMedia", "platform_mismatch",
                fmt.Sprintf("URL for %s must contain platform name", platform))
        }
    }
}

// Advanced error handling
func handleValidationErrors(err error) {
    if validationErrors, ok := err.(validation.ValidationErrors); ok {
        fmt.Println("❌ Advanced Validation Failed:")
        
        // Categorize errors
        fieldErrors := make(map[string][]validation.ValidationError)
        businessLogicErrors := make([]validation.ValidationError, 0)
        
        for _, validationError := range validationErrors {
            if validationError.Tag == "age_mismatch" || 
               validationError.Tag == "requires_newsletter" ||
               validationError.Tag == "platform_mismatch" {
                businessLogicErrors = append(businessLogicErrors, validationError)
            } else {
                fieldErrors[validationError.Field] = append(
                    fieldErrors[validationError.Field], validationError)
            }
        }
        
        // Show field validation errors
        if len(fieldErrors) > 0 {
            fmt.Println("\n📝 Field Validation Errors:")
            for field, errors := range fieldErrors {
                fmt.Printf("  %s:\n", field)
                for _, err := range errors {
                    fmt.Printf("    - %s\n", err.Message)
                }
            }
        }
        
        // Show business logic errors
        if len(businessLogicErrors) > 0 {
            fmt.Println("\n🏢 Business Logic Errors:")
            for _, err := range businessLogicErrors {
                fmt.Printf("  - %s: %s\n", err.Field, err.Message)
            }
        }
        
        // Show summary
        fmt.Printf("\n📊 Summary: %d total errors (%d field, %d business logic)\n",
            len(validationErrors), len(fieldErrors), len(businessLogicErrors))
        
        // Export to JSON for logging
        if jsonBytes, err := validationErrors.JSON(); err == nil {
            fmt.Printf("📄 JSON export: %d bytes ready for logging\n", len(jsonBytes))
        }
    }
}
```

**Key Points:**

* ✅ **Custom validators**: `secure_password`, `iso3166_1_alpha2`
* ✅ **Struct-level validation**: Complex business logic validation
* ✅ **Cross-field validation**: `eqfield=Password`
* ✅ **Conditional validation**: `required_if=Newsletter true`
* ✅ **Deep validation**: Maps and slices with `dive`
* ✅ **Pointer handling**: Optional `*Address` fields
* ✅ **Advanced error categorization**: Field vs business logic errors


## Zero Reflection Usage

**Use Case**: Maximum performance validation with compile-time type safety and zero reflection overhead.

```go
package main

import (
    "fmt"
    "github.com/mateothegreat/go-validation/rules"
)

func main() {
    fmt.Println("=== Zero Reflection Validation ===")
    
    // 1. Direct validator usage (fastest possible)
    emailValidator := rules.NewCharacteristic(rules.RuleStaticEmail)
    if err := emailValidator.Validate("email", "john@example.com"); err != nil {
        fmt.Printf("❌ Email validation failed: %v\n", err)
    } else {
        fmt.Println("✅ Email is valid (zero reflection)")
    }
    
    // 2. Generic range validator (type-safe, zero reflection)
    ageValidator := rules.NewNumericRange[int](18, 120)
    if err := ageValidator.Validate("age", 25); err != nil {
        fmt.Printf("❌ Age validation failed: %v\n", err)
    } else {
        fmt.Println("✅ Age is valid (zero reflection)")
    }
    
    // 3. String length validator (zero reflection)
    nameValidator := rules.NewStringLength("min", 2, 0)
    if err := nameValidator.Validate("name", "John"); err != nil {
        fmt.Printf("❌ Name validation failed: %v\n", err)
    } else {
        fmt.Println("✅ Name is valid (zero reflection)")
    }
    
    // 4. Factory pattern with caching (minimal reflection)
    rangeValidator, err := rules.GetRule[int64]("range_int64", "range=1000:99999")
    if err != nil {
        fmt.Printf("❌ Failed to get validator: %v\n", err)
        return
    }
    
    if err := rangeValidator.Validate("salary", int64(50000)); err != nil {
        fmt.Printf("❌ Salary validation failed: %v\n", err)
    } else {
        fmt.Println("✅ Salary is valid (cached factory)")
    }
    
    // 5. Manual field validation (you control reflection)
    validateUserManually()
    
    fmt.Println("\n🚀 All zero-reflection validations completed!")
}

// Manual validation - you control when reflection is used
func validateUserManually() {
    type User struct {
        Name  string
        Email string
        Age   int
    }
    
    user := User{
        Name:  "John Doe",
        Email: "john@example.com", 
        Age:   25,
    }
    
    // Manual validation with direct field access (no reflection)
    var errors []string
    
    // Validate name
    if len(user.Name) < 2 {
        errors = append(errors, "Name must be at least 2 characters")
    }
    
    // Validate email using zero-reflection validator
    emailValidator := rules.NewCharacteristic(rules.RuleStaticEmail)
    if err := emailValidator.Validate("Email", user.Email); err != nil {
        errors = append(errors, err.Error())
    }
    
    // Validate age using zero-reflection validator
    ageValidator := rules.NewNumericRange[int](18, 120)
    if err := ageValidator.Validate("Age", user.Age); err != nil {
        errors = append(errors, err.Error())
    }
    
    if len(errors) > 0 {
        fmt.Printf("❌ Manual validation failed: %v\n", errors)
    } else {
        fmt.Println("✅ Manual validation passed (zero reflection)")
    }
}
```

**Performance Benefits:**

* ✅ **2-3ns/op**: Direct validator calls with zero reflection
* ✅ **0 allocations**: No boxing/unboxing of interface{} values
* ✅ **Compile-time safety**: Type errors caught at compile time
* ✅ **Predictable performance**: No reflection overhead variations

**When to Use:**

* 🎯 **High-frequency validation**: APIs processing thousands of requests/second
* 🎯 **Performance-critical paths**: Real-time systems or hot loops
* 🎯 **Simple data structures**: When struct tags aren't needed
* 🎯 **Known field types**: When you can validate fields individually


## When to Use Reflection-Based Functionality

**Use Case**: When you need the convenience of automatic struct validation and can accept the performance trade-off.

```go
package main

import (
    "fmt"
    "reflect"
    "github.com/mateothegreat/go-validation"
)

// Complex nested structure requiring reflection
type Organization struct {
    Name        string                 `validate:"required,min=2"`
    Departments []Department           `validate:"dive"`
    Metadata    map[string]interface{} `validate:"dive,keys,alpha,endkeys,required"`
    Settings    interface{}            `validate:"required"`  // Dynamic type
}

type Department struct {
    Name      string     `validate:"required"`
    Manager   *Employee  `validate:"omitempty"`  // Optional pointer
    Employees []Employee `validate:"dive"`
}

type Employee struct {
    FirstName string `validate:"required,min=2"`
    LastName  string `validate:"required,min=2"`
    Email     string `validate:"required,email"`
    Position  string `validate:"required"`
}

func main() {
    fmt.Println("=== Reflection-Based Validation ===")
    
    // 1. Complex nested structure validation (requires reflection)
    org := Organization{
        Name: "Tech Corp",
        Departments: []Department{
            {
                Name: "Engineering",
                Manager: &Employee{
                    FirstName: "Alice",
                    LastName:  "Johnson",
                    Email:     "alice@techcorp.com",
                    Position:  "Engineering Manager",
                },
                Employees: []Employee{
                    {
                        FirstName: "Bob",
                        LastName:  "Smith",
                        Email:     "bob@techcorp.com",
                        Position:  "Software Engineer",
                    },
                    {
                        FirstName: "Carol",
                        LastName:  "Williams",
                        Email:     "carol@techcorp.com",
                        Position:  "Senior Engineer",
                    },
                },
            },
        },
        Metadata: map[string]interface{}{
            "industry":    "technology",
            "founded":     2010,
            "location":    "San Francisco",
        },
        Settings: map[string]string{
            "theme": "dark",
            "lang":  "en",
        },
    }
    
    // Validate complex structure (reflection required for traversal)
    err := validation.Struct(org)
    if err != nil {
        fmt.Printf("❌ Organization validation failed: %v\n", err)
    } else {
        fmt.Println("✅ Organization validation passed!")
        fmt.Printf("   Validated %d departments with %d total employees\n", 
            len(org.Departments), countEmployees(org))
    }
    
    // 2. Dynamic validation based on runtime type (requires reflection)
    validateDynamicData()
    
    // 3. Generic validation function (reflection-based)
    validateAnyStruct()
    
    // 4. Custom struct-level validation (requires reflection)
    demonstrateStructLevelValidation()
    
    fmt.Println("\n🔍 Reflection-based validations completed!")
}

// Dynamic validation when you don't know the type at compile time
func validateDynamicData() {
    // Simulate receiving data from JSON/database with unknown structure
    var data interface{} = struct {
        Name  string `validate:"required,min=3"`
        Value int    `validate:"required,min=1"`
    }{
        Name:  "Dynamic Item",
        Value: 42,
    }
    
    // Validate unknown type (requires reflection)
    err := validation.Struct(data)
    if err != nil {
        fmt.Printf("❌ Dynamic validation failed: %v\n", err)
    } else {
        fmt.Println("✅ Dynamic data validation passed!")
    }
}

// Generic validation function that works with any struct
func validateAnyStruct() {
    // Function that can validate any struct type
    validateGeneric := func(v interface{}) error {
        // Check if it's a struct using reflection
        val := reflect.ValueOf(v)
        if val.Kind() == reflect.Ptr {
            val = val.Elem()
        }
        
        if val.Kind() != reflect.Struct {
            return fmt.Errorf("not a struct")
        }
        
        // Use reflection-based validation
        return validation.Struct(v)
    }
    
    // Test with different struct types
    testStructs := []interface{}{
        struct {
            Email string `validate:"required,email"`
        }{Email: "test@example.com"},
        
        struct {
            Age int `validate:"required,min=18"`
        }{Age: 25},
    }
    
    for i, testStruct := range testStructs {
        if err := validateGeneric(testStruct); err != nil {
            fmt.Printf("❌ Generic validation %d failed: %v\n", i+1, err)
        } else {
            fmt.Printf("✅ Generic validation %d passed!\n", i+1)
        }
    }
}

// Struct-level validation requires reflection to access fields
func demonstrateStructLevelValidation() {
    type BankAccount struct {
        AccountNumber string  `validate:"required,len=10,numeric"`
        Balance       float64 `validate:"required,min=0"`
        IsActive      bool    `validate:"required"`
        OverdraftLimit float64 `validate:"omitempty,min=0"`
    }
    
    // Register struct-level validation (requires reflection)
    validation.RegisterStructValidation(func(sl validation.StructLevel) {
        account := sl.Current().Interface().(BankAccount)
        
        // Business rule: Inactive accounts cannot have positive balance
        if !account.IsActive && account.Balance > 0 {
            sl.ReportError("Balance", "Balance", "inactive_positive_balance",
                "inactive accounts cannot have positive balance")
        }
        
        // Business rule: Overdraft limit cannot exceed balance by more than 1000
        if account.OverdraftLimit > account.Balance + 1000 {
            sl.ReportError("OverdraftLimit", "OverdraftLimit", "excessive_overdraft",
                "overdraft limit cannot exceed balance + $1000")
        }
    }, BankAccount{})
    
    account := BankAccount{
        AccountNumber:  "1234567890",
        Balance:        1500.00,
        IsActive:       true,
        OverdraftLimit: 500.00,
    }
    
    err := validation.Struct(account)
    if err != nil {
        fmt.Printf("❌ Bank account validation failed: %v\n", err)
    } else {
        fmt.Println("✅ Bank account validation passed!")
    }
}

// Helper function
func countEmployees(org Organization) int {
    total := 0
    for _, dept := range org.Departments {
        total += len(dept.Employees)
        if dept.Manager != nil {
            total++
        }
    }
    return total
}
```

**When Reflection Is Necessary:**

* ✅ **Struct traversal**: Automatic field enumeration and tag reading
* ✅ **Nested validation**: Deep validation of complex structures
* ✅ **Dynamic types**: Validating `interface{}` or unknown types at runtime
* ✅ **Generic functions**: Functions that work with any struct type
* ✅ **Slice/map validation**: Automatic iteration over collections
* ✅ **Pointer dereferencing**: Handling `*struct` and `**struct` automatically

**Performance Trade-offs:**

* ❌ **\~1.3μs/op**: vs 2-3ns/op for direct validation
* ❌ **17 allocations**: vs 0 allocations for direct validation
* ✅ **Developer productivity**: Automatic validation vs manual field handling
* ✅ **Maintainability**: Declarative tags vs imperative validation code

**Best Practices:**



1. **Use reflection-based validation** for complex, nested structures
2. **Use zero-reflection validation** for high-frequency, simple validation
3. **Profile your application** to determine if the performance difference matters
4. **Consider hybrid approach**: Manual validation for hot paths, struct validation for complex data


## Summary

| Approach             | Performance         | Convenience | Use Case                              |
| -------------------- | ------------------- | ----------- | ------------------------------------- |
| **Zero Reflection**  | 2-3ns/op, 0 allocs  | Manual      | Hot paths, simple validation          |
| **Reflection-Based** | 1.3μs/op, 17 allocs | Automatic   | Complex structures, rapid development |

Choose based on your specific needs:

* 🚀 **Performance-critical**: Use zero-reflection approach
* 🛠️ **Development speed**: Use reflection-based struct validation
* ⚖️ **Balanced**: Use reflection for complex data, direct validation for hot paths


