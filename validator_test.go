package validation

import (
	"testing"
)

// Embedded struct definitions
type BaseConfig struct {
	Host string `yaml:"host" validate:"required" default:"localhost"`
	Port int    `yaml:"port" validate:"required,min=1,max=65535" default:"bad"`
}

type AuthConfig struct {
	Username string `yaml:"username" validate:"required"`
	Password string `yaml:"password" validate:"required"`
}

type AppConfig struct {
	Name        string     `yaml:"name" validate:"required,minlen=3,maxlen=50"`
	Environment string     `yaml:"environment" validate:"oneof=dev|staging|prod"`
	Version     string     `yaml:"version" validate:"required,regex=^v[0-9]+\\.[0-9]+\\.[0-9]+$"`
	Sub         *SubConfig `yaml:"sub" validate:"required"`
	// Test embedded structs - this should cause the issue
	Redis RedisConfig `yaml:"redis" validate:"required"`
}

type SubConfig struct {
	Foo string `yaml:"foo" validate:"required,minlen=3"`
}

// RedisConfig with embedded structs
type RedisConfig struct {
	BaseConfig        // Embedded - fields should be flattened
	AuthConfig        // Embedded - fields should be flattened
	Database   int    `yaml:"database" validate:"min=0,max=15"`
	Timeout    string `yaml:"timeout" validate:"required"`
}

func TestConfig(t *testing.T) {
	cfg := &AppConfig{}

	// assert.NoError(t, err)
	// assert.Equal(t, cfg.Name, "go-validate-demo")
	// assert.Equal(t, cfg.Environment, "dev")
	// assert.Equal(t, cfg.Version, "v1.0.0")
	// assert.NotNil(t, cfg.Sub, "pointer struct should not be nil")
	// assert.Equal(t, cfg.Sub.Foo, "test-value")
	// assert.Equal(t, cfg.Redis.Host, "localhost")
	// assert.Equal(t, cfg.Redis.Port, 6379)
	// assert.Equal(t, cfg.Redis.Username, "default")
	// litter.Dump(cfg)
}

// func TestPointerStructValidation(t *testing.T) {
// 	// Test case where pointer struct is missing (should fail validation)
// 	cfg := &AppConfig{
// 		Name:        "test-app",
// 		Environment: "dev",
// 		Version:     "v1.0.0",
// 		Sub:         nil, // This should fail validation due to "required" tag
// 		Redis: RedisConfig{
// 			BaseConfig: BaseConfig{Host: "localhost", Port: 6379},
// 			AuthConfig: AuthConfig{Username: "user", Password: "pass"},
// 			Timeout:    "5s",
// 		},
// 	}

// 	err := validation.NewUnifiedValidator(validation.DefaultValidatorConfig()).Validate(cfg)
// 	assert.Error(t, err, "validation should fail when required pointer struct is nil")
// 	assert.Contains(t, err.Error(), "Sub", "error should mention the Sub field")

// 	// Test case where pointer struct is present and valid
// 	cfg.Sub = &SubConfig{Foo: "valid-value"}
// 	err = validation.NewUnifiedValidator(validation.DefaultValidatorConfig()).Validate(cfg)
// 	assert.NoError(t, err, "validation should pass when pointer struct is properly set")
// }
