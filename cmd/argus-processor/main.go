// Argus Processor - TDOA signal processing tool for transmitter localization
// This program processes multiple synchronized argus data files to calculate
// transmitter locations using Time Difference of Arrival (TDOA) algorithms
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"argus-collector/internal/processor"
	"argus-collector/internal/version"

	"github.com/spf13/cobra"
)

var (
	inputPattern   string   // File pattern for input files (e.g., "argus-?_*.dat")
	outputFormat   string   // Output format: geojson, kml, csv
	outputDir      string   // Output directory
	algorithm      string   // TDOA algorithm: basic, weighted, kalman
	confidence     float64  // Minimum confidence threshold
	maxDistance    float64  // Maximum expected transmitter distance (km)
	frequencyRange []string // Frequency range to analyze
	verbose        bool     // Enable verbose logging
	showVersion    bool     // Show version information
	dryRun         bool     // Show what would be processed without doing it
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "argus-processor",
	Short: "TDOA signal processing tool for transmitter localization",
	Long: `Argus Processor analyzes multiple synchronized argus data files to calculate
transmitter locations using Time Difference of Arrival (TDOA) algorithms.

The tool performs cross-correlation analysis between signal data from multiple
receivers to determine the time delays, then uses hyperbolic positioning to
calculate the transmitter location with confidence intervals.

Supported output formats:
  - GeoJSON: For web mapping applications
  - KML: For Google Earth visualization  
  - CSV: For spreadsheet analysis and custom plotting

Example usage:
  argus-processor --input "data/argus-?_1754061697.dat"
  argus-processor --input "/path/to/station*.dat" --algorithm weighted --confidence 0.8 --output-format geojson
  argus-processor --input "*.dat" --dry-run --verbose`,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle version flag
		if showVersion {
			fmt.Println(version.GetVersionInfo("Argus Processor"))
			return
		}

		if err := runProcessor(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Version flag
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "show version information")

	// Input/Output flags
	rootCmd.Flags().StringVarP(&inputPattern, "input", "i", "", "input file pattern (e.g., 'argus-?_*.dat')")
	rootCmd.Flags().StringVarP(&outputFormat, "output-format", "f", "kml", "output format (geojson, kml, csv)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "./tdoa-results", "output directory")

	// Processing flags
	rootCmd.Flags().StringVarP(&algorithm, "algorithm", "a", "basic", "TDOA algorithm (basic, weighted, kalman)")
	rootCmd.Flags().Float64VarP(&confidence, "confidence", "c", 0.5, "minimum confidence threshold (0.0-1.0)")
	rootCmd.Flags().Float64VarP(&maxDistance, "max-distance", "d", 50.0, "maximum expected transmitter distance (km)")
	rootCmd.Flags().StringSliceVar(&frequencyRange, "frequency-range", []string{}, "frequency range to analyze (e.g., '433.9-434.0')")

	// Control flags
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be processed without doing it")

	// Mark required flags, but version should be handled first
	rootCmd.MarkFlagRequired("input")

	// Handle version flag early
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.GetVersionInfo("Argus Processor"))
			os.Exit(0)
		}
		return nil
	}
}

