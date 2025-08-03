package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigAnalyzer_AnalyzeFile tests file analysis functionality
func TestConfigAnalyzer_AnalyzeFile(t *testing.T) {
	// Create a temporary test file
	testFile := createTestFile(t, `
package main

import "context"

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   ` + "`yaml:\"server\" validate:\"required\"`" + `
	Database DatabaseConfig ` + "`yaml:\"database\" validate:\"required\"`" + `
	API      APIConfig      ` + "`yaml:\"api\" validate:\"required\"`" + `
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host string ` + "`yaml:\"host\" validate:\"required,hostname\"`" + `
	Port int    ` + "`yaml:\"port\" validate:\"required,min=1,max=65535\"`" + `
	TLS  bool   ` + "`yaml:\"tls\"`" + `
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	URL      string ` + "`yaml:\"url\" validate:\"required,url\"`" + `
	Username string ` + "`yaml:\"username\" validate:\"required\"`" + `
	Password string ` + "`yaml:\"password\" validate:\"required,min=8\"`" + `
	MaxConns int    ` + "`yaml:\"max_connections\" validate:\"min=1,max=100\"`" + `
}

// APIConfig represents API configuration
type APIConfig struct {
	Key     string   ` + "`yaml:\"key\" validate:\"required,len=32\"`" + `
	Timeout int      ` + "`yaml:\"timeout\" validate:\"min=1,max=300\"`" + `
	Hosts   []string ` + "`yaml:\"hosts\" validate:\"dive,hostname\"`" + `
}
`)
	defer os.Remove(testFile)

	// Analyze the file
	analyzer := NewConfigAnalyzer()
	result, err := analyzer.AnalyzeFile(testFile)

	// Verify results
	if err != nil {
		t.Fatalf("Failed to analyze file: %v", err)
	}

	if result == nil {
		t.Fatal("Analysis result is nil")
	}

	// Check that we found the expected structs
	expectedStructs := []string{"Config", "ServerConfig", "DatabaseConfig", "APIConfig"}
	for _, structName := range expectedStructs {
		if _, exists := result.Structs[structName]; !exists {
			t.Errorf("Expected struct %s not found", structName)
		}
	}

	// Test specific struct analysis
	configStruct := result.Structs["Config"]
	if configStruct == nil {
		t.Fatal("Config struct not found")
	}

	if len(configStruct.Fields) != 3 {
		t.Errorf("Expected 3 fields in Config struct, got %d", len(configStruct.Fields))
	}

	// Test field analysis
	serverField := findField(configStruct.Fields, "Server")
	if serverField == nil {
		t.Fatal("Server field not found in Config struct")
	}

	if !serverField.IsNested {
		t.Error("Server field should be marked as nested")
	}

	if serverField.NestedType != "ServerConfig" {
		t.Errorf("Expected nested type ServerConfig, got %s", serverField.NestedType)
	}

	// Test validation rules
	if len(serverField.ValidationRules) == 0 {
		t.Error("Server field should have validation rules")
	}

	requiredRule := findValidationRule(serverField.ValidationRules, "required")
	if requiredRule == nil {
		t.Error("Server field should have required validation rule")
	}
}

