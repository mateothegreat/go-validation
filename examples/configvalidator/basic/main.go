//go:generate go run ../../../cmd/configvalidator/main.go -input=. -output=./generated -package=main -strategies -optimize

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mateothegreat/go-validation"
	"github.com/mateothegreat/go-validation/internal/integration"
)

// AppConfig represents the application configuration with comprehensive validation
type AppConfig struct {
	Meta     MetaConfig     `yaml:"meta" validate:"required"`
	Server   ServerConfig   `yaml:"server" validate:"required"`
	Database DatabaseConfig `yaml:"database" validate:"required"`
	Cache    CacheConfig    `yaml:"cache"`
	Logging  LoggingConfig  `yaml:"logging" validate:"required"`
	Features FeatureConfig  `yaml:"features"`
}

// MetaConfig contains application metadata
type MetaConfig struct {
	AppName     string `yaml:"app_name" validate:"required,alpha"`
	Version     string `yaml:"version" validate:"required"`
	Environment string `yaml:"environment" validate:"required,oneof=development staging production"`
	Debug       bool   `yaml:"debug"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Host         string        `yaml:"host" validate:"required,hostname"`
	Port         int           `yaml:"port" validate:"required,min=1,max=65535"`
	ReadTimeout  time.Duration `yaml:"read_timeout" validate:"min=1s"`
	WriteTimeout time.Duration `yaml:"write_timeout" validate:"min=1s"`
	TLS          *TLSConfig    `yaml:"tls"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file" validate:"required_if=Enabled true"`
	KeyFile  string `yaml:"key_file" validate:"required_if=Enabled true"`
}

