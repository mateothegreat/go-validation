package benchmarks

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BenchmarkTableBuilder provides a fluent interface for building benchmark tables
type BenchmarkTableBuilder struct {
	suite *BenchmarkSuite
}

// NewBenchmarkTable creates a new benchmark table builder
func NewBenchmarkTable(suiteName string) *BenchmarkTableBuilder {
	return &BenchmarkTableBuilder{
		suite: NewBenchmarkSuite(suiteName),
	}
}

// WithCase adds a benchmark case using fluent interface
func (btb *BenchmarkTableBuilder) WithCase(name string, fn TestableFunction, args ...interface{}) *BenchmarkTableBuilder {
	btb.suite.AddCase(BenchmarkCase{
		Name:     name,
		Function: fn,
		Args:     args,
	})
	return btb
}

// WithCaseExpectingError adds a benchmark case that expects an error
func (btb *BenchmarkTableBuilder) WithCaseExpectingError(name string, fn TestableFunction, args ...interface{}) *BenchmarkTableBuilder {
	btb.suite.AddCase(BenchmarkCase{
		Name:        name,
		Function:    fn,
		Args:        args,
		ExpectError: true,
	})
	return btb
}

// WithScaling adds input size scaling dimensions
func (btb *BenchmarkTableBuilder) WithScaling(name string, values ...interface{}) *BenchmarkTableBuilder {
	btb.suite.AddScalingDimension(ScalingDimension{
		Name:   name,
		Values: values,
	})
	return btb
}

// WithInputSizeScaling adds common input size scaling (10, 100, 1K, 10K, 100K)
func (btb *BenchmarkTableBuilder) WithInputSizeScaling() *BenchmarkTableBuilder {
	return btb.WithScaling("InputSize", 10, 100, 1000, 10000, 100000)
}

// WithConcurrency sets concurrency levels to test
func (btb *BenchmarkTableBuilder) WithConcurrency(levels ...int) *BenchmarkTableBuilder {
	btb.suite.Concurrency = levels
	return btb
}

// WithThreshold sets performance thresholds for regression detection
func (btb *BenchmarkTableBuilder) WithThreshold(caseName string, maxNsPerOp float64, maxAllocs int) *BenchmarkTableBuilder {
	btb.suite.SetThreshold(caseName, PerformanceThreshold{
		MaxNsPerOp:       maxNsPerOp,
		MaxAllocsPerOp:   maxAllocs,
		TolerancePercent: 10.0, // Default 10% tolerance
	})
	return btb
}

// WithBaseline loads baseline performance data from file
func (btb *BenchmarkTableBuilder) WithBaseline(filePath string) *BenchmarkTableBuilder {
	if baselines, err := LoadBaselines(filePath); err == nil {
		for name, baseline := range baselines {
			btb.suite.SetBaseline(name, baseline)
		}
	}
	return btb
}

// Build returns the configured benchmark suite
func (btb *BenchmarkTableBuilder) Build() *BenchmarkSuite {
	return btb.suite
}

// DataGenerator provides utilities for generating test data at scale
type DataGenerator struct{}

// GenerateInts creates integer slices of specified sizes
func (dg *DataGenerator) GenerateInts(size int, min, max int64) []int64 {
	data := make([]int64, size)
	step := (max - min) / int64(size)
	if step == 0 {
		step = 1
	}
	
	for i := 0; i < size; i++ {
		data[i] = min + int64(i)*step
	}
	return data
}

// GenerateStrings creates string slices of specified sizes and lengths
func (dg *DataGenerator) GenerateStrings(count, length int) []string {
	data := make([]string, count)
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	
	for i := 0; i < count; i++ {
		str := make([]byte, length)
		for j := range str {
			str[j] = chars[(i*length+j)%len(chars)]
		}
		data[i] = string(str)
	}
	return data
}

// GenerateFloats creates float64 slices of specified sizes
func (dg *DataGenerator) GenerateFloats(size int, min, max float64) []float64 {
	data := make([]float64, size)
	step := (max - min) / float64(size)
	
	for i := 0; i < size; i++ {
		data[i] = min + float64(i)*step
	}
	return data
}

// ResultAnalyzer provides statistical analysis of benchmark results
type ResultAnalyzer struct {
	results []BenchmarkResult
}

// NewResultAnalyzer creates a new result analyzer
func NewResultAnalyzer(results []BenchmarkResult) *ResultAnalyzer {
	return &ResultAnalyzer{results: results}
}