// TestConfigAnalyzer_ValidationRuleParsing tests validation rule parsing
func TestConfigAnalyzer_ValidationRuleParsing(t *testing.T) {
	testFile := createTestFile(t, `
package test

type TestStruct struct {
	Email    string ` + "`validate:\"required,email\"`" + `
	Age      int    ` + "`validate:\"min=18,max=120\"`" + `
	Name     string ` + "`validate:\"required,min=2,max=50,alpha\"`" + `
	Category string ` + "`validate:\"oneof=admin user guest\"`" + `
	Website  string ` + "`validate:\"omitempty,url\"`" + `
}
`)
	defer os.Remove(testFile)

	analyzer := NewConfigAnalyzer()
	result, err := analyzer.AnalyzeFile(testFile)

	if err != nil {
		t.Fatalf("Failed to analyze file: %v", err)
	}

	testStruct := result.Structs["TestStruct"]
	if testStruct == nil {
		t.Fatal("TestStruct not found")
	}

	// Test email field
	emailField := findField(testStruct.Fields, "Email")
	if emailField == nil {
		t.Fatal("Email field not found")
	}

	expectedRules := []string{"required", "email"}
	for _, expectedRule := range expectedRules {
		if findValidationRule(emailField.ValidationRules, expectedRule) == nil {
			t.Errorf("Email field missing %s validation rule", expectedRule)
		}
	}

	// Test age field with parameters
	ageField := findField(testStruct.Fields, "Age")
	if ageField == nil {
		t.Fatal("Age field not found")
	}

	minRule := findValidationRule(ageField.ValidationRules, "min")
	if minRule == nil {
		t.Fatal("Age field missing min validation rule")
	}

	if minRule.Parameter != "18" {
		t.Errorf("Expected min parameter 18, got %s", minRule.Parameter)
	}

	maxRule := findValidationRule(ageField.ValidationRules, "max")
	if maxRule == nil {
		t.Fatal("Age field missing max validation rule")
	}

	if maxRule.Parameter != "120" {
		t.Errorf("Expected max parameter 120, got %s", maxRule.Parameter)
	}

	// Test oneof rule
	categoryField := findField(testStruct.Fields, "Category")
	if categoryField == nil {
		t.Fatal("Category field not found")
	}

	oneofRule := findValidationRule(categoryField.ValidationRules, "oneof")
	if oneofRule == nil {
		t.Fatal("Category field missing oneof validation rule")
	}

	if oneofRule.Parameter != "admin user guest" {
		t.Errorf("Expected oneof parameter 'admin user guest', got %s", oneofRule.Parameter)
	}
}

