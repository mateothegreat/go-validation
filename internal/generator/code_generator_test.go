package generator

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mateothegreat/go-validation/internal/analyzer"
)

// TestCodeGenerator_Generate tests the complete code generation process
func TestCodeGenerator_Generate(t *testing.T) {
	// Create test analysis result
	analysisResult := createTestAnalysisResult()

	// Create temporary output directory
	outputDir := t.TempDir()

	// Configure generator
	options := GeneratorOptions{
		PackageName:         "testpkg",
		OutputDir:           outputDir,
		GenerateStrategies:  true,
		EnableOptimizations: true,
		IncludeDebugInfo:    false,
		FailFast:            false,
		GenerateTests:       false,
	}

	// Generate code
	generator := NewCodeGenerator(analysisResult, options)
	err := generator.Generate()
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Verify generated files exist
	expectedFiles := []string{
		"testconfig_validator_gen.go",
		"validation_strategy_gen.go",
	}

	for _, filename := range expectedFiles {
		filepath := filepath.Join(outputDir, filename)
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			t.Errorf("Expected generated file %s not found", filename)
		}
	}

	// Verify generated code compiles
	err = verifyGeneratedCodeCompiles(t, outputDir)
	if err != nil {
		t.Errorf("Generated code does not compile: %v", err)
	}
}

// TestCodeGenerator_ValidatorStruct tests validator struct generation
func TestCodeGenerator_ValidatorStruct(t *testing.T) {
	analysisResult := createTestAnalysisResult()
	options := GeneratorOptions{
		PackageName:         "test",
		EnableOptimizations: true,
	}

	generator := NewCodeGenerator(analysisResult, options)

	// Generate validator struct for TestConfig
	decl := generator.generateValidatorStruct("TestConfig")

	// Verify the declaration
	if decl.Tok != token.TYPE {
		t.Error("Expected TYPE declaration")
	}

	if len(decl.Specs) != 1 {
		t.Error("Expected exactly one type spec")
	}

	typeSpec := decl.Specs[0].(*ast.TypeSpec)
	if typeSpec.Name.Name != "TestConfigValidator" {
		t.Errorf("Expected validator name TestConfigValidator, got %s", typeSpec.Name.Name)
	}

	structType := typeSpec.Type.(*ast.StructType)
	if len(structType.Fields.List) < 1 {
		t.Error("Expected at least one field in validator struct")
	}

	// Check for errors field
	errorsField := structType.Fields.List[0]
	if len(errorsField.Names) == 0 || errorsField.Names[0].Name != "errors" {
		t.Error("Expected errors field in validator struct")
	}
}

// TestCodeGenerator_ValidateMethod tests validation method generation
func TestCodeGenerator_ValidateMethod(t *testing.T) {
	analysisResult := createTestAnalysisResult()
	options := GeneratorOptions{
		PackageName: "test",
		FailFast:    true,
	}

	generator := NewCodeGenerator(analysisResult, options)
	structInfo := analysisResult.Structs["TestConfig"]

	// Generate validate method
	method := generator.generateValidateMethod("TestConfig", structInfo)

	// Verify method signature
	if method.Name.Name != "Validate" {
		t.Errorf("Expected method name Validate, got %s", method.Name.Name)
	}

	// Check receiver
	if len(method.Recv.List) != 1 {
		t.Error("Expected exactly one receiver")
	}

	receiver := method.Recv.List[0]
	if len(receiver.Names) == 0 || receiver.Names[0].Name != "v" {
		t.Error("Expected receiver name 'v'")
	}

	// Check parameters
	if len(method.Type.Params.List) != 1 {
		t.Error("Expected exactly one parameter")
	}

	param := method.Type.Params.List[0]
	if len(param.Names) == 0 || param.Names[0].Name != "cfg" {
		t.Error("Expected parameter name 'cfg'")
	}

	// Check return type
	if len(method.Type.Results.List) != 1 {
		t.Error("Expected exactly one return value")
	}

	// Verify method body has statements
	if len(method.Body.List) == 0 {
		t.Error("Expected non-empty method body")
	}
}

