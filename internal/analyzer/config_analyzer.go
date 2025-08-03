// Package analyzer provides AST-based configuration struct analysis for
// zero-reflection validation code generation.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ConfigAnalyzer analyzes Go configuration structs and extracts validation metadata
type ConfigAnalyzer struct {
	fileSet      *token.FileSet
	packageName  string
	parsedFiles  map[string]*ast.File
	structs      map[string]*StructInfo
	dependencies map[string][]string // struct dependency graph
	yamlPaths    map[string]string   // field to YAML path mapping
}

// StructInfo represents analyzed struct information
type StructInfo struct {
	Name           string
	Package        string
	Fields         []FieldInfo
	Position       token.Pos
	IsConfig       bool
	YAMLPath       string
	Dependencies   []string // nested struct dependencies
	ValidationTags map[string][]ValidationRule
}

// FieldInfo represents analyzed field information
type FieldInfo struct {
	Name            string
	Type            string
	GoType          GoType
	ValidationRules []ValidationRule
	YAMLTag         string
	EnvTag          string
	DefaultValue    string
	Position        token.Pos
	IsOptional      bool
	IsNested        bool
	NestedType      string
	IsSlice         bool
	IsMap           bool
	KeyType         string
	ElementType     string
}

// GoType represents detailed Go type information
type GoType struct {
	Kind        TypeKind
	Name        string
	Package     string
	IsPointer   bool
	IsSlice     bool
	IsMap       bool
	IsInterface bool
	KeyType     *GoType
	ElemType    *GoType
}

// TypeKind represents the fundamental type categories
type TypeKind int

const (
	TypeUnknown TypeKind = iota
	TypeString
	TypeInt
	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt64
	TypeUint
	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint64
	TypeFloat32
	TypeFloat64
	TypeBool
	TypeStruct
	TypeSlice
	TypeMap
	TypeInterface
	TypePointer
)

// ValidationRule represents a single validation constraint
type ValidationRule struct {
	Name          string
	Parameter     string
	IsConditional bool
	DependsOn     []string // for cross-field validation
	ErrorMessage  string
	Priority      int // for optimization ordering
}

// AnalysisResult contains the complete analysis results
type AnalysisResult struct {
	Structs      map[string]*StructInfo
	Dependencies map[string][]string
	YAMLPaths    map[string]string
	Imports      []string
	PackageName  string
}

// NewConfigAnalyzer creates a new configuration analyzer
func NewConfigAnalyzer() *ConfigAnalyzer {
	return &ConfigAnalyzer{
		fileSet:      token.NewFileSet(),
		parsedFiles:  make(map[string]*ast.File),
		structs:      make(map[string]*StructInfo),
		dependencies: make(map[string][]string),
		yamlPaths:    make(map[string]string),
	}
}

// AnalyzeDirectory analyzes all Go files in a directory for config structs
func (ca *ConfigAnalyzer) AnalyzeDirectory(dir string) (*AnalysisResult, error) {
	// Parse all Go files in directory
	if err := ca.parseDirectory(dir); err != nil {
		return nil, fmt.Errorf("failed to parse directory: %w", err)
	}

	// Extract struct information
	if err := ca.extractStructs(); err != nil {
		return nil, fmt.Errorf("failed to extract structs: %w", err)
	}

	// Build dependency graph
	ca.buildDependencyGraph()

	// Generate YAML path mappings
	ca.generateYAMLPaths()

	// Optimize validation rules
	ca.optimizeValidationRules()

	return &AnalysisResult{
		Structs:      ca.structs,
		Dependencies: ca.dependencies,
		YAMLPaths:    ca.yamlPaths,
		Imports:      ca.extractRequiredImports(),
		PackageName:  ca.packageName,
	}, nil
}