// TestConfigAnalyzer_NestedStructs tests nested struct analysis
func TestConfigAnalyzer_NestedStructs(t *testing.T) {
	testFile := createTestFile(t, `
package test

type Config struct {
	Database DatabaseConfig ` + "`yaml:\"database\" validate:\"required\"`" + `
	Cache    CacheConfig    ` + "`yaml:\"cache\"`" + `
}

type DatabaseConfig struct {
	Host     string ` + "`yaml:\"host\" validate:\"required,hostname\"`" + `
	Port     int    ` + "`yaml:\"port\" validate:\"required,min=1,max=65535\"`" + `
	Settings DBSettings ` + "`yaml:\"settings\" validate:\"required\"`" + `
}

type DBSettings struct {
	MaxConns int ` + "`yaml:\"max_connections\" validate:\"min=1,max=100\"`" + `
	Timeout  int ` + "`yaml:\"timeout\" validate:\"min=1\"`" + `
}

type CacheConfig struct {
	TTL  int    ` + "`yaml:\"ttl\" validate:\"min=1\"`" + `
	Type string ` + "`yaml:\"type\" validate:\"oneof=redis memory\"`" + `
}
`)
	defer os.Remove(testFile)

	analyzer := NewConfigAnalyzer()
	result, err := analyzer.AnalyzeFile(testFile)

	if err != nil {
		t.Fatalf("Failed to analyze file: %v", err)
	}

	// Check dependency graph
	if len(result.Dependencies) == 0 {
		t.Error("Expected dependencies to be built")
	}

	configDeps := result.Dependencies["Config"]
	expectedDeps := []string{"DatabaseConfig", "CacheConfig"}
	
	for _, expectedDep := range expectedDeps {
		found := false
		for _, dep := range configDeps {
			if dep == expectedDep {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected dependency %s not found in Config dependencies", expectedDep)
		}
	}

	// Check nested struct fields
	configStruct := result.Structs["Config"]
	dbField := findField(configStruct.Fields, "Database")
	if !dbField.IsNested {
		t.Error("Database field should be marked as nested")
	}
	if dbField.NestedType != "DatabaseConfig" {
		t.Errorf("Expected nested type DatabaseConfig, got %s", dbField.NestedType)
	}
}

// TestConfigAnalyzer_TypeAnalysis tests Go type analysis
func TestConfigAnalyzer_TypeAnalysis(t *testing.T) {
	testFile := createTestFile(t, `
package test

type Config struct {
	Name     string            ` + "`validate:\"required\"`" + `
	Port     int               ` + "`validate:\"min=1\"`" + `
	Enabled  bool              ` + "`validate:\"required\"`" + `
	Tags     []string          ` + "`validate:\"dive,min=1\"`" + `
	Metadata map[string]string ` + "`validate:\"dive,keys,alpha,endkeys,required\"`" + `
	OptPtr   *string           ` + "`validate:\"omitempty,min=1\"`" + `
}
`)
	defer os.Remove(testFile)

	analyzer := NewConfigAnalyzer()
	result, err := analyzer.AnalyzeFile(testFile)

	if err != nil {
		t.Fatalf("Failed to analyze file: %v", err)
	}

	configStruct := result.Structs["Config"]
	
	// Test string field
	nameField := findField(configStruct.Fields, "Name")
	if nameField.GoType.Kind != TypeString {
		t.Errorf("Expected string type for Name field, got %v", nameField.GoType.Kind)
	}

	// Test int field
	portField := findField(configStruct.Fields, "Port")
	if portField.GoType.Kind != TypeInt {
		t.Errorf("Expected int type for Port field, got %v", portField.GoType.Kind)
	}

	// Test bool field
	enabledField := findField(configStruct.Fields, "Enabled")
	if enabledField.GoType.Kind != TypeBool {
		t.Errorf("Expected bool type for Enabled field, got %v", enabledField.GoType.Kind)
	}

	// Test slice field
	tagsField := findField(configStruct.Fields, "Tags")
	if !tagsField.GoType.IsSlice {
		t.Error("Tags field should be marked as slice")
	}
	if tagsField.GoType.ElemType.Kind != TypeString {
		t.Errorf("Expected string element type for Tags field, got %v", tagsField.GoType.ElemType.Kind)
	}

	// Test map field
	metadataField := findField(configStruct.Fields, "Metadata")
	if !metadataField.GoType.IsMap {
		t.Error("Metadata field should be marked as map")
	}
	if metadataField.GoType.KeyType.Kind != TypeString {
		t.Errorf("Expected string key type for Metadata field, got %v", metadataField.GoType.KeyType.Kind)
	}

	// Test pointer field
	optPtrField := findField(configStruct.Fields, "OptPtr")
	if !optPtrField.GoType.IsPointer {
		t.Error("OptPtr field should be marked as pointer")
	}
	if optPtrField.GoType.ElemType.Kind != TypeString {
		t.Errorf("Expected string element type for OptPtr field, got %v", optPtrField.GoType.ElemType.Kind)
	}
}

// TestConfigAnalyzer_YAMLPaths tests YAML path generation
func TestConfigAnalyzer_YAMLPaths(t *testing.T) {
	testFile := createTestFile(t, `
package test

type Config struct {
	Server   ServerConfig ` + "`yaml:\"server\"`" + `
	Database DatabaseConfig ` + "`yaml:\"db\"`" + `
}

type ServerConfig struct {
	Host string ` + "`yaml:\"hostname\"`" + `
	Port int    ` + "`yaml:\"port_number\"`" + `
}

type DatabaseConfig struct {
	URL string ` + "`yaml:\"connection_url\"`" + `
}
`)
	defer os.Remove(testFile)

	analyzer := NewConfigAnalyzer()
	result, err := analyzer.AnalyzeFile(testFile)

	if err != nil {
		t.Fatalf("Failed to analyze file: %v", err)
	}

	// Check YAML paths
	expectedPaths := map[string]string{
		"Config.Server":   "server",
		"Config.Database": "db",
		"ServerConfig.Host": "server.hostname",
		"ServerConfig.Port": "server.port_number",
		"DatabaseConfig.URL": "db.connection_url",
	}

	for fieldKey, expectedPath := range expectedPaths {
		if actualPath, exists := result.YAMLPaths[fieldKey]; !exists {
			t.Errorf("YAML path for %s not found", fieldKey)
		} else if actualPath != expectedPath {
			t.Errorf("Expected YAML path %s for %s, got %s", expectedPath, fieldKey, actualPath)
		}
	}
}

// TestConfigAnalyzer_CrossFieldValidation tests cross-field validation analysis
func TestConfigAnalyzer_CrossFieldValidation(t *testing.T) {
	testFile := createTestFile(t, `
package test

type User struct {
	Password        string ` + "`validate:\"required,min=8\"`" + `
	ConfirmPassword string ` + "`validate:\"required,eqfield=Password\"`" + `
	Age             int    ` + "`validate:\"required,min=18\"`" + `
	ParentEmail     string ` + "`validate:\"required_if=Age 17,omitempty,email\"`" + `
}
`)
	defer os.Remove(testFile)

	analyzer := NewConfigAnalyzer()
	result, err := analyzer.AnalyzeFile(testFile)

	if err != nil {
		t.Fatalf("Failed to analyze file: %v", err)
	}

	userStruct := result.Structs["User"]
	
	// Test eqfield validation
	confirmField := findField(userStruct.Fields, "ConfirmPassword")
	eqfieldRule := findValidationRule(confirmField.ValidationRules, "eqfield")
	if eqfieldRule == nil {
		t.Fatal("ConfirmPassword field missing eqfield validation rule")
	}

	if !eqfieldRule.IsConditional {
		t.Error("eqfield rule should be marked as conditional")
	}

	if len(eqfieldRule.DependsOn) == 0 {
		t.Error("eqfield rule should have dependencies")
	}

	if eqfieldRule.DependsOn[0] != "Password" {
		t.Errorf("Expected dependency on Password, got %s", eqfieldRule.DependsOn[0])
	}

	// Test required_if validation
	parentEmailField := findField(userStruct.Fields, "ParentEmail")
	requiredIfRule := findValidationRule(parentEmailField.ValidationRules, "required_if")
	if requiredIfRule == nil {
		t.Fatal("ParentEmail field missing required_if validation rule")
	}

	if !requiredIfRule.IsConditional {
		t.Error("required_if rule should be marked as conditional")
	}
}

// Helper functions

func createTestFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.go")
	
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	return filename
}

