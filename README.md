# Go Validation Library

A comprehensive, high-performance validation library for Go that provides struct tag validation, custom validators, and production-ready error handling.

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.20-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## Features

- 🚀 **High Performance**: Zero-allocation validation path (2-3ns/op)
- 📝 **Struct Tag Validation**: Industry-standard validation tags
- 🎯 **Comprehensive Validators**: 30+ built-in validation rules
- 🔧 **Custom Validators**: Easy registration of custom validation functions
- 🌐 **Cross-Field Validation**: Compare fields within the same struct
- 📊 **Conditional Validation**: Required-if, required-unless, required-with patterns
- 🏗️ **Nested Validation**: Deep validation of nested structs, slices, and maps
- 📋 **Structured Errors**: Detailed error information with field paths
- 🔄 **Framework Integration**: Ready for HTTP middleware integration
- 📚 **Production Ready**: Thread-safe, extensive testing, benchmarking

## Quick Start

### Installation

```bash
go get github.com/mateothegreat/go-validation
```

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/mateothegreat/go-validation"
)

type User struct {
    Name     string `validate:"required,min=2,max=50"`
    Email    string `validate:"required,email"`
    Age      int    `validate:"required,min=18,max=120"`
    Password string `validate:"required,min=8"`
    Website  string `validate:"omitempty,url"`
}

func main() {
    user := User{
        Name:     "John Doe",
        Email:    "john@example.com",
        Age:      25,
        Password: "secretpassword",
        Website:  "https://johndoe.com",
    }

    // Validate struct
    err := validation.Struct(user)
    if err != nil {
        fmt.Printf("Validation failed: %v\n", err)
        return
    }

    fmt.Println("User is valid!")
}
```

### Individual Variable Validation

```go
// Validate single variables
err := validation.Var("john@example.com", "email")
err = validation.Var("password123", "min=8")
err = validation.Var(25, "min=18,max=65")
```

## Validation Rules

### String Validation

| Rule | Description | Example |
|------|-------------|---------|
| `required` | Field must not be empty | `validate:"required"` |
| `min=n` | Minimum length | `validate:"min=3"` |
| `max=n` | Maximum length | `validate:"max=50"` |
| `len=n` | Exact length | `validate:"len=10"` |
| `alpha` | Alphabetic characters only | `validate:"alpha"` |
| `alphanum` | Alphanumeric characters only | `validate:"alphanum"` |
| `numeric` | Numeric characters only | `validate:"numeric"` |
| `email` | Valid email format | `validate:"email"` |
| `url` | Valid URL format | `validate:"url"` |
| `oneof` | One of specified values | `validate:"oneof=red green blue"` |

### Numeric Validation

| Rule | Description | Example |
|------|-------------|---------|
| `min=n` | Minimum value | `validate:"min=18"` |
| `max=n` | Maximum value | `validate:"max=65"` |
| `eq=n` | Equal to value | `validate:"eq=42"` |
| `ne=n` | Not equal to value | `validate:"ne=0"` |

### Network Validation

| Rule | Description | Example |
|------|-------------|---------|
| `ip` | Valid IP address (v4 or v6) | `validate:"ip"` |
| `ipv4` | Valid IPv4 address | `validate:"ipv4"` |
| `ipv6` | Valid IPv6 address | `validate:"ipv6"` |
| `cidr` | Valid CIDR notation | `validate:"cidr"` |
| `mac` | Valid MAC address | `validate:"mac"` |
| `hostname` | Valid hostname | `validate:"hostname"` |

### Format Validation

| Rule | Description | Example |
|------|-------------|---------|
| `uuid` | Valid UUID | `validate:"uuid"` |
| `uuid4` | Valid UUID v4 | `validate:"uuid4"` |
| `datetime` | Valid datetime (ISO 8601) | `validate:"datetime"` |
| `date` | Valid date (YYYY-MM-DD) | `validate:"date"` |
| `time` | Valid time (HH:MM:SS) | `validate:"time"` |
| `json` | Valid JSON string | `validate:"json"` |
| `base64` | Valid base64 string | `validate:"base64"` |
| `creditcard` | Valid credit card (Luhn) | `validate:"creditcard"` |
| `phone` | Valid phone (E.164 format) | `validate:"phone"` |

### Cross-Field Validation

| Rule | Description | Example |
|------|-------------|---------|
| `eqfield=Field` | Equal to another field | `validate:"eqfield=Password"` |
| `nefield=Field` | Not equal to another field | `validate:"nefield=OldPassword"` |
| `gtfield=Field` | Greater than another field | `validate:"gtfield=StartDate"` |
| `ltfield=Field` | Less than another field | `validate:"ltfield=EndDate"` |

### Conditional Validation

| Rule | Description | Example |
|------|-------------|---------|
| `required_if=Field Value` | Required if field equals value | `validate:"required_if=Type premium"` |
| `required_unless=Field Value` | Required unless field equals value | `validate:"required_unless=Status active"` |
| `required_with=Field` | Required if field has any value | `validate:"required_with=Address"` |
| `required_without=Field` | Required if field is empty | `validate:"required_without=Phone"` |

## Advanced Features

### Nested Struct Validation

```go
type Address struct {
    Street   string `validate:"required,min=5"`
    City     string `validate:"required"`
    PostCode string `validate:"required,len=5"`
}