// AnalyzeFile analyzes a specific Go file for config structs
func (ca *ConfigAnalyzer) AnalyzeFile(filename string) (*AnalysisResult, error) {
	// Parse the specific file
	file, err := parser.ParseFile(ca.fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	ca.parsedFiles[filename] = file
	ca.packageName = file.Name.Name

	// Extract struct information from the file
	if err := ca.extractStructsFromFile(file); err != nil {
		return nil, fmt.Errorf("failed to extract structs from file: %w", err)
	}

	// Build dependency graph
	ca.buildDependencyGraph()

	// Generate YAML path mappings
	ca.generateYAMLPaths()

	// Optimize validation rules
	ca.optimizeValidationRules()

	return &AnalysisResult{
		Structs:      ca.structs,
		Dependencies: ca.dependencies,
		YAMLPaths:    ca.yamlPaths,
		Imports:      ca.extractRequiredImports(),
		PackageName:  ca.packageName,
	}, nil
}

// parseDirectory parses all Go files in the specified directory
func (ca *ConfigAnalyzer) parseDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(ca.fileSet, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse file %s: %w", path, err)
		}

		ca.parsedFiles[path] = file
		if ca.packageName == "" {
			ca.packageName = file.Name.Name
		}

		return nil
	})
}

// extractStructs extracts struct information from all parsed files
func (ca *ConfigAnalyzer) extractStructs() error {
	for _, file := range ca.parsedFiles {
		if err := ca.extractStructsFromFile(file); err != nil {
			return err
		}
	}
	return nil
}

// extractStructsFromFile extracts struct information from a single file
func (ca *ConfigAnalyzer) extractStructsFromFile(file *ast.File) error {
	ast.Inspect(file, func(node ast.Node) bool {
		if typeSpec, ok := node.(*ast.TypeSpec); ok {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				structInfo := ca.analyzeStruct(typeSpec.Name.Name, structType)
				if structInfo != nil {
					ca.structs[structInfo.Name] = structInfo
				}
			}
		}
		return true
	})
	return nil
}

// analyzeStruct analyzes a single struct and extracts validation information
func (ca *ConfigAnalyzer) analyzeStruct(name string, structType *ast.StructType) *StructInfo {
	structInfo := &StructInfo{
		Name:           name,
		Package:        ca.packageName,
		Position:       structType.Pos(),
		ValidationTags: make(map[string][]ValidationRule),
	}

	// Check if this is a config struct (has yaml tags or validation tags)
	hasConfigTags := false

	for _, field := range structType.Fields.List {
		fieldInfo := ca.analyzeField(field)
		if fieldInfo != nil {
			structInfo.Fields = append(structInfo.Fields, *fieldInfo)
			if len(fieldInfo.ValidationRules) > 0 || fieldInfo.YAMLTag != "" {
				hasConfigTags = true
			}
		}
	}

	if !hasConfigTags {
		return nil // Not a config struct
	}

	structInfo.IsConfig = true
	return structInfo
}

// analyzeField analyzes a single struct field
func (ca *ConfigAnalyzer) analyzeField(field *ast.Field) *FieldInfo {
	if len(field.Names) == 0 {
		return nil // Anonymous field, skip for now
	}

	fieldName := field.Names[0].Name
	fieldInfo := &FieldInfo{
		Name:     fieldName,
		Position: field.Pos(),
		GoType:   ca.analyzeGoType(field.Type),
	}

	// Set type string for readability
	fieldInfo.Type = ca.goTypeToString(fieldInfo.GoType)

	// Extract struct tags
	if field.Tag != nil {
		ca.extractFieldTags(field.Tag.Value, fieldInfo)
	}

	// Determine if field is nested config
	if fieldInfo.GoType.Kind == TypeStruct && !ca.isBuiltinType(fieldInfo.GoType.Name) {
		fieldInfo.IsNested = true
		fieldInfo.NestedType = fieldInfo.GoType.Name
	}

	return fieldInfo
}

// analyzeGoType analyzes a Go type expression and returns detailed type information
func (ca *ConfigAnalyzer) analyzeGoType(expr ast.Expr) GoType {
	switch t := expr.(type) {
	case *ast.Ident:
		return GoType{
			Kind:    ca.identToTypeKind(t.Name),
			Name:    t.Name,
			Package: ca.packageName,
		}

	case *ast.StarExpr:
		innerType := ca.analyzeGoType(t.X)
		return GoType{
			Kind:      TypePointer,
			Name:      "*" + innerType.Name,
			Package:   innerType.Package,
			IsPointer: true,
			ElemType:  &innerType,
		}

	case *ast.ArrayType:
		elemType := ca.analyzeGoType(t.Elt)
		return GoType{
			Kind:     TypeSlice,
			Name:     "[]" + elemType.Name,
			Package:  elemType.Package,
			IsSlice:  true,
			ElemType: &elemType,
		}

	case *ast.MapType:
		keyType := ca.analyzeGoType(t.Key)
		valueType := ca.analyzeGoType(t.Value)
		return GoType{
			Kind:     TypeMap,
			Name:     "map[" + keyType.Name + "]" + valueType.Name,
			Package:  valueType.Package,
			IsMap:    true,
			KeyType:  &keyType,
			ElemType: &valueType,
		}

	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return GoType{
				Kind:    TypeStruct, // Assume external types are structs
				Name:    t.Sel.Name,
				Package: pkg.Name,
			}
		}

	case *ast.InterfaceType:
		return GoType{
			Kind:        TypeInterface,
			Name:        "interface{}",
			IsInterface: true,
		}
	}

	return GoType{
		Kind: TypeUnknown,
		Name: "unknown",
	}
}