// TestCodeGenerator_FieldValidation tests field validation generation
func TestCodeGenerator_FieldValidation(t *testing.T) {
	analysisResult := createTestAnalysisResult()
	options := GeneratorOptions{
		PackageName:         "test",
		EnableOptimizations: true,
	}

	generator := NewCodeGenerator(analysisResult, options)
	structInfo := analysisResult.Structs["TestConfig"]

	// Find email field
	var emailField *analyzer.FieldInfo
	for _, field := range structInfo.Fields {
		if field.Name == "Email" {
			emailField = &field
			break
		}
	}

	if emailField == nil {
		t.Fatal("Email field not found in test data")
	}

	// Generate field validation
	stmts := generator.generateFieldValidation("TestConfig", emailField)

	if len(stmts) == 0 {
		t.Error("Expected validation statements to be generated")
	}

	// Verify statements contain validation logic
	hasValidationCall := false
	for _, stmt := range stmts {
		if containsValidationCall(stmt) {
			hasValidationCall = true
			break
		}
	}

	if !hasValidationCall {
		t.Error("Expected validation call in generated statements")
	}
}

// TestCodeGenerator_OptimizedValidation tests optimized validation generation
func TestCodeGenerator_OptimizedValidation(t *testing.T) {
	// Create field with multiple validation rules
	field := analyzer.FieldInfo{
		Name: "Username",
		Type: "string",
		GoType: analyzer.GoType{
			Kind: analyzer.TypeString,
			Name: "string",
		},
		ValidationRules: []analyzer.ValidationRule{
			{Name: "required", Parameter: ""},
			{Name: "min", Parameter: "3"},
			{Name: "max", Parameter: "50"},
			{Name: "alpha", Parameter: ""},
		},
	}

	analysisResult := &analyzer.AnalysisResult{
		Structs: map[string]*analyzer.StructInfo{
			"TestStruct": {
				Name:   "TestStruct",
				Fields: []analyzer.FieldInfo{field},
			},
		},
		PackageName: "test",
	}

	options := GeneratorOptions{
		PackageName:         "test",
		EnableOptimizations: true,
	}

	generator := NewCodeGenerator(analysisResult, options)

	// Generate optimized validation for required rule
	fieldAccess := &ast.SelectorExpr{
		X:   ast.NewIdent("cfg"),
		Sel: ast.NewIdent("Username"),
	}

	stmts := generator.generateRequiredValidation(&field, fieldAccess)

	if len(stmts) == 0 {
		t.Error("Expected optimized required validation statements")
	}

	// Verify it's an if statement with string comparison
	ifStmt, ok := stmts[0].(*ast.IfStmt)
	if !ok {
		t.Error("Expected if statement for required validation")
	}

	// Check for string comparison in condition
	binaryExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		t.Error("Expected binary expression in if condition")
	}

	if binaryExpr.Op != token.EQL {
		t.Error("Expected equality operator in condition")
	}
}

// TestCodeGenerator_NestedValidation tests nested struct validation
func TestCodeGenerator_NestedValidation(t *testing.T) {
	// Create nested field
	field := analyzer.FieldInfo{
		Name:       "Server",
		Type:       "ServerConfig",
		IsNested:   true,
		NestedType: "ServerConfig",
	}

	generator := &CodeGenerator{}
	fieldAccess := &ast.SelectorExpr{
		X:   ast.NewIdent("cfg"),
		Sel: ast.NewIdent("Server"),
	}

	stmts := generator.generateNestedValidation(&field, fieldAccess)

	if len(stmts) == 0 {
		t.Error("Expected nested validation statements")
	}

	// Verify structure of nested validation
	ifStmt, ok := stmts[0].(*ast.IfStmt)
	if !ok {
		t.Error("Expected if statement for nested validation")
	}

	// Check that it creates a nested validator
	assignStmt, ok := ifStmt.Init.(*ast.AssignStmt)
	if !ok {
		t.Error("Expected assignment statement in if init")
	}

	if len(assignStmt.Lhs) == 0 {
		t.Error("Expected left-hand side in assignment")
	}

	if ident, ok := assignStmt.Lhs[0].(*ast.Ident); !ok || ident.Name != "nestedValidator" {
		t.Error("Expected nestedValidator assignment")
	}
}