// AnalyzeScaling examines performance scaling across input sizes
func (ra *ResultAnalyzer) AnalyzeScaling() ScalingAnalysis {
	// Group results by base name (without size suffix)
	groups := make(map[string][]BenchmarkResult)
	
	for _, result := range ra.results {
		baseName := ra.extractBaseName(result.Name)
		groups[baseName] = append(groups[baseName], result)
	}
	
	analysis := ScalingAnalysis{
		Functions: make(map[string]FunctionScaling),
	}
	
	for funcName, results := range groups {
		if len(results) < 2 {
			continue // Need at least 2 data points for scaling analysis
		}
		
		// Sort by input size
		sort.Slice(results, func(i, j int) bool {
			return results[i].InputSize < results[j].InputSize
		})
		
		scaling := ra.calculateScaling(results)
		analysis.Functions[funcName] = scaling
	}
	
	return analysis
}

// calculateScaling determines if performance scales linearly, logarithmically, etc.
func (ra *ResultAnalyzer) calculateScaling(results []BenchmarkResult) FunctionScaling {
	n := len(results)
	if n < 2 {
		return FunctionScaling{Complexity: "unknown"}
	}
	
	// Calculate ratios between consecutive measurements
	var ratios []float64
	for i := 1; i < n; i++ {
		sizeRatio := float64(results[i].InputSize) / float64(results[i-1].InputSize)
		timeRatio := results[i].NsPerOp / results[i-1].NsPerOp
		ratios = append(ratios, timeRatio/sizeRatio)
	}
	
	// Analyze ratio patterns to determine complexity
	avgRatio := ra.calculateMean(ratios)
	variance := ra.calculateVariance(ratios, avgRatio)
	
	var complexity string
	if variance < 0.1 { // Low variance
		if avgRatio < 1.2 {
			complexity = "O(1)" // Constant time
		} else if avgRatio < 2.0 {
			complexity = "O(log n)" // Logarithmic
		} else {
			complexity = "O(n)" // Linear
		}
	} else {
		complexity = "O(n²) or worse" // High variance suggests worse complexity
	}
	
	return FunctionScaling{
		Complexity:    complexity,
		AvgRatio:      avgRatio,
		Variance:      variance,
		DataPoints:    n,
		MinInputSize:  results[0].InputSize,
		MaxInputSize:  results[n-1].InputSize,
		MinNsPerOp:    results[0].NsPerOp,
		MaxNsPerOp:    results[n-1].NsPerOp,
	}
}

// extractBaseName removes size and concurrency suffixes from benchmark names
func (ra *ResultAnalyzer) extractBaseName(name string) string {
	// Remove common suffixes like "_Size1000_Conc1"
	if idx := strings.Index(name, "_Size"); idx != -1 {
		return name[:idx]
	}
	if idx := strings.Index(name, "_Conc"); idx != -1 {
		return name[:idx]
	}
	return name
}

// calculateMean calculates the arithmetic mean of a slice
func (ra *ResultAnalyzer) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateVariance calculates the variance of a slice
func (ra *ResultAnalyzer) calculateVariance(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return sumSquares / float64(len(values)-1)
}

// ScalingAnalysis contains the results of scaling analysis
type ScalingAnalysis struct {
	Functions map[string]FunctionScaling `json:"functions"`
	Timestamp time.Time                  `json:"timestamp"`
}

// FunctionScaling describes the scaling characteristics of a function
type FunctionScaling struct {
	Complexity    string  `json:"complexity"`
	AvgRatio      float64 `json:"avg_ratio"`
	Variance      float64 `json:"variance"`
	DataPoints    int     `json:"data_points"`
	MinInputSize  int     `json:"min_input_size"`
	MaxInputSize  int     `json:"max_input_size"`
	MinNsPerOp    float64 `json:"min_ns_per_op"`
	MaxNsPerOp    float64 `json:"max_ns_per_op"`
}