// identToTypeKind maps identifier names to type kinds
func (ca *ConfigAnalyzer) identToTypeKind(name string) TypeKind {
	switch name {
	case "string":
		return TypeString
	case "int":
		return TypeInt
	case "int8":
		return TypeInt8
	case "int16":
		return TypeInt16
	case "int32":
		return TypeInt32
	case "int64":
		return TypeInt64
	case "uint":
		return TypeUint
	case "uint8":
		return TypeUint8
	case "uint16":
		return TypeUint16
	case "uint32":
		return TypeUint32
	case "uint64":
		return TypeUint64
	case "float32":
		return TypeFloat32
	case "float64":
		return TypeFloat64
	case "bool":
		return TypeBool
	default:
		return TypeStruct
	}
}

// goTypeToString converts a GoType back to a readable string
func (ca *ConfigAnalyzer) goTypeToString(goType GoType) string {
	if goType.IsPointer {
		return "*" + goType.ElemType.Name
	}
	if goType.IsSlice {
		return "[]" + goType.ElemType.Name
	}
	if goType.IsMap {
		return "map[" + goType.KeyType.Name + "]" + goType.ElemType.Name
	}
	if goType.Package != "" && goType.Package != ca.packageName {
		return goType.Package + "." + goType.Name
	}
	return goType.Name
}

// extractFieldTags extracts validation and configuration tags from a field
func (ca *ConfigAnalyzer) extractFieldTags(tagValue string, fieldInfo *FieldInfo) {
	// Remove backticks
	tagValue = strings.Trim(tagValue, "`")

	// Parse individual tags
	tags := ca.parseStructTags(tagValue)

	// Extract validation rules
	if validateTag, exists := tags["validate"]; exists {
		fieldInfo.ValidationRules = ca.parseValidationRules(validateTag)
	}

	// Extract YAML tag
	if yamlTag, exists := tags["yaml"]; exists {
		fieldInfo.YAMLTag = ca.parseYAMLTag(yamlTag)
	}

	// Extract environment variable tag
	if envTag, exists := tags["env"]; exists {
		fieldInfo.EnvTag = envTag
	}

	// Extract default value
	if defaultTag, exists := tags["default"]; exists {
		fieldInfo.DefaultValue = defaultTag
	}

	// Check if field is optional
	fieldInfo.IsOptional = ca.isFieldOptional(fieldInfo.ValidationRules)
}

// parseStructTags parses struct tag string into key-value pairs
func (ca *ConfigAnalyzer) parseStructTags(tagStr string) map[string]string {
	tags := make(map[string]string)

	// Simple tag parsing - this could be enhanced for more complex cases
	parts := strings.Fields(tagStr)
	for _, part := range parts {
		if colonIdx := strings.Index(part, ":"); colonIdx != -1 {
			key := part[:colonIdx]
			value := strings.Trim(part[colonIdx+1:], `"`)
			tags[key] = value
		}
	}

	return tags
}