// TestCodeGenerator_PointerHandling tests pointer field handling
func TestCodeGenerator_PointerHandling(t *testing.T) {
	// Create pointer field
	field := analyzer.FieldInfo{
		Name: "OptionalValue",
		Type: "*string",
		GoType: analyzer.GoType{
			Kind:      analyzer.TypePointer,
			IsPointer: true,
			ElemType: &analyzer.GoType{
				Kind: analyzer.TypeString,
				Name: "string",
			},
		},
		ValidationRules: []analyzer.ValidationRule{
			{Name: "required", Parameter: ""},
		},
	}

	generator := &CodeGenerator{}
	fieldAccess := &ast.SelectorExpr{
		X:   ast.NewIdent("cfg"),
		Sel: ast.NewIdent("OptionalValue"),
	}

	stmts := generator.generatePointerNilCheck(&field, fieldAccess)

	if len(stmts) == 0 {
		t.Error("Expected pointer nil check statements")
	}

	// Verify nil check structure
	ifStmt, ok := stmts[0].(*ast.IfStmt)
	if !ok {
		t.Error("Expected if statement for nil check")
	}

	// Check for nil comparison
	binaryExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		t.Error("Expected binary expression for nil check")
	}

	if binaryExpr.Op != token.EQL {
		t.Error("Expected equality operator for nil check")
	}

	// Check for nil identifier
	if ident, ok := binaryExpr.Y.(*ast.Ident); !ok || ident.Name != "nil" {
		t.Error("Expected comparison with nil")
	}
}

// TestCodeGenerator_OneOfValidation tests oneof validation generation
func TestCodeGenerator_OneOfValidation(t *testing.T) {
	field := analyzer.FieldInfo{
		Name: "Role",
		Type: "string",
		GoType: analyzer.GoType{
			Kind: analyzer.TypeString,
			Name: "string",
		},
	}

	rule := analyzer.ValidationRule{
		Name:      "oneof",
		Parameter: "admin user guest",
	}

	generator := &CodeGenerator{}
	fieldAccess := &ast.SelectorExpr{
		X:   ast.NewIdent("cfg"),
		Sel: ast.NewIdent("Role"),
	}

	stmts := generator.generateOneOfValidation(&field, rule, fieldAccess)

	if len(stmts) == 0 {
		t.Error("Expected oneof validation statements")
	}

	// Verify if statement structure
	ifStmt, ok := stmts[0].(*ast.IfStmt)
	if !ok {
		t.Error("Expected if statement for oneof validation")
	}

	// The condition should be a complex expression with multiple != comparisons
	// connected by && operators
	if ifStmt.Cond == nil {
		t.Error("Expected condition in oneof validation")
	}
}

// TestCodeGenerator_AlphaValidation tests alphabetic validation generation
func TestCodeGenerator_AlphaValidation(t *testing.T) {
	field := analyzer.FieldInfo{
		Name: "Name",
		Type: "string",
	}

	generator := &CodeGenerator{}
	fieldAccess := &ast.SelectorExpr{
		X:   ast.NewIdent("cfg"),
		Sel: ast.NewIdent("Name"),
	}

	stmts := generator.generateAlphaValidation(&field, fieldAccess)

	if len(stmts) == 0 {
		t.Error("Expected alpha validation statements")
	}

	// Verify range statement for character iteration
	rangeStmt, ok := stmts[0].(*ast.RangeStmt)
	if !ok {
		t.Error("Expected range statement for alpha validation")
	}

	if rangeStmt.Value == nil {
		t.Error("Expected value in range statement")
	}

	if ident, ok := rangeStmt.Value.(*ast.Ident); !ok || ident.Name != "r" {
		t.Error("Expected range value 'r'")
	}
}

// Helper functions