func findField(fields []FieldInfo, name string) *FieldInfo {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func findValidationRule(rules []ValidationRule, name string) *ValidationRule {
	for i := range rules {
		if rules[i].Name == name {
			return &rules[i]
		}
	}
	return nil
}

// Benchmark tests

func BenchmarkConfigAnalyzer_AnalyzeFile(b *testing.B) {
	content := `
package benchmark

type Config struct {
	Server   ServerConfig   ` + "`yaml:\"server\" validate:\"required\"`" + `
	Database DatabaseConfig ` + "`yaml:\"database\" validate:\"required\"`" + `
	API      APIConfig      ` + "`yaml:\"api\" validate:\"required\"`" + `
}

type ServerConfig struct {
	Host string ` + "`yaml:\"host\" validate:\"required,hostname\"`" + `
	Port int    ` + "`yaml:\"port\" validate:\"required,min=1,max=65535\"`" + `
}

type DatabaseConfig struct {
	URL      string ` + "`yaml:\"url\" validate:\"required,url\"`" + `
	Username string ` + "`yaml:\"username\" validate:\"required\"`" + `
	Password string ` + "`yaml:\"password\" validate:\"required,min=8\"`" + `
}

type APIConfig struct {
	Key     string   ` + "`yaml:\"key\" validate:\"required,len=32\"`" + `
	Hosts   []string ` + "`yaml:\"hosts\" validate:\"dive,hostname\"`" + `
}
`

	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "benchmark.go")
	
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create benchmark file: %v", err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		analyzer := NewConfigAnalyzer()
		_, err := analyzer.AnalyzeFile(filename)
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
	}
}