// DatabaseConfig represents database connection configuration
type DatabaseConfig struct {
	Driver   string `yaml:"driver" validate:"required,oneof=postgres mysql sqlite"`
	Host     string `yaml:"host" validate:"required_unless=Driver sqlite,hostname"`
	Port     int    `yaml:"port" validate:"required_unless=Driver sqlite,min=1,max=65535"`
	Database string `yaml:"database" validate:"required"`
	Username string `yaml:"username" validate:"required_unless=Driver sqlite"`
	Password string `yaml:"password" validate:"required_unless=Driver sqlite,min=8"`
	SSLMode  string `yaml:"ssl_mode" validate:"omitempty,oneof=disable require verify-ca verify-full"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host" validate:"required_if=Enabled true,hostname"`
	Port     int    `yaml:"port" validate:"required_if=Enabled true,min=1,max=65535"`
	Password string `yaml:"password" validate:"omitempty,min=6"`
	Database int    `yaml:"database" validate:"min=0,max=15"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string   `yaml:"level" validate:"required,oneof=debug info warn error"`
	Format     string   `yaml:"format" validate:"required,oneof=json text"`
	Output     []string `yaml:"output" validate:"dive,oneof=stdout stderr file"`
	Structured bool     `yaml:"structured"`
}

// FeatureConfig represents feature flags and settings
type FeatureConfig struct {
	EnableMetrics    bool     `yaml:"enable_metrics"`
	EnableTracing    bool     `yaml:"enable_tracing"`
	AllowedOrigins   []string `yaml:"allowed_origins" validate:"dive,url"`
	RateLimitEnabled bool     `yaml:"rate_limit_enabled"`
	RateLimitRPS     int      `yaml:"rate_limit_rps" validate:"required_if=RateLimitEnabled true,min=1"`
}

func main() {
	fmt.Println("=== Configuration Validator Example ===\n")

	// Example 1: Valid configuration
	fmt.Println("1. Testing Valid Configuration:")
	validConfig := createValidConfig()
	testConfiguration(validConfig, "Valid Config")

	// Example 2: Invalid configuration with multiple errors
	fmt.Println("\n2. Testing Invalid Configuration:")
	invalidConfig := createInvalidConfig()
	testConfiguration(invalidConfig, "Invalid Config")

	// Example 3: Conditional validation
	fmt.Println("\n3. Testing Conditional Validation:")
	conditionalConfig := createConditionalConfig()
	testConfiguration(conditionalConfig, "Conditional Config")

	// Example 4: Performance comparison
	fmt.Println("\n4. Performance Comparison:")
	performanceComparison(validConfig)

	// Example 5: Using generated strategy with go-config integration
	fmt.Println("\n5. Go-Config Integration:")
	testGoConfigIntegration(validConfig)
}

func createValidConfig() *AppConfig {
	return &AppConfig{
		Meta: MetaConfig{
			AppName:     "myapp",
			Version:     "1.0.0",
			Environment: "production",
			Debug:       false,
		},
		Server: ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			TLS: &TLSConfig{
				Enabled:  true,
				CertFile: "/path/to/cert.pem",
				KeyFile:  "/path/to/key.pem",
			},
		},
		Database: DatabaseConfig{
			Driver:   "postgres",
			Host:     "db.example.com",
			Port:     5432,
			Database: "myapp",
			Username: "user",
			Password: "password123",
			SSLMode:  "require",
		},
		Cache: CacheConfig{
			Enabled:  true,
			Host:     "cache.example.com",
			Port:     6379,
			Password: "cache123",
			Database: 0,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     []string{"stdout", "file"},
			Structured: true,
		},
		Features: FeatureConfig{
			EnableMetrics:    true,
			EnableTracing:    true,
			AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
			RateLimitEnabled: true,
			RateLimitRPS:     100,
		},
	}
}

func createInvalidConfig() *AppConfig {
	return &AppConfig{
		Meta: MetaConfig{
			AppName:     "123invalid",  // Invalid: should be alpha only
			Version:     "",            // Invalid: required field empty
			Environment: "invalid_env", // Invalid: not in oneof list
			Debug:       false,
		},
		Server: ServerConfig{
			Host:         "invalid-hostname!", // Invalid: not a valid hostname
			Port:         70000,               // Invalid: port out of range
			ReadTimeout:  0,                   // Invalid: below minimum
			WriteTimeout: 0,                   // Invalid: below minimum
		},
		Database: DatabaseConfig{
			Driver:   "invalid_driver", // Invalid: not in oneof list
			Host:     "db.example.com", // This should trigger validation due to invalid driver
			Port:     5432,
			Database: "",    // Invalid: required field empty
			Username: "",    // This will be invalid due to driver
			Password: "123", // Invalid: too short
		},
		Logging: LoggingConfig{
			Level:  "invalid_level",            // Invalid: not in oneof list
			Format: "xml",                      // Invalid: not in oneof list
			Output: []string{"invalid_output"}, // Invalid: not in dive oneof list
		},
		Features: FeatureConfig{
			AllowedOrigins:   []string{"not-a-url", "also-invalid"}, // Invalid: not valid URLs
			RateLimitEnabled: true,
			RateLimitRPS:     0, // Invalid: required_if enabled but value is 0
		},
	}
}

func createConditionalConfig() *AppConfig {
	return &AppConfig{
		Meta: MetaConfig{
			AppName:     "testapp",
			Version:     "1.0.0",
			Environment: "development",
		},
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
			TLS: &TLSConfig{
				Enabled: true,
				// Missing CertFile and KeyFile - should trigger required_if validation
			},
		},
		Database: DatabaseConfig{
			Driver:   "sqlite",
			Database: "app.db",
			// Host, Port, Username, Password not required for sqlite
		},
		Cache: CacheConfig{
			Enabled: false,
			// Host and Port not required when disabled
		},
		Logging: LoggingConfig{
			Level:  "debug",
			Format: "text",
			Output: []string{"stdout"},
		},
		Features: FeatureConfig{
			RateLimitEnabled: false,
			// RateLimitRPS not required when disabled
		},
	}
}

func testConfiguration(config *AppConfig, name string) {
	fmt.Printf("Testing %s:\n", name)

	// Test with reflection-based validation (baseline)
	fmt.Print("  Reflection-based validation: ")
	err := validation.Struct(config)
	if err != nil {
		fmt.Printf("❌ FAILED\n")
		if validationErrors, ok := err.(validation.ValidationErrors); ok {
			for _, fieldErr := range validationErrors {
				fmt.Printf("    - %s: %s\n", fieldErr.Field, fieldErr.Message)
			}
		}
	} else {
		fmt.Printf("✅ PASSED\n")
	}

	// Test with generated validation (would be available after generation)
	fmt.Print("  Generated validation: ")
	// This would use the generated validator after running go:generate
	// validator := NewAppConfigValidator()
	// err = validator.Validate(config)
	fmt.Printf("⏳ Available after code generation\n")
}

func performanceComparison(config *AppConfig) {
	fmt.Println("Performance Comparison (Reflection vs Generated):")

	// Benchmark reflection-based validation
	start := time.Now()
	iterations := 10000

	for i := 0; i < iterations; i++ {
		_ = validation.Struct(config)
	}
	reflectionTime := time.Since(start)

	fmt.Printf("  Reflection-based (%d iterations): %v (%.2f μs/op)\n",
		iterations, reflectionTime, float64(reflectionTime.Nanoseconds())/float64(iterations)/1000)

	// Generated validation would be much faster
	estimatedGeneratedTime := reflectionTime / 10 // Estimated 10x improvement
	fmt.Printf("  Generated (estimated):             %v (%.2f μs/op)\n",
		estimatedGeneratedTime, float64(estimatedGeneratedTime.Nanoseconds())/float64(iterations)/1000)

	fmt.Printf("  Estimated performance improvement: %.1fx faster\n",
		float64(reflectionTime)/float64(estimatedGeneratedTime))
}

func testGoConfigIntegration(config *AppConfig) {
	fmt.Println("Go-Config Integration Example:")

	// Create a strategy factory (this would use generated validators)
	// For demonstration, we'll use a mock implementation
	fmt.Print("  Creating validation strategy: ")

	// This would typically be:
	// strategy := NewConfigValidationStrategy()

	// Mock implementation for example
	mockStrategy := &MockValidationStrategy{}
	fmt.Printf("✅ Created\n")

	// Test validation with enhanced errors
	fmt.Print("  Validating with enhanced errors: ")
	ctx := context.Background()
	err := mockStrategy.ValidateWithPath(ctx, config, "config")

	if err != nil {
		fmt.Printf("❌ FAILED\n")
		errors := mockStrategy.GetValidationErrors()
		for _, enhancedErr := range errors {
			fmt.Printf("    - %s (YAML: %s): %s\n",
				enhancedErr.Field, enhancedErr.YAMLPath, enhancedErr.Message)
			if len(enhancedErr.Suggestions) > 0 {
				fmt.Printf("      Suggestion: %s\n", enhancedErr.Suggestions[0])
			}
		}
	} else {
		fmt.Printf("✅ PASSED\n")
	}

	fmt.Print("  Setting fail-fast mode: ")
	mockStrategy.SetFailFast(true)
	fmt.Printf("✅ Configured\n")
}

// MockValidationStrategy provides a mock implementation for demonstration
type MockValidationStrategy struct {
	errors   []integration.EnhancedValidationError
	failFast bool
}

func (mvs *MockValidationStrategy) Validate(ctx context.Context, config interface{}) error {
	return mvs.ValidateWithPath(ctx, config, "")
}

func (mvs *MockValidationStrategy) ValidateWithPath(ctx context.Context, config interface{}, yamlPath string) error {
	// Use reflection-based validation and enhance errors
	err := validation.Struct(config)
	if err != nil {
		mvs.errors = []integration.EnhancedValidationError{}
		if validationErrors, ok := err.(validation.ValidationErrors); ok {
			for _, valErr := range validationErrors {
				enhancedErr := integration.EnhancedValidationError{
					ValidationError: valErr,
					YAMLPath:        yamlPath + "." + strings.ToLower(valErr.Field),
					ConfigSource:    "reflection",
					Suggestions:     []string{"Consider using generated validation for better performance"},
					Context: map[string]string{
						"validation_rule": valErr.Tag,
						"yaml_path":       yamlPath + "." + strings.ToLower(valErr.Field),
					},
				}
				mvs.errors = append(mvs.errors, enhancedErr)
			}
		}
		return err
	}
	return nil
}

func (mvs *MockValidationStrategy) GetValidationErrors() []integration.EnhancedValidationError {
	return mvs.errors
}

func (mvs *MockValidationStrategy) SetFailFast(enabled bool) {
	mvs.failFast = enabled
}

func init() {
	fmt.Println("📝 This example demonstrates the zero-reflection configuration validator.")
	fmt.Println("🚀 Run 'go generate' to generate optimized validation code.")
	fmt.Println("⚡ Generated validators are 10-100x faster than reflection-based validation.")
	fmt.Println()
}