func createTestAnalysisResult() *analyzer.AnalysisResult {
	return &analyzer.AnalysisResult{
		Structs: map[string]*analyzer.StructInfo{
			"TestConfig": {
				Name:    "TestConfig",
				Package: "test",
				Fields: []analyzer.FieldInfo{
					{
						Name: "Name",
						Type: "string",
						GoType: analyzer.GoType{
							Kind: analyzer.TypeString,
							Name: "string",
						},
						ValidationRules: []analyzer.ValidationRule{
							{Name: "required", Parameter: ""},
							{Name: "min", Parameter: "2"},
							{Name: "max", Parameter: "50"},
						},
					},
					{
						Name: "Email",
						Type: "string",
						GoType: analyzer.GoType{
							Kind: analyzer.TypeString,
							Name: "string",
						},
						ValidationRules: []analyzer.ValidationRule{
							{Name: "required", Parameter: ""},
							{Name: "email", Parameter: ""},
						},
					},
					{
						Name: "Port",
						Type: "int",
						GoType: analyzer.GoType{
							Kind: analyzer.TypeInt,
							Name: "int",
						},
						ValidationRules: []analyzer.ValidationRule{
							{Name: "required", Parameter: ""},
							{Name: "min", Parameter: "1"},
							{Name: "max", Parameter: "65535"},
						},
					},
				},
			},
		},
		Dependencies: map[string][]string{
			"TestConfig": {},
		},
		YAMLPaths: map[string]string{
			"TestConfig.Name":  "name",
			"TestConfig.Email": "email",
			"TestConfig.Port":  "port",
		},
		Imports: []string{
			"fmt",
			"github.com/mateothegreat/go-validation",
		},
		PackageName: "test",
	}
}

func verifyGeneratedCodeCompiles(t *testing.T, outputDir string) error {
	// Walk through generated files and try to parse them
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Parse the file to check for syntax errors
		fset := token.NewFileSet()
		_, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return parseErr
		}

		return nil
	})
}

func containsValidationCall(stmt ast.Stmt) bool {
	// Simple check for validation function calls
	var found bool
	ast.Inspect(stmt, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if ident.Name == "validation" {
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	return found
}

// Benchmark tests

func BenchmarkCodeGenerator_Generate(b *testing.B) {
	analysisResult := createTestAnalysisResult()
	outputDir := b.TempDir()

	options := GeneratorOptions{
		PackageName:         "benchmark",
		OutputDir:           outputDir,
		EnableOptimizations: true,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		generator := NewCodeGenerator(analysisResult, options)
		err := generator.Generate()
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}

		// Clean up for next iteration
		os.RemoveAll(outputDir)
		os.MkdirAll(outputDir, 0o755)
	}
}

func BenchmarkCodeGenerator_ValidateMethod(b *testing.B) {
	analysisResult := createTestAnalysisResult()
	options := GeneratorOptions{PackageName: "benchmark"}
	generator := NewCodeGenerator(analysisResult, options)
	structInfo := analysisResult.Structs["TestConfig"]

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = generator.generateValidateMethod("TestConfig", structInfo)
	}
}

func BenchmarkCodeGenerator_FieldValidation(b *testing.B) {
	analysisResult := createTestAnalysisResult()
	options := GeneratorOptions{PackageName: "benchmark"}
	generator := NewCodeGenerator(analysisResult, options)
	structInfo := analysisResult.Structs["TestConfig"]

	field := &structInfo.Fields[0] // Name field

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = generator.generateFieldValidation("TestConfig", field)
	}
}

// Integration tests