func BenchmarkConfigAnalyzer_ParseValidationRules(b *testing.B) {
	analyzer := NewConfigAnalyzer()
	validateTag := "required,min=8,max=50,alpha,oneof=admin user guest"
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		rules := analyzer.parseValidationRules(validateTag)
		if len(rules) == 0 {
			b.Fatal("No validation rules parsed")
		}
	}
}

// Integration tests

func TestConfigAnalyzer_RealWorldExample(t *testing.T) {
	testFile := createTestFile(t, `
package config

import "time"

// AppConfig represents the complete application configuration
type AppConfig struct {
	Meta     MetaConfig     ` + "`yaml:\"meta\" validate:\"required\"`" + `
	Server   ServerConfig   ` + "`yaml:\"server\" validate:\"required\"`" + `
	Database DatabaseConfig ` + "`yaml:\"database\" validate:\"required\"`" + `
	Redis    RedisConfig    ` + "`yaml:\"redis\"`" + `
	Logging  LoggingConfig  ` + "`yaml:\"logging\" validate:\"required\"`" + `
	Features FeatureConfig  ` + "`yaml:\"features\"`" + `
}

type MetaConfig struct {
	AppName     string ` + "`yaml:\"app_name\" validate:\"required,alpha\"`" + `
	Version     string ` + "`yaml:\"version\" validate:\"required\"`" + `
	Environment string ` + "`yaml:\"environment\" validate:\"required,oneof=development staging production\"`" + `
	Debug       bool   ` + "`yaml:\"debug\"`" + `
}

type ServerConfig struct {
	Host         string        ` + "`yaml:\"host\" validate:\"required,hostname\"`" + `
	Port         int           ` + "`yaml:\"port\" validate:\"required,min=1,max=65535\"`" + `
	ReadTimeout  time.Duration ` + "`yaml:\"read_timeout\" validate:\"min=1s\"`" + `
	WriteTimeout time.Duration ` + "`yaml:\"write_timeout\" validate:\"min=1s\"`" + `
	TLS          *TLSConfig    ` + "`yaml:\"tls\"`" + `
}

type TLSConfig struct {
	Enabled  bool   ` + "`yaml:\"enabled\"`" + `
	CertFile string ` + "`yaml:\"cert_file\" validate:\"required_if=Enabled true\"`" + `
	KeyFile  string ` + "`yaml:\"key_file\" validate:\"required_if=Enabled true\"`" + `
}

type DatabaseConfig struct {
	Driver   string ` + "`yaml:\"driver\" validate:\"required,oneof=postgres mysql sqlite\"`" + `
	Host     string ` + "`yaml:\"host\" validate:\"required_unless=Driver sqlite,hostname\"`" + `
	Port     int    ` + "`yaml:\"port\" validate:\"required_unless=Driver sqlite,min=1,max=65535\"`" + `
	Database string ` + "`yaml:\"database\" validate:\"required\"`" + `
	Username string ` + "`yaml:\"username\" validate:\"required_unless=Driver sqlite\"`" + `
	Password string ` + "`yaml:\"password\" validate:\"required_unless=Driver sqlite,min=8\"`" + `
	SSLMode  string ` + "`yaml:\"ssl_mode\" validate:\"omitempty,oneof=disable require verify-ca verify-full\"`" + `
}

type RedisConfig struct {
	Enabled  bool   ` + "`yaml:\"enabled\"`" + `
	Host     string ` + "`yaml:\"host\" validate:\"required_if=Enabled true,hostname\"`" + `
	Port     int    ` + "`yaml:\"port\" validate:\"required_if=Enabled true,min=1,max=65535\"`" + `
	Password string ` + "`yaml:\"password\" validate:\"omitempty,min=6\"`" + `
	Database int    ` + "`yaml:\"database\" validate:\"min=0,max=15\"`" + `
}

type LoggingConfig struct {
	Level      string   ` + "`yaml:\"level\" validate:\"required,oneof=debug info warn error\"`" + `
	Format     string   ` + "`yaml:\"format\" validate:\"required,oneof=json text\"`" + `
	Output     []string ` + "`yaml:\"output\" validate:\"dive,oneof=stdout stderr file\"`" + `
	Structured bool     ` + "`yaml:\"structured\"`" + `
}

type FeatureConfig struct {
	EnableMetrics    bool     ` + "`yaml:\"enable_metrics\"`" + `
	EnableTracing    bool     ` + "`yaml:\"enable_tracing\"`" + `
	AllowedOrigins   []string ` + "`yaml:\"allowed_origins\" validate:\"dive,url\"`" + `
	RateLimitEnabled bool     ` + "`yaml:\"rate_limit_enabled\"`" + `
	RateLimitRPS     int      ` + "`yaml:\"rate_limit_rps\" validate:\"required_if=RateLimitEnabled true,min=1\"`" + `
}
`)
	defer os.Remove(testFile)

	analyzer := NewConfigAnalyzer()
	result, err := analyzer.AnalyzeFile(testFile)

	if err != nil {
		t.Fatalf("Failed to analyze real-world config: %v", err)
	}

	// Verify all expected structs were found
	expectedStructs := []string{
		"AppConfig", "MetaConfig", "ServerConfig", "TLSConfig",
		"DatabaseConfig", "RedisConfig", "LoggingConfig", "FeatureConfig",
	}

	for _, structName := range expectedStructs {
		if _, exists := result.Structs[structName]; !exists {
			t.Errorf("Expected struct %s not found", structName)
		}
	}

	// Verify complex validation rules
	dbConfig := result.Structs["DatabaseConfig"]
	hostField := findField(dbConfig.Fields, "Host")
	requiredUnlessRule := findValidationRule(hostField.ValidationRules, "required_unless")
	
	if requiredUnlessRule == nil {
		t.Error("DatabaseConfig.Host should have required_unless rule")
	}

	if !requiredUnlessRule.IsConditional {
		t.Error("required_unless should be marked as conditional")
	}

	// Verify nested dependencies
	appDeps := result.Dependencies["AppConfig"]
	if len(appDeps) < 5 {
		t.Errorf("AppConfig should have at least 5 dependencies, got %d", len(appDeps))
	}

	// Verify YAML paths for nested structures
	tlsCertPath := result.YAMLPaths["TLSConfig.CertFile"]
	if !strings.Contains(tlsCertPath, "tls") || !strings.Contains(tlsCertPath, "cert_file") {
		t.Errorf("Unexpected YAML path for TLS cert file: %s", tlsCertPath)
	}
}

// Error case tests

func TestConfigAnalyzer_InvalidFile(t *testing.T) {
	analyzer := NewConfigAnalyzer()
	_, err := analyzer.AnalyzeFile("nonexistent.go")
	
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestConfigAnalyzer_EmptyFile(t *testing.T) {
	testFile := createTestFile(t, "package test\n")
	defer os.Remove(testFile)

	analyzer := NewConfigAnalyzer()
	result, err := analyzer.AnalyzeFile(testFile)

	if err != nil {
		t.Fatalf("Unexpected error for empty file: %v", err)
	}

	if len(result.Structs) != 0 {
		t.Error("Expected no structs in empty file")
	}
}

func TestConfigAnalyzer_NoValidationTags(t *testing.T) {
	testFile := createTestFile(t, `
package test

type Config struct {
	Name string
	Port int
}
`)
	defer os.Remove(testFile)

	analyzer := NewConfigAnalyzer()
	result, err := analyzer.AnalyzeFile(testFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should not find Config struct since it has no validation tags
	if _, exists := result.Structs["Config"]; exists {
		t.Error("Should not include structs without validation tags")
	}
}