// runProcessor is the main application logic
func runProcessor(cmd *cobra.Command) error {
	// Display banner
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║               ARGUS TDOA PROCESSOR %s                ║\n", fmt.Sprintf("%-8s", version.GetFullVersion()))
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	if verbose {
		fmt.Printf("🔧 Configuration:\n")
		fmt.Printf("   Input Pattern: %s\n", inputPattern)
		fmt.Printf("   Output Format: %s\n", outputFormat)
		fmt.Printf("   Output Directory: %s\n", outputDir)
		fmt.Printf("   Algorithm: %s\n", algorithm)
		fmt.Printf("   Confidence Threshold: %.2f\n", confidence)
		fmt.Printf("   Max Distance: %.1f km\n", maxDistance)
		if len(frequencyRange) > 0 {
			fmt.Printf("   Frequency Range: %s\n", strings.Join(frequencyRange, ", "))
		}
		fmt.Printf("   Dry Run: %t\n\n", dryRun)
	}

	// Find matching files
	files, err := findMatchingFiles(inputPattern)
	if err != nil {
		return fmt.Errorf("failed to find input files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found matching pattern '%s'. Make sure:\n  - Pattern includes correct path (e.g., 'data/argus-*.dat')\n  - Files exist and have .dat extension\n  - Pattern is quoted to prevent shell expansion", inputPattern)
	}

	if len(files) < 3 {
		return fmt.Errorf("TDOA processing requires at least 3 input files, found %d:\n%s\nPattern: '%s'\nTip: Use quotes around patterns to prevent shell expansion: --input 'data/argus*.dat'", len(files), formatFileList(files), inputPattern)
	}

	fmt.Printf("📁 Found %d input files:\n", len(files))
	for i, file := range files {
		fmt.Printf("   %d. %s\n", i+1, filepath.Base(file))
	}
	fmt.Println()

	if dryRun {
		fmt.Printf("🔍 DRY RUN: Would process %d files with %s algorithm\n", len(files), algorithm)
		fmt.Printf("📤 Would generate output in %s format to: %s\n", outputFormat, outputDir)
		return nil
	}

	// Create processor configuration
	config := &processor.Config{
		Algorithm:      algorithm,
		Confidence:     confidence,
		MaxDistance:    maxDistance,
		FrequencyRange: frequencyRange,
		Verbose:        verbose,
	}

	// Initialize processor
	proc, err := processor.NewProcessor(config)
	if err != nil {
		return fmt.Errorf("failed to initialize processor: %w", err)
	}

	// Process the files
	fmt.Printf("⚙️  Processing %d files with %s algorithm...\n", len(files), algorithm)

	// Estimate processing time based on file count
	estimatedTime := len(files) * len(files) * 30 // Rough estimate: 30 seconds per pair
	if estimatedTime > 60 {
		fmt.Printf("⏱️  Estimated processing time: ~%d minutes (large files may take longer)\n", estimatedTime/60)
	} else {
		fmt.Printf("⏱️  Estimated processing time: ~%d seconds\n", estimatedTime)
	}

	result, err := proc.ProcessFiles(files)
	if err != nil {
		return fmt.Errorf("TDOA processing failed: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate output filename
	outputFile := generateOutputFilename(result, outputFormat, outputDir)

	// Export results
	fmt.Printf("📤 Exporting results to %s...\n", outputFile)

	if err := exportResults(result, outputFormat, outputFile); err != nil {
		return fmt.Errorf("failed to export results: %w", err)
	}

	// Display summary
	displaySummary(result, outputFile)

	return nil
}

// formatFileList formats a list of files for error messages
func formatFileList(files []string) string {
	if len(files) == 0 {
		return "  (none)"
	}

	result := ""
	for i, file := range files {
		result += fmt.Sprintf("  %d. %s\n", i+1, filepath.Base(file))
	}
	return result
}

// findMatchingFiles finds files matching the input pattern
func findMatchingFiles(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	// Filter for .dat files only
	var datFiles []string
	for _, match := range matches {
		if strings.HasSuffix(strings.ToLower(match), ".dat") {
			datFiles = append(datFiles, match)
		}
	}

	return datFiles, nil
}

// generateOutputFilename creates an output filename based on processing results
func generateOutputFilename(result *processor.Result, format, outputDir string) string {
	// Format: tdoa_YYYYMMDD_HHMMSS_433920000Hz_heatmap.geojson
	timestamp := result.ProcessingTime.Format("20060102_150405")
	frequency := fmt.Sprintf("%.0fHz", result.Frequency)

	var suffix string
	switch format {
	case "geojson":
		suffix = ".geojson"
	case "kml":
		suffix = ".kml"
	case "csv":
		suffix = ".csv"
	default:
		suffix = ".json"
	}

	filename := fmt.Sprintf("tdoa_%s_%s_heatmap%s", timestamp, frequency, suffix)
	return filepath.Join(outputDir, filename)
}

// exportResults exports the processing results in the specified format
func exportResults(result *processor.Result, format, filename string) error {
	switch format {
	case "geojson":
		return result.ExportGeoJSON(filename)
	case "kml":
		return result.ExportKML(filename)
	case "csv":
		return result.ExportCSV(filename)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// displaySummary shows a summary of the processing results
func displaySummary(result *processor.Result, outputFile string) {
	fmt.Printf("\n✅ TDOA Processing Complete!\n\n")

	fmt.Printf("📊 Results Summary:\n")
	fmt.Printf("┌─────────────────────────┬─────────────────────────────────────────┐\n")
	fmt.Printf("│ Parameter               │ Value                                   │\n")
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Estimated Location      │ %.6f°, %.6f°              │\n", result.Location.Latitude, result.Location.Longitude)
	fmt.Printf("│ Confidence              │ %.2f                                  │\n", result.Confidence)
	fmt.Printf("│ Error Radius            │ %.1f meters                           │\n", result.ErrorRadius)
	fmt.Printf("│ Processing Time         │ %s                   │\n", result.ProcessingTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("│ Files Processed         │ %d                                      │\n", len(result.ReceiverLocations))
	fmt.Printf("│ Frequency               │ %.3f MHz                              │\n", result.Frequency/1e6)
	fmt.Printf("│ Algorithm               │ %-39s │\n", result.Algorithm)
	fmt.Printf("└─────────────────────────┴─────────────────────────────────────────┘\n\n")

	fmt.Printf("📁 Output File: %s\n", outputFile)
	fmt.Printf("🗺️  Open the output file in mapping software or web applications\n")
	fmt.Printf("   for visualization of the transmitter location and confidence area.\n\n")
}

// main is the entry point of the application
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