func TestCodeGenerator_ComplexIntegration(t *testing.T) {
	// Create complex analysis result with nested structs
	analysisResult := &analyzer.AnalysisResult{
		Structs: map[string]*analyzer.StructInfo{
			"AppConfig": {
				Name: "AppConfig",
				Fields: []analyzer.FieldInfo{
					{
						Name:       "Server",
						Type:       "ServerConfig",
						IsNested:   true,
						NestedType: "ServerConfig",
						ValidationRules: []analyzer.ValidationRule{
							{Name: "required", Parameter: ""},
						},
					},
					{
						Name:       "Database",
						Type:       "DatabaseConfig",
						IsNested:   true,
						NestedType: "DatabaseConfig",
						ValidationRules: []analyzer.ValidationRule{
							{Name: "required", Parameter: ""},
						},
					},
				},
			},
			"ServerConfig": {
				Name: "ServerConfig",
				Fields: []analyzer.FieldInfo{
					{
						Name:   "Host",
						Type:   "string",
						GoType: analyzer.GoType{Kind: analyzer.TypeString, Name: "string"},
						ValidationRules: []analyzer.ValidationRule{
							{Name: "required", Parameter: ""},
							{Name: "hostname", Parameter: ""},
						},
					},
					{
						Name:   "Port",
						Type:   "int",
						GoType: analyzer.GoType{Kind: analyzer.TypeInt, Name: "int"},
						ValidationRules: []analyzer.ValidationRule{
							{Name: "required", Parameter: ""},
							{Name: "min", Parameter: "1"},
							{Name: "max", Parameter: "65535"},
						},
					},
				},
			},
			"DatabaseConfig": {
				Name: "DatabaseConfig",
				Fields: []analyzer.FieldInfo{
					{
						Name:   "URL",
						Type:   "string",
						GoType: analyzer.GoType{Kind: analyzer.TypeString, Name: "string"},
						ValidationRules: []analyzer.ValidationRule{
							{Name: "required", Parameter: ""},
							{Name: "url", Parameter: ""},
						},
					},
				},
			},
		},
		Dependencies: map[string][]string{
			"AppConfig": {"ServerConfig", "DatabaseConfig"},
		},
		Imports:     []string{"fmt", "github.com/mateothegreat/go-validation"},
		PackageName: "config",
	}

	outputDir := t.TempDir()
	options := GeneratorOptions{
		PackageName:         "config",
		OutputDir:           outputDir,
		GenerateStrategies:  true,
		EnableOptimizations: true,
	}

	generator := NewCodeGenerator(analysisResult, options)
	err := generator.Generate()
	if err != nil {
		t.Fatalf("Complex integration failed: %v", err)
	}

	// Verify all struct validators were generated
	expectedFiles := []string{
		"appconfig_validator_gen.go",
		"serverconfig_validator_gen.go",
		"databaseconfig_validator_gen.go",
		"validation_strategy_gen.go",
	}

	for _, filename := range expectedFiles {
		path := filepath.Join(outputDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not generated", filename)
		}
	}

	// Verify generated code compiles
	err = verifyGeneratedCodeCompiles(t, outputDir)
	if err != nil {
		t.Errorf("Generated complex code does not compile: %v", err)
	}
}

// Error case tests

func TestCodeGenerator_InvalidOutputDir(t *testing.T) {
	analysisResult := createTestAnalysisResult()
	options := GeneratorOptions{
		PackageName: "test",
		OutputDir:   "/invalid/path/that/does/not/exist",
	}

	generator := NewCodeGenerator(analysisResult, options)
	err := generator.Generate()

	if err == nil {
		t.Error("Expected error for invalid output directory")
	}
}

func TestCodeGenerator_EmptyAnalysisResult(t *testing.T) {
	analysisResult := &analyzer.AnalysisResult{
		Structs:     map[string]*analyzer.StructInfo{},
		PackageName: "empty",
	}

	outputDir := t.TempDir()
	options := GeneratorOptions{
		PackageName: "empty",
		OutputDir:   outputDir,
	}

	generator := NewCodeGenerator(analysisResult, options)
	err := generator.Generate()
	// Should not fail, but also shouldn't generate any struct validators
	if err != nil {
		t.Errorf("Unexpected error for empty analysis result: %v", err)
	}

	// Check that only strategy file might be generated (if strategies enabled)
	files, _ := os.ReadDir(outputDir)
	structValidatorFiles := 0

	for _, file := range files {
		if strings.Contains(file.Name(), "_validator_gen.go") &&
			!strings.Contains(file.Name(), "strategy") {
			structValidatorFiles++
		}
	}

	if structValidatorFiles > 0 {
		t.Error("No struct validator files should be generated for empty analysis")
	}
}

// Code quality tests

func TestCodeGenerator_GeneratedCodeFormat(t *testing.T) {
	analysisResult := createTestAnalysisResult()
	outputDir := t.TempDir()

	options := GeneratorOptions{
		PackageName: "format_test",
		OutputDir:   outputDir,
	}

	generator := NewCodeGenerator(analysisResult, options)
	err := generator.Generate()
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	// Read generated file and verify it's properly formatted
	filename := filepath.Join(outputDir, "testconfig_validator_gen.go")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	// Parse and reformat to check if it's already properly formatted
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		t.Fatalf("Generated code is not valid Go: %v", err)
	}

	var formatted strings.Builder
	err = format.Node(&formatted, fset, parsed)
	if err != nil {
		t.Fatalf("Failed to format generated code: %v", err)
	}

	// Compare original and formatted - they should be identical
	if string(content) != formatted.String() {
		t.Error("Generated code is not properly formatted")
	}

	fmt.Println(formatted.String())
	fmt.Println("--------------------------------")
	fmt.Println(string(content))
}