// parseValidationRules parses validation tag value into individual rules
func (ca *ConfigAnalyzer) parseValidationRules(validateTag string) []ValidationRule {
	var rules []ValidationRule

	// Split by comma and parse each rule
	ruleParts := strings.Split(validateTag, ",")
	for i, rulePart := range ruleParts {
		rulePart = strings.TrimSpace(rulePart)
		if rulePart == "" || rulePart == "-" {
			continue
		}

		rule := ValidationRule{
			Priority: i, // Maintain order for optimization
		}

		// Parse rule name and parameter
		if equalIdx := strings.Index(rulePart, "="); equalIdx != -1 {
			rule.Name = rulePart[:equalIdx]
			rule.Parameter = rulePart[equalIdx+1:]
		} else {
			rule.Name = rulePart
		}

		// Determine if rule is conditional
		rule.IsConditional = ca.isConditionalRule(rule.Name)

		// Extract dependencies for cross-field validation
		if ca.isCrossFieldRule(rule.Name) {
			rule.DependsOn = ca.extractCrossFieldDependencies(rule)
		}

		rules = append(rules, rule)
	}

	return rules
}

// parseYAMLTag extracts the YAML field name from yaml tag
func (ca *ConfigAnalyzer) parseYAMLTag(yamlTag string) string {
	// Handle yaml:"field_name,omitempty" format
	if commaIdx := strings.Index(yamlTag, ","); commaIdx != -1 {
		return yamlTag[:commaIdx]
	}
	return yamlTag
}

// isFieldOptional determines if a field is optional based on validation rules
func (ca *ConfigAnalyzer) isFieldOptional(rules []ValidationRule) bool {
	for _, rule := range rules {
		if rule.Name == "required" {
			return false
		}
		if rule.Name == "omitempty" {
			return true
		}
	}
	return true // Default to optional if no required rule
}

// isConditionalRule determines if a validation rule is conditional
func (ca *ConfigAnalyzer) isConditionalRule(ruleName string) bool {
	conditionalRules := map[string]bool{
		"required_if":      true,
		"required_unless":  true,
		"required_with":    true,
		"required_without": true,
	}
	return conditionalRules[ruleName]
}

// isCrossFieldRule determines if a validation rule involves cross-field validation
func (ca *ConfigAnalyzer) isCrossFieldRule(ruleName string) bool {
	crossFieldRules := map[string]bool{
		"eqfield":          true,
		"nefield":          true,
		"gtfield":          true,
		"gtefiled":         true,
		"ltfield":          true,
		"ltefield":         true,
		"required_if":      true,
		"required_unless":  true,
		"required_with":    true,
		"required_without": true,
	}
	return crossFieldRules[ruleName]
}

// extractCrossFieldDependencies extracts field dependencies from cross-field rules
func (ca *ConfigAnalyzer) extractCrossFieldDependencies(rule ValidationRule) []string {
	switch rule.Name {
	case "eqfield", "nefield", "gtfield", "gtefiled", "ltfield", "ltefield":
		return []string{rule.Parameter}
	case "required_if", "required_unless":
		// Format: "required_if=FieldName value"
		parts := strings.Fields(rule.Parameter)
		if len(parts) >= 1 {
			return []string{parts[0]}
		}
	case "required_with", "required_without":
		return []string{rule.Parameter}
	}
	return nil
}

// buildDependencyGraph builds the struct dependency graph
func (ca *ConfigAnalyzer) buildDependencyGraph() {
	for structName, structInfo := range ca.structs {
		var dependencies []string

		for _, field := range structInfo.Fields {
			if field.IsNested && field.NestedType != structName {
				dependencies = append(dependencies, field.NestedType)
			}
		}

		ca.dependencies[structName] = dependencies
		structInfo.Dependencies = dependencies
	}
}

// generateYAMLPaths generates YAML path mappings for configuration fields
func (ca *ConfigAnalyzer) generateYAMLPaths() {
	for _, structInfo := range ca.structs {
		ca.generateStructYAMLPaths(structInfo, "")
	}
}

// generateStructYAMLPaths generates YAML paths for a struct recursively
func (ca *ConfigAnalyzer) generateStructYAMLPaths(structInfo *StructInfo, prefix string) {
	for _, field := range structInfo.Fields {
		yamlName := field.YAMLTag
		if yamlName == "" {
			yamlName = strings.ToLower(field.Name)
		}

		var fullPath string
		if prefix == "" {
			fullPath = yamlName
		} else {
			fullPath = prefix + "." + yamlName
		}

		fieldKey := structInfo.Name + "." + field.Name
		ca.yamlPaths[fieldKey] = fullPath

		// Recurse into nested structs
		if field.IsNested {
			if nestedStruct, exists := ca.structs[field.NestedType]; exists {
				ca.generateStructYAMLPaths(nestedStruct, fullPath)
			}
		}
	}
}