type User struct {
    Name    string  `validate:"required"`
    Address Address `validate:"required"`
}
```

### Slice and Map Validation

```go
type User struct {
    Tags     []string          `validate:"dive,required,min=2"`
    Metadata map[string]string `validate:"dive,keys,alpha,endkeys,required"`
}
```

### Custom Validators

```go
// Register custom validation function
err := validation.RegisterValidation("isawesome", func(fl validation.FieldLevel) bool {
    return fl.Field().String() == "awesome"
})

// Use in struct tags
type Product struct {
    Quality string `validate:"isawesome"`
}
```

### Struct-Level Validation

```go
// Register struct-level validation for complex business logic
validation.RegisterStructValidation(func(sl validation.StructLevel) {
    user := sl.Current().Interface().(User)
    
    if user.Age < 21 && user.CanPurchaseAlcohol {
        sl.ReportError("CanPurchaseAlcohol", "CanPurchaseAlcohol", 
            "age_restriction", "users under 21 cannot purchase alcohol")
    }
}, User{})
```

### Error Handling

```go
err := validation.Struct(user)
if err != nil {
    // Cast to ValidationErrors for detailed error information
    if validationErrors, ok := err.(validation.ValidationErrors); ok {
        
        // Get all field names with errors
        fields := validationErrors.Fields()
        
        // Filter errors by field
        emailErrors := validationErrors.FilterByField("Email")
        
        // Filter errors by validation tag
        requiredErrors := validationErrors.FilterByTag("required")
        
        // Convert to map for easy lookup
        errorMap := validationErrors.AsMap()
        
        // Serialize to JSON
        jsonBytes, _ := validationErrors.JSON()
    }
}
```

## Performance

The library is optimized for high-performance scenarios:

### Benchmarks

```
BenchmarkValidatorStruct-10                300000    4500 ns/op    120 B/op    3 allocs/op
BenchmarkValidatorVar_Email-10            2000000     850 ns/op     32 B/op    1 allocs/op
BenchmarkValidatorVar_Required-10        50000000       3 ns/op      0 B/op    0 allocs/op
```

### Performance Tips

1. **Reuse Validator Instance**: Create once, use many times
2. **Use Package Functions**: For simple validation, use `validation.Struct()` and `validation.Var()`
3. **Avoid Reflection**: Built-in validators are optimized to minimize reflection
4. **Enable Fail Fast**: Set `FailFast: true` for early termination on first error

## Configuration

### Custom Validator Configuration

```go
config := validation.ValidatorConfig{
    TagName:      "validation",  // Use custom tag name
    FailFast:     true,          // Stop on first error
    IgnoreFields: []string{"InternalField"}, // Skip certain fields
}

validator := validation.NewWithConfig(config)
```

### Field Name Functions

```go
// Custom field name extraction (e.g., use JSON tags)
validator.SetFieldNameFunc(func(fld reflect.StructField) string {
    if jsonTag := fld.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
        if name := strings.Split(jsonTag, ",")[0]; name != "" {
            return name
        }
    }
    return fld.Name
})
```

## HTTP Middleware Integration

### Gin Framework

```go
func ValidationMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Bind and validate request
        var req UserRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": "invalid JSON"})
            return
        }
        
        if err := validation.Struct(req); err != nil {
            if validationErrors, ok := err.(validation.ValidationErrors); ok {
                c.JSON(400, gin.H{
                    "error": "validation failed",
                    "details": validationErrors.AsMap(),
                })
                return
            }
        }
        
        c.Set("validated_request", req)
        c.Next()
    }
}
```

### Echo Framework

```go
func ValidationMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Similar implementation for Echo
            return next(c)
        }
    }
}
```

## Testing

The library includes comprehensive tests covering:

- All built-in validation rules
- Cross-field validation scenarios
- Nested struct validation
- Error handling and collection
- Performance benchmarks
- Edge cases and error conditions

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

## Examples

See the [examples](examples/) directory for complete working examples:

- [Basic Validation](examples/basic/main.go)
- [Advanced Features](examples/advanced/main.go)
- [HTTP API Integration](examples/http/main.go)
- [Custom Validators](examples/custom/main.go)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Run benchmarks to ensure no performance regression
6. Commit your changes (`git commit -am 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Create a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by popular validation libraries in other languages
- Built with performance and usability in mind
- Designed for production use in high-scale applications

## Support

- 📖 [Documentation](https://pkg.go.dev/github.com/mateothegreat/go-validation)
- 🐛 [Issue Tracker](https://github.com/mateothegreat/go-validation/issues)
- 💬 [Discussions](https://github.com/mateothegreat/go-validation/discussions)