// ReportGenerator creates comprehensive reports from benchmark results
type ReportGenerator struct {
	results []BenchmarkResult
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(results []BenchmarkResult) *ReportGenerator {
	return &ReportGenerator{results: results}
}

// GenerateMarkdownReport creates a markdown report suitable for documentation
func (rg *ReportGenerator) GenerateMarkdownReport() string {
	var report strings.Builder
	
	report.WriteString("# Benchmark Report\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))
	
	// Summary statistics
	report.WriteString("## Summary\n\n")
	report.WriteString(fmt.Sprintf("- Total benchmarks: %d\n", len(rg.results)))
	
	regressions := 0
	for _, result := range rg.results {
		if result.RegressionFlag {
			regressions++
		}
	}
	report.WriteString(fmt.Sprintf("- Performance regressions: %d\n\n", regressions))
	
	// Performance table
	report.WriteString("## Performance Results\n\n")
	report.WriteString("| Benchmark | Input Size | ns/op | allocs/op | Status |\n")
	report.WriteString("|-----------|------------|-------|-----------|--------|\n")
	
	for _, result := range rg.results {
		status := "✅ Pass"
		if result.RegressionFlag {
			status = "❌ Regression"
		}
		
		report.WriteString(fmt.Sprintf("| %s | %d | %.2f | %d | %s |\n",
			result.Name, result.InputSize, result.NsPerOp, result.AllocsPerOp, status))
	}
	
	// Scaling analysis
	analyzer := NewResultAnalyzer(rg.results)
	scaling := analyzer.AnalyzeScaling()
	
	if len(scaling.Functions) > 0 {
		report.WriteString("\n## Scaling Analysis\n\n")
		for funcName, analysis := range scaling.Functions {
			report.WriteString(fmt.Sprintf("### %s\n", funcName))
			report.WriteString(fmt.Sprintf("- **Complexity**: %s\n", analysis.Complexity))
			report.WriteString(fmt.Sprintf("- **Input size range**: %d - %d\n", analysis.MinInputSize, analysis.MaxInputSize))
			report.WriteString(fmt.Sprintf("- **Performance range**: %.2f - %.2f ns/op\n\n", analysis.MinNsPerOp, analysis.MaxNsPerOp))
		}
	}
	
	return report.String()
}

// SaveResults saves benchmark results to a JSON file for later analysis
func SaveResults(results []BenchmarkResult, filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	// Write results to file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// LoadResults loads benchmark results from a JSON file
func LoadResults(filePath string) ([]BenchmarkResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()
	
	var results []BenchmarkResult
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&results)
	return results, err
}

// LoadBaselines loads baseline performance data for regression testing
func LoadBaselines(filePath string) (map[string]BenchmarkResult, error) {
	results, err := LoadResults(filePath)
	if err != nil {
		return nil, err
	}
	
	baselines := make(map[string]BenchmarkResult)
	for _, result := range results {
		baselines[result.Name] = result
	}
	return baselines, nil
}

// CompareResults compares two sets of benchmark results for regression analysis
func CompareResults(baseline, current []BenchmarkResult, tolerance float64) []RegressionResult {
	baselineMap := make(map[string]BenchmarkResult)
	for _, result := range baseline {
		baselineMap[result.Name] = result
	}
	
	var regressions []RegressionResult
	
	for _, currentResult := range current {
		if baselineResult, exists := baselineMap[currentResult.Name]; exists {
			regression := analyzeRegression(baselineResult, currentResult, tolerance)
			if regression.IsRegression {
				regressions = append(regressions, regression)
			}
		}
	}
	
	return regressions
}

// analyzeRegression compares two benchmark results for performance regression
func analyzeRegression(baseline, current BenchmarkResult, tolerance float64) RegressionResult {
	timeDiff := current.NsPerOp - baseline.NsPerOp
	timePercent := (timeDiff / baseline.NsPerOp) * 100
	
	allocDiff := current.AllocsPerOp - baseline.AllocsPerOp
	allocPercent := 0.0
	if baseline.AllocsPerOp > 0 {
		allocPercent = (float64(allocDiff) / float64(baseline.AllocsPerOp)) * 100
	}
	
	isRegression := math.Abs(timePercent) > tolerance || math.Abs(allocPercent) > tolerance
	
	return RegressionResult{
		Name:           current.Name,
		IsRegression:   isRegression,
		BaselineNs:     baseline.NsPerOp,
		CurrentNs:      current.NsPerOp,
		TimeDiffNs:     timeDiff,
		TimeDiffPercent: timePercent,
		BaselineAllocs: baseline.AllocsPerOp,
		CurrentAllocs:  current.AllocsPerOp,
		AllocDiff:      allocDiff,
		AllocDiffPercent: allocPercent,
	}
}

// RegressionResult contains the analysis of a potential performance regression
type RegressionResult struct {
	Name             string  `json:"name"`
	IsRegression     bool    `json:"is_regression"`
	BaselineNs       float64 `json:"baseline_ns"`
	CurrentNs        float64 `json:"current_ns"`
	TimeDiffNs       float64 `json:"time_diff_ns"`
	TimeDiffPercent  float64 `json:"time_diff_percent"`
	BaselineAllocs   int     `json:"baseline_allocs"`
	CurrentAllocs    int     `json:"current_allocs"`
	AllocDiff        int     `json:"alloc_diff"`
	AllocDiffPercent float64 `json:"alloc_diff_percent"`
}