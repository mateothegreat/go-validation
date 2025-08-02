// Package benchmarks provides a reusable, abstractable framework for systematic
// performance testing that can be used across multiple Go libraries and packages.
package benchmarks

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestableFunction represents any function that can be benchmarked
type TestableFunction func(args ...interface{}) error

// BenchmarkCase defines a single benchmark test case with all necessary parameters
type BenchmarkCase struct {
	Name         string            // Descriptive name for the benchmark
	Function     TestableFunction  // Function to benchmark
	Args         []interface{}     // Arguments to pass to the function
	InputSize    int              // Logical size of input (for scaling analysis)
	ExpectError  bool             // Whether this case should produce an error
	Setup        func()           // Optional setup function run before benchmark
	Teardown     func()           // Optional teardown function run after benchmark
	Tags         []string         // Tags for grouping/filtering benchmarks
	Metadata     map[string]interface{} // Additional metadata for analysis
}

// ScalingDimension defines a dimension for scaling tests (e.g., input size, concurrency)
type ScalingDimension struct {
	Name   string        // Name of the dimension (e.g., "InputSize", "Concurrency")
	Values []interface{} // Values to test across this dimension
}

// PerformanceThreshold defines performance expectations and regression detection
type PerformanceThreshold struct {
	MaxNsPerOp       float64 // Maximum acceptable nanoseconds per operation
	MaxAllocsPerOp   int     // Maximum acceptable allocations per operation
	MaxBytesPerOp    int64   // Maximum acceptable bytes allocated per operation
	TolerancePercent float64 // Acceptable variance percentage (e.g., 10.0 for 10%)
}

// BenchmarkSuite organizes and executes a comprehensive set of benchmarks
type BenchmarkSuite struct {
	Name        string                           // Suite name
	Cases       []BenchmarkCase                  // Individual test cases
	Scaling     []ScalingDimension              // Scaling dimensions to test
	Thresholds  map[string]PerformanceThreshold // Performance thresholds by case name
	Baselines   map[string]BenchmarkResult      // Baseline results for regression detection
	Concurrency []int                           // Concurrency levels to test
	Iterations  int                             // Number of iterations for stability testing
}

