package validation

import (
	"fmt"
)

// ExampleConfig demonstrates the new size validation features with byte units
type ExampleConfig struct {
	// Basic validation (backward compatible)
	Username string `json:"username" validate:"required,minlen=3,maxlen=20"`
	
	// NEW: Byte unit validation for file uploads
	SmallFile   string `json:"small_file" validate:"required,maxlen=10KB"`    // Max 10 kilobytes
	MediumFile  string `json:"medium_file" validate:"required,maxlen=5MB"`   // Max 5 megabytes  
	LargeFile   string `json:"large_file" validate:"required,maxlen=1GB"`    // Max 1 gigabyte
	
	// NEW: Mixed byte unit validations
	Avatar      string `json:"avatar" validate:"required,maxlen=500KB"`      // Profile image limit
	Document    string `json:"document" validate:"required,maxlen=25MB"`     // Document upload limit
	VideoFile   string `json:"video" validate:"maxlen=2GB"`                  // Video file limit
	
	// Legacy character-specific validation (still supported)
	DisplayName string `json:"display_name" validate:"required,maxlen=50:chars"`
	Description string `json:"description" validate:"maxlen=200:runes"`
	
	// Integer value validation (unchanged)
	Port      int     `json:"port" validate:"required,min=1,max=65535"`
	MaxSize   int     `json:"max_size" validate:"min=1:value,max=1000000:value"`
	
	// Float validation (unchanged)
	Percentage float64 `json:"percentage" validate:"min=0,max=100"`
}

// DemonstrateByteUnitValidation shows how the new byte unit validation system works
func DemonstrateByteUnitValidation() {
	validator := NewUnifiedValidator(ValidatorConfig{
		Strategy: StrategyFast,
	})

	// Example test cases with byte unit validation
	testCases := []struct {
		name   string
		config ExampleConfig
	}{
		{
			name: "Valid Config - All sizes within limits",
			config: ExampleConfig{
				Username:    "john_doe",
				SmallFile:   "Small content",                  // ~13 bytes < 10KB
				MediumFile:  "Medium file content here",       // ~26 bytes < 5MB
				LargeFile:   "Large file placeholder",         // ~24 bytes < 1GB
				Avatar:      "Avatar image data",              // ~17 bytes < 500KB
				Document:    "Document content",               // ~16 bytes < 25MB
				VideoFile:   "Video placeholder",              // ~17 bytes < 2GB
				DisplayName: "John Doe",                       // 8 chars < 50
				Description: "A brief description",            // 19 runes < 200
				Port:        8080,                             // Valid port
				MaxSize:     50000,                            // Valid size
				Percentage:  85.5,                             // Valid percentage
			},
		},
		{
			name: "Invalid - Small file exceeds 10KB",
			config: ExampleConfig{
				Username:  "john_doe",
				SmallFile: string(make([]byte, 11*1024)),      // 11KB > 10KB limit
				MediumFile:  "Valid medium content",
				LargeFile:   "Valid large content", 
				Avatar:      "Valid avatar",
				Document:    "Valid document",
				VideoFile:   "Valid video",
				DisplayName: "John Doe",
				Description: "A brief description",
				Port:        8080,
				MaxSize:     50000,
				Percentage:  85.5,
			},
		},
		{
			name: "Invalid - Medium file exceeds 5MB",
			config: ExampleConfig{
				Username:    "john_doe",
				SmallFile:   "Valid small content",
				MediumFile:  string(make([]byte, 6*1024*1024)), // 6MB > 5MB limit
				LargeFile:   "Valid large content",
				Avatar:      "Valid avatar", 
				Document:    "Valid document",
				VideoFile:   "Valid video",
				DisplayName: "John Doe",
				Description: "A brief description",
				Port:        8080,
				MaxSize:     50000,
				Percentage:  85.5,
			},
		},
		{
			name: "Invalid - Avatar exceeds 500KB",
			config: ExampleConfig{
				Username:    "john_doe",
				SmallFile:   "Valid small content",
				MediumFile:  "Valid medium content",
				LargeFile:   "Valid large content",
				Avatar:      string(make([]byte, 600*1024)),    // 600KB > 500KB limit
				Document:    "Valid document",
				VideoFile:   "Valid video",
				DisplayName: "John Doe",
				Description: "A brief description",
				Port:        8080,
				MaxSize:     50000,
				Percentage:  85.5,
			},
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\\n=== Testing: %s ===\\n", tc.name)
		
		err := validator.Validate(tc.config)
		if err != nil {
			fmt.Printf("Validation failed: %v\\n", err)
		} else {
			fmt.Printf("Validation passed!\\n")
		}
	}
}

// ExampleByteUnitSpecUsage demonstrates the new byte unit specifications
func ExampleByteUnitSpecUsage() {
	fmt.Println("\\n=== Byte Unit Specification Examples ===")
	
	examples := []struct {
		rule        string
		value       any
		comparison  string
		description string
	}{
		// NEW: Byte unit specifications
		{"10KB", "Small file content", "max", "Kilobyte limit validation"},
		{"5MB", string(make([]byte, 3*1024*1024)), "max", "Megabyte limit validation (3MB < 5MB)"},
		{"1GB", string(make([]byte, 500*1024*1024)), "max", "Gigabyte limit validation (500MB < 1GB)"},
		{"500KB", string(make([]byte, 256*1024)), "max", "Half-megabyte limit (256KB < 500KB)"},
		{"100B", "This string is longer than 100 bytes and should fail the validation test", "max", "Byte limit validation (should fail)"},
		
		// Legacy specifications (still supported)
		{"20:chars", "héllo wørld", "max", "UTF-8 character count validation"},
		{"100", 50, "max", "Integer value validation"},
		{"10:runes", "🚀🌟💯", "max", "Unicode emoji validation"},
		
		// Mixed examples
		{"2MB", "Short text", "max", "Large limit with small content"},
		{"50:chars", "这是中文测试内容", "max", "Chinese character validation"},
	}
	
	for _, ex := range examples {
		spec, err := ParseSizeSpec(ex.rule)
		if err != nil {
			fmt.Printf("Error parsing '%s': %v\\n", ex.rule, err)
			continue
		}
		
		err = ValidateSize("test_field", ex.value, spec, ex.comparison)
		status := "✅ PASS"
		if err != nil {
			status = "❌ FAIL: " + err.Error()
		}
		
		fmt.Printf("Rule: %-15s %s - %s\\n", 
			ex.rule, status, ex.description)
	}
}

// ExampleByteUnitConversions demonstrates byte unit conversions
func ExampleByteUnitConversions() {
	fmt.Println("\\n=== Byte Unit Conversion Examples ===")
	
	conversions := []string{
		"1KB",    // 1024 bytes
		"2MB",    // 2 * 1024 * 1024 bytes
		"0.5GB",  // 0.5 * 1024 * 1024 * 1024 bytes
		"1.5TB",  // 1.5 * 1024^4 bytes
		"100B",   // 100 bytes
	}
	
	for _, rule := range conversions {
		spec, err := ParseSizeSpec(rule)
		if err != nil {
			fmt.Printf("Error parsing '%s': %v\\n", rule, err)
			continue
		}
		
		fmt.Printf("%-8s = %15d bytes\\n", rule, spec.Value)
	}
}