// optimizeValidationRules optimizes validation rules for performance
func (ca *ConfigAnalyzer) optimizeValidationRules() {
	for _, structInfo := range ca.structs {
		for i := range structInfo.Fields {
			ca.optimizeFieldValidationRules(&structInfo.Fields[i])
		}
	}
}

// optimizeFieldValidationRules optimizes validation rules for a single field
func (ca *ConfigAnalyzer) optimizeFieldValidationRules(field *FieldInfo) {
	if len(field.ValidationRules) <= 1 {
		return
	}

	// Sort rules by priority (likely to fail first)
	ca.sortRulesByPriority(field.ValidationRules)

	// Merge compatible rules where possible
	field.ValidationRules = ca.mergeCompatibleRules(field.ValidationRules)
}

// sortRulesByPriority sorts validation rules by execution priority
func (ca *ConfigAnalyzer) sortRulesByPriority(rules []ValidationRule) {
	// Priority order: required -> type checks -> format checks -> range checks
	priority := map[string]int{
		"required":  1,
		"omitempty": 2,
		"alpha":     3,
		"alphanum":  3,
		"numeric":   3,
		"email":     4,
		"url":       4,
		"min":       5,
		"max":       5,
		"len":       5,
	}

	for i := range rules {
		if p, exists := priority[rules[i].Name]; exists {
			rules[i].Priority = p
		} else {
			rules[i].Priority = 10 // Default priority
		}
	}
}

// mergeCompatibleRules merges validation rules that can be combined
func (ca *ConfigAnalyzer) mergeCompatibleRules(rules []ValidationRule) []ValidationRule {
	// For now, return as-is. Could implement rule fusion optimizations here
	return rules
}

// extractRequiredImports determines what imports are needed for generated code
func (ca *ConfigAnalyzer) extractRequiredImports() []string {
	imports := []string{
		"fmt",
		"github.com/mateothegreat/go-validation",
	}

	// Check if we need additional imports based on validation rules
	for _, structInfo := range ca.structs {
		for _, field := range structInfo.Fields {
			for _, rule := range field.ValidationRules {
				switch rule.Name {
				case "uuid", "uuid4":
					imports = ca.addImportIfMissing(imports, "github.com/google/uuid")
				case "datetime", "date", "time":
					imports = ca.addImportIfMissing(imports, "time")
				case "json":
					imports = ca.addImportIfMissing(imports, "encoding/json")
				case "base64":
					imports = ca.addImportIfMissing(imports, "encoding/base64")
				case "ip", "ipv4", "ipv6", "cidr", "mac":
					imports = ca.addImportIfMissing(imports, "net")
				case "url", "uri":
					imports = ca.addImportIfMissing(imports, "net/url")
				}
			}
		}
	}

	return imports
}

// addImportIfMissing adds an import if it's not already in the list
func (ca *ConfigAnalyzer) addImportIfMissing(imports []string, newImport string) []string {
	for _, imp := range imports {
		if imp == newImport {
			return imports
		}
	}
	return append(imports, newImport)
}

// isBuiltinType checks if a type is a Go builtin type
func (ca *ConfigAnalyzer) isBuiltinType(typeName string) bool {
	builtins := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "byte": true, "rune": true,
		"error": true, "interface{}": true,
	}
	return builtins[typeName]
}

// GetStructInfo returns information about a specific struct
func (ca *ConfigAnalyzer) GetStructInfo(name string) (*StructInfo, bool) {
	info, exists := ca.structs[name]
	return info, exists
}

// GetAllStructs returns all analyzed struct information
func (ca *ConfigAnalyzer) GetAllStructs() map[string]*StructInfo {
	return ca.structs
}

// String returns a string representation of the analysis result
func (ar *AnalysisResult) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Package: %s\n", ar.PackageName))
	sb.WriteString(fmt.Sprintf("Structs: %d\n", len(ar.Structs)))
	sb.WriteString(fmt.Sprintf("Imports: %v\n", ar.Imports))

	for name, structInfo := range ar.Structs {
		sb.WriteString(fmt.Sprintf("  %s: %d fields\n", name, len(structInfo.Fields)))
	}

	return sb.String()
}