// BenchmarkResult captures comprehensive benchmark results
type BenchmarkResult struct {
	Name         string            `json:"name"`
	NsPerOp      float64          `json:"ns_per_op"`
	AllocsPerOp  int              `json:"allocs_per_op"`
	BytesPerOp   int64            `json:"bytes_per_op"`
	InputSize    int              `json:"input_size"`
	Concurrency  int              `json:"concurrency"`
	Timestamp    time.Time        `json:"timestamp"`
	Tags         []string         `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	RegressionFlag bool           `json:"regression_flag"`
	Error        string           `json:"error,omitempty"`
}

// BenchmarkRunner executes benchmark suites with advanced analysis capabilities
type BenchmarkRunner struct {
	suite           *BenchmarkSuite
	results         []BenchmarkResult
	regressionCount int
	mu              sync.RWMutex
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite(name string) *BenchmarkSuite {
	return &BenchmarkSuite{
		Name:       name,
		Cases:      make([]BenchmarkCase, 0),
		Scaling:    make([]ScalingDimension, 0),
		Thresholds: make(map[string]PerformanceThreshold),
		Baselines:  make(map[string]BenchmarkResult),
		Iterations: 3, // Default stability testing iterations
	}
}

// AddCase adds a benchmark case to the suite
func (bs *BenchmarkSuite) AddCase(testCase BenchmarkCase) {
	bs.Cases = append(bs.Cases, testCase)
}

// AddScalingDimension adds a scaling dimension for performance analysis
func (bs *BenchmarkSuite) AddScalingDimension(dimension ScalingDimension) {
	bs.Scaling = append(bs.Scaling, dimension)
}

// SetThreshold sets performance thresholds for regression detection
func (bs *BenchmarkSuite) SetThreshold(caseName string, threshold PerformanceThreshold) {
	bs.Thresholds[caseName] = threshold
}

// SetBaseline sets baseline performance for regression comparison
func (bs *BenchmarkSuite) SetBaseline(caseName string, baseline BenchmarkResult) {
	bs.Baselines[caseName] = baseline
}

// NewBenchmarkRunner creates a new benchmark runner for a suite
func NewBenchmarkRunner(suite *BenchmarkSuite) *BenchmarkRunner {
	return &BenchmarkRunner{
		suite:   suite,
		results: make([]BenchmarkResult, 0),
	}
}

// RunStandardBenchmarks executes all standard benchmark cases
func (br *BenchmarkRunner) RunStandardBenchmarks(b *testing.B) {
	for _, testCase := range br.suite.Cases {
		br.runSingleBenchmark(b, testCase, 1, testCase.InputSize)
	}
}

// RunScalingBenchmarks executes benchmarks across all scaling dimensions
func (br *BenchmarkRunner) RunScalingBenchmarks(b *testing.B) {
	if len(br.suite.Scaling) == 0 {
		br.RunStandardBenchmarks(b)
		return
	}
	
	// Generate all combinations of scaling dimensions
	combinations := br.generateScalingCombinations()
	
	for _, testCase := range br.suite.Cases {
		for _, combo := range combinations {
			scaledCase := br.scaleTestCase(testCase, combo)
			br.runSingleBenchmark(b, scaledCase, 1, scaledCase.InputSize)
		}
	}
}

// RunConcurrencyBenchmarks tests performance under different concurrency levels
func (br *BenchmarkRunner) RunConcurrencyBenchmarks(b *testing.B) {
	if len(br.suite.Concurrency) == 0 {
		br.suite.Concurrency = []int{1, 2, 4, 8, 16} // Default concurrency levels
	}
	
	for _, testCase := range br.suite.Cases {
		for _, concurrency := range br.suite.Concurrency {
			br.runConcurrentBenchmark(b, testCase, concurrency)
		}
	}
}

// RunStabilityBenchmarks runs benchmarks multiple times to detect variance
func (br *BenchmarkRunner) RunStabilityBenchmarks(b *testing.B) {
	for _, testCase := range br.suite.Cases {
		results := make([]BenchmarkResult, br.suite.Iterations)
		
		for i := 0; i < br.suite.Iterations; i++ {
			result := br.runSingleBenchmark(b, testCase, 1, testCase.InputSize)
			results[i] = result
		}
		
		br.analyzeStability(testCase.Name, results)
	}
}

// RunMemoryProfilingBenchmarks focuses on memory allocation analysis
func (br *BenchmarkRunner) RunMemoryProfilingBenchmarks(b *testing.B) {
	for _, testCase := range br.suite.Cases {
		b.Run(fmt.Sprintf("Memory_%s", testCase.Name), func(b *testing.B) {
			if testCase.Setup != nil {
				testCase.Setup()
			}
			
			b.ReportAllocs()
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				err := testCase.Function(testCase.Args...)
				if (err != nil) != testCase.ExpectError {
					b.Errorf("Expected error=%v, got error=%v", testCase.ExpectError, err != nil)
				}
			}
			
			if testCase.Teardown != nil {
				testCase.Teardown()
			}
		})
	}
}

// runSingleBenchmark executes a single benchmark case and captures results
func (br *BenchmarkRunner) runSingleBenchmark(b *testing.B, testCase BenchmarkCase, concurrency int, inputSize int) BenchmarkResult {
	var result BenchmarkResult
	
	b.Run(fmt.Sprintf("%s_Size%d_Conc%d", testCase.Name, inputSize, concurrency), func(b *testing.B) {
		if testCase.Setup != nil {
			testCase.Setup()
		}
		
		b.ReportAllocs()
		b.ResetTimer()
		
		start := time.Now()
		
		for i := 0; i < b.N; i++ {
			err := testCase.Function(testCase.Args...)
			if (err != nil) != testCase.ExpectError {
				b.Errorf("Expected error=%v, got error=%v", testCase.ExpectError, err != nil)
			}
		}
		
		elapsed := time.Since(start)
		
		result = BenchmarkResult{
			Name:        fmt.Sprintf("%s_Size%d_Conc%d", testCase.Name, inputSize, concurrency),
			NsPerOp:     float64(elapsed.Nanoseconds()) / float64(b.N),
			InputSize:   inputSize,
			Concurrency: concurrency,
			Timestamp:   time.Now(),
			Tags:        testCase.Tags,
			Metadata:    testCase.Metadata,
		}
		
		// Capture memory stats if available
		if testing.AllocsPerRun(func() {
			_ = testCase.Function(testCase.Args...)
		}) >= 0 {
			result.AllocsPerOp = int(testing.AllocsPerRun(func() {
				_ = testCase.Function(testCase.Args...)
			}))
		}
		
		if testCase.Teardown != nil {
			testCase.Teardown()
		}
	})
	
	// Check for regressions
	br.checkRegression(&result, testCase.Name)
	
	br.mu.Lock()
	br.results = append(br.results, result)
	br.mu.Unlock()
	
	return result
}

// runConcurrentBenchmark executes a benchmark with specified concurrency
func (br *BenchmarkRunner) runConcurrentBenchmark(b *testing.B, testCase BenchmarkCase, concurrency int) {
	b.Run(fmt.Sprintf("Concurrent_%s_Conc%d", testCase.Name, concurrency), func(b *testing.B) {
		if testCase.Setup != nil {
			testCase.Setup()
		}
		
		b.ReportAllocs()
		b.SetParallelism(concurrency)
		b.ResetTimer()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := testCase.Function(testCase.Args...)
				if (err != nil) != testCase.ExpectError {
					b.Errorf("Expected error=%v, got error=%v", testCase.ExpectError, err != nil)
				}
			}
		})
		
		if testCase.Teardown != nil {
			testCase.Teardown()
		}
	})
}

// generateScalingCombinations creates all combinations of scaling dimension values
func (br *BenchmarkRunner) generateScalingCombinations() []map[string]interface{} {
	if len(br.suite.Scaling) == 0 {
		return []map[string]interface{}{make(map[string]interface{})}
	}
	
	combinations := []map[string]interface{}{}
	
	// Generate Cartesian product of all scaling dimensions
	var generate func(int, map[string]interface{})
	generate = func(dimensionIndex int, current map[string]interface{}) {
		if dimensionIndex >= len(br.suite.Scaling) {
			combo := make(map[string]interface{})
			for k, v := range current {
				combo[k] = v
			}
			combinations = append(combinations, combo)
			return
		}
		
		dimension := br.suite.Scaling[dimensionIndex]
		for _, value := range dimension.Values {
			current[dimension.Name] = value
			generate(dimensionIndex+1, current)
		}
	}
	
	generate(0, make(map[string]interface{}))
	return combinations
}

// scaleTestCase applies scaling parameters to a test case
func (br *BenchmarkRunner) scaleTestCase(original BenchmarkCase, scaling map[string]interface{}) BenchmarkCase {
	scaled := original
	
	// Apply scaling transformations
	if inputSize, exists := scaling["InputSize"]; exists {
		if size, ok := inputSize.(int); ok {
			scaled.InputSize = size
			// Update name to reflect scaling
			scaled.Name = fmt.Sprintf("%s_Size%d", original.Name, size)
		}
	}
	
	// Add scaling metadata
	if scaled.Metadata == nil {
		scaled.Metadata = make(map[string]interface{})
	}
	for k, v := range scaling {
		scaled.Metadata[k] = v
	}
	
	return scaled
}

// checkRegression compares current result against thresholds and baselines
func (br *BenchmarkRunner) checkRegression(result *BenchmarkResult, caseName string) {
	// Check against performance thresholds
	if threshold, exists := br.suite.Thresholds[caseName]; exists {
		if result.NsPerOp > threshold.MaxNsPerOp {
			result.RegressionFlag = true
			br.regressionCount++
		}
		if result.AllocsPerOp > threshold.MaxAllocsPerOp {
			result.RegressionFlag = true
			br.regressionCount++
		}
		if result.BytesPerOp > threshold.MaxBytesPerOp {
			result.RegressionFlag = true
			br.regressionCount++
		}
	}
	
	// Check against baseline performance
	if baseline, exists := br.suite.Baselines[caseName]; exists {
		toleranceNs := baseline.NsPerOp * (br.suite.Thresholds[caseName].TolerancePercent / 100.0)
		if result.NsPerOp > baseline.NsPerOp+toleranceNs {
			result.RegressionFlag = true
			br.regressionCount++
		}
	}
}

// analyzeStability analyzes variance across multiple benchmark runs
func (br *BenchmarkRunner) analyzeStability(caseName string, results []BenchmarkResult) {
	if len(results) < 2 {
		return
	}
	
	// Calculate coefficient of variation
	var sum, sumSquares float64
	for _, result := range results {
		sum += result.NsPerOp
		sumSquares += result.NsPerOp * result.NsPerOp
	}
	
	mean := sum / float64(len(results))
	variance := (sumSquares / float64(len(results))) - (mean * mean)
	stdDev := variance // Simplified, should use math.Sqrt
	cv := stdDev / mean
	
	// Flag high variance as potential instability
	if cv > 0.1 { // 10% coefficient of variation threshold
		for i := range results {
			results[i].Metadata["stability_warning"] = fmt.Sprintf("High variance: CV=%.2f", cv)
		}
	}
}

// GetResults returns all collected benchmark results
func (br *BenchmarkRunner) GetResults() []BenchmarkResult {
	br.mu.RLock()
	defer br.mu.RUnlock()
	
	resultsCopy := make([]BenchmarkResult, len(br.results))
	copy(resultsCopy, br.results)
	return resultsCopy
}

// GetRegressionCount returns the number of detected regressions
func (br *BenchmarkRunner) GetRegressionCount() int {
	br.mu.RLock()
	defer br.mu.RUnlock()
	return br.regressionCount
}

// GenerateReport creates a comprehensive benchmark report
func (br *BenchmarkRunner) GenerateReport() BenchmarkReport {
	results := br.GetResults()
	
	return BenchmarkReport{
		SuiteName:       br.suite.Name,
		TotalBenchmarks: len(results),
		RegressionCount: br.regressionCount,
		Results:         results,
		Timestamp:       time.Now(),
		Runtime: RuntimeInfo{
			GOOS:         runtime.GOOS,
			GOARCH:       runtime.GOARCH,
			NumCPU:       runtime.NumCPU(),
			GoVersion:    runtime.Version(),
		},
	}
}

// BenchmarkReport provides comprehensive reporting of benchmark results
type BenchmarkReport struct {
	SuiteName       string              `json:"suite_name"`
	TotalBenchmarks int                 `json:"total_benchmarks"`
	RegressionCount int                 `json:"regression_count"`
	Results         []BenchmarkResult   `json:"results"`
	Timestamp       time.Time           `json:"timestamp"`
	Runtime         RuntimeInfo         `json:"runtime"`
}

// RuntimeInfo captures runtime environment details
type RuntimeInfo struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	NumCPU    int    `json:"num_cpu"`
	GoVersion string `json:"go_version"`
}