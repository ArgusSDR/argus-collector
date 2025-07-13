// Argus Reader - Data file analysis utility
// This program analyzes Argus Collector data files and displays metadata,
// signal analysis, and optionally exports sample data.
package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"argus-collector/internal/filewriter"

	"github.com/spf13/cobra"
)

var (
	verbose       bool
	showSamples   bool
	sampleCount   int
	sampleOffset  int
	outputFile    string
	format        string
	analyze       bool
	histogram     bool
	exportCSV     bool
	exportJSON    bool
)

// SignalAnalysis contains detailed signal analysis results
type SignalAnalysis struct {
	MinPower       float64
	MaxPower       float64
	AvgPower       float64
	RMSPower       float64
	PeakToPeak     float64
	DynamicRange   float64
	SNREstimate    float64
	SampleCount    uint32
	PowerHistogram []HistogramBin
}

// HistogramBin represents a power histogram bin
type HistogramBin struct {
	MinPower float64
	MaxPower float64
	Count    int
	Percent  float64
}

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "argus-reader [file.dat]",
	Short: "Analyze Argus Collector data files",
	Long: `Argus Reader analyzes data files created by Argus Collector and provides
detailed information about the collected RF signals including metadata,
signal strength analysis, and sample data export capabilities.

Examples:
  argus-reader data.dat                    # Show basic file information
  argus-reader data.dat --analyze          # Perform detailed signal analysis
  argus-reader data.dat --samples --count 1000  # Show first 1000 samples
  argus-reader data.dat --export-csv       # Export sample data to CSV
  argus-reader data.dat --histogram        # Show power distribution histogram`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runReader(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.Flags().BoolVarP(&showSamples, "samples", "s", false, "display sample data")
	rootCmd.Flags().IntVarP(&sampleCount, "count", "c", 10, "number of samples to display")
	rootCmd.Flags().IntVarP(&sampleOffset, "offset", "o", 0, "sample offset to start from")
	rootCmd.Flags().StringVar(&outputFile, "output", "", "output file for exports")
	rootCmd.Flags().StringVar(&format, "format", "table", "output format (table, csv, json)")
	rootCmd.Flags().BoolVarP(&analyze, "analyze", "a", false, "perform detailed signal analysis")
	rootCmd.Flags().BoolVar(&histogram, "histogram", false, "show power distribution histogram")
	rootCmd.Flags().BoolVar(&exportCSV, "export-csv", false, "export sample data to CSV")
	rootCmd.Flags().BoolVar(&exportJSON, "export-json", false, "export metadata to JSON")
}

// runReader is the main reader function
func runReader(filename string) error {
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                      ARGUS DATA READER                      ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	// Read metadata (fast operation)
	metadata, totalSamples, err := filewriter.ReadMetadata(filename)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Display basic file information
	displayFileInfo(filename, metadata, totalSamples)

	// Display metadata details
	displayMetadata(metadata)

	// Perform signal analysis if requested
	if analyze || histogram {
		analysis, err := performSignalAnalysis(filename, totalSamples)
		if err != nil {
			return fmt.Errorf("signal analysis failed: %w", err)
		}
		displaySignalAnalysis(analysis)

		if histogram {
			displayHistogram(analysis.PowerHistogram)
		}
	}

	// Display sample data if requested
	if showSamples {
		if err := displaySamples(filename, sampleOffset, sampleCount); err != nil {
			return fmt.Errorf("failed to display samples: %w", err)
		}
	}

	// Export data if requested
	if exportCSV {
		if err := exportSampleData(filename, outputFile); err != nil {
			return fmt.Errorf("CSV export failed: %w", err)
		}
	}

	if exportJSON {
		if err := exportMetadataJSON(metadata, outputFile); err != nil {
			return fmt.Errorf("JSON export failed: %w", err)
		}
	}

	return nil
}

// displayFileInfo shows basic file information
func displayFileInfo(filename string, metadata *filewriter.Metadata, sampleCount uint32) {
	// Get file size
	fileInfo, err := os.Stat(filename)
	var fileSize int64
	if err == nil {
		fileSize = fileInfo.Size()
	}

	fmt.Printf("📄 File Information:\n")
	fmt.Printf("┌─────────────────────────┬─────────────────────────────────────────┐\n")
	fmt.Printf("│ Parameter               │ Value                                   │\n")
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Filename                │ %-39s │\n", filename)
	fmt.Printf("│ File Size               │ %-39s │\n", formatBytes(fileSize))
	fmt.Printf("│ Collection ID           │ %-39s │\n", metadata.CollectionID)
	fmt.Printf("│ File Format Version     │ %-39d │\n", metadata.FileFormatVersion)
	fmt.Printf("│ Total Samples           │ %-39d │\n", sampleCount)
	
	// Calculate duration
	duration := float64(sampleCount) / float64(metadata.SampleRate)
	fmt.Printf("│ Collection Duration     │ %-39s │\n", formatDuration(duration))
	
	// Calculate data rate
	dataRate := float64(sampleCount) * 8 / duration // 8 bytes per complex64 sample
	fmt.Printf("│ Data Rate               │ %-39s │\n", formatDataRate(dataRate))
	
	fmt.Printf("└─────────────────────────┴─────────────────────────────────────────┘\n\n")
}

// displayMetadata shows detailed metadata information
func displayMetadata(metadata *filewriter.Metadata) {
	fmt.Printf("📡 Collection Metadata:\n")
	fmt.Printf("┌─────────────────────────┬─────────────────────────────────────────┐\n")
	fmt.Printf("│ Parameter               │ Value                                   │\n")
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Frequency               │ %.3f MHz                              │\n", float64(metadata.Frequency)/1e6)
	fmt.Printf("│ Sample Rate             │ %.3f MSps                             │\n", float64(metadata.SampleRate)/1e6)
	fmt.Printf("│ Collection Time         │ %-39s │\n", metadata.CollectionTime.Format("2006-01-02 15:04:05.000 MST"))
	fmt.Printf("│ GPS Timestamp           │ %-39s │\n", metadata.GPSTimestamp.Format("2006-01-02 15:04:05.000 MST"))
	fmt.Printf("│ Device Info             │ %-39s │\n", metadata.DeviceInfo)
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ GPS Latitude            │ %14.8f°                        │\n", metadata.GPSLocation.Latitude)
	fmt.Printf("│ GPS Longitude           │ %14.8f°                        │\n", metadata.GPSLocation.Longitude)
	fmt.Printf("│ GPS Altitude            │ %14.2f m                         │\n", metadata.GPSLocation.Altitude)
	fmt.Printf("└─────────────────────────┴─────────────────────────────────────────┘\n\n")
}

// performSignalAnalysis analyzes signal characteristics
func performSignalAnalysis(filename string, totalSamples uint32) (*SignalAnalysis, error) {
	fmt.Printf("📊 Performing Signal Analysis...\n")
	
	// Determine how many samples to analyze
	analyzeCount := uint32(math.Min(float64(totalSamples), 1000000)) // Max 1M samples for analysis
	if verbose {
		fmt.Printf("   Analyzing %d of %d total samples\n", analyzeCount, totalSamples)
	}

	// Read samples for analysis
	samples, err := filewriter.ReadSamples(filename, 0, analyzeCount)
	if err != nil {
		return nil, err
	}

	analysis := &SignalAnalysis{
		SampleCount: uint32(len(samples)),
		MinPower:    math.Inf(1),
		MaxPower:    math.Inf(-1),
	}

	var sumPower, sumSquaredPower float64
	var powers []float64

	// Calculate signal statistics
	for _, sample := range samples {
		// Calculate instantaneous power: |I + jQ|²
		power := float64(real(sample)*real(sample) + imag(sample)*imag(sample))
		powerDBm := 10.0 * math.Log10(power + 1e-12) // Convert to dBm with noise floor protection

		powers = append(powers, powerDBm)
		sumPower += powerDBm
		sumSquaredPower += powerDBm * powerDBm

		if powerDBm < analysis.MinPower {
			analysis.MinPower = powerDBm
		}
		if powerDBm > analysis.MaxPower {
			analysis.MaxPower = powerDBm
		}
	}

	// Calculate statistics
	n := float64(len(samples))
	analysis.AvgPower = sumPower / n
	analysis.RMSPower = math.Sqrt(sumSquaredPower / n)
	analysis.PeakToPeak = analysis.MaxPower - analysis.MinPower
	analysis.DynamicRange = analysis.PeakToPeak

	// Estimate SNR (simple noise floor estimation)
	sort.Float64s(powers)
	noiseFloor := powers[int(0.1*float64(len(powers)))] // Bottom 10% as noise estimate
	signalPeak := powers[int(0.9*float64(len(powers)))] // Top 10% as signal estimate
	analysis.SNREstimate = signalPeak - noiseFloor

	// Create power histogram
	analysis.PowerHistogram = createPowerHistogram(powers, 20) // 20 bins

	if verbose {
		fmt.Printf("   ✓ Analyzed %d samples\n", len(samples))
	}

	return analysis, nil
}

// createPowerHistogram creates a power distribution histogram
func createPowerHistogram(powers []float64, numBins int) []HistogramBin {
	if len(powers) == 0 {
		return nil
	}

	minPower := powers[0]
	maxPower := powers[len(powers)-1]
	binWidth := (maxPower - minPower) / float64(numBins)

	bins := make([]HistogramBin, numBins)
	for i := range bins {
		bins[i].MinPower = minPower + float64(i)*binWidth
		bins[i].MaxPower = minPower + float64(i+1)*binWidth
	}

	// Count samples in each bin
	totalSamples := len(powers)
	for _, power := range powers {
		binIndex := int((power - minPower) / binWidth)
		if binIndex >= numBins {
			binIndex = numBins - 1
		}
		if binIndex < 0 {
			binIndex = 0
		}
		bins[binIndex].Count++
	}

	// Calculate percentages
	for i := range bins {
		bins[i].Percent = float64(bins[i].Count) * 100.0 / float64(totalSamples)
	}

	return bins
}

// displaySignalAnalysis shows signal analysis results
func displaySignalAnalysis(analysis *SignalAnalysis) {
	fmt.Printf("📈 Signal Analysis Results:\n")
	fmt.Printf("┌─────────────────────────┬─────────────────────────────────────────┐\n")
	fmt.Printf("│ Parameter               │ Value                                   │\n")
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Samples Analyzed        │ %-39d │\n", analysis.SampleCount)
	fmt.Printf("│ Minimum Power           │ %14.2f dBm                      │\n", analysis.MinPower)
	fmt.Printf("│ Maximum Power           │ %14.2f dBm                      │\n", analysis.MaxPower)
	fmt.Printf("│ Average Power           │ %14.2f dBm                      │\n", analysis.AvgPower)
	fmt.Printf("│ RMS Power               │ %14.2f dBm                      │\n", analysis.RMSPower)
	fmt.Printf("│ Peak-to-Peak Range      │ %14.2f dB                       │\n", analysis.PeakToPeak)
	fmt.Printf("│ Dynamic Range           │ %14.2f dB                       │\n", analysis.DynamicRange)
	fmt.Printf("│ SNR Estimate            │ %14.2f dB                       │\n", analysis.SNREstimate)
	fmt.Printf("└─────────────────────────┴─────────────────────────────────────────┘\n\n")
}

// displayHistogram shows the power distribution histogram
func displayHistogram(histogram []HistogramBin) {
	fmt.Printf("📊 Power Distribution Histogram:\n")
	fmt.Printf("┌─────────────────┬─────────────────┬─────────┬──────────┬─────────────────────┐\n")
	fmt.Printf("│ Min Power (dBm) │ Max Power (dBm) │ Count   │ Percent  │ Distribution        │\n")
	fmt.Printf("├─────────────────┼─────────────────┼─────────┼──────────┼─────────────────────┤\n")

	maxCount := 0
	for _, bin := range histogram {
		if bin.Count > maxCount {
			maxCount = bin.Count
		}
	}

	for _, bin := range histogram {
		// Create visual bar
		barLength := int(20 * float64(bin.Count) / float64(maxCount))
		bar := strings.Repeat("█", barLength) + strings.Repeat("░", 20-barLength)

		fmt.Printf("│ %13.1f   │ %13.1f   │ %7d │ %7.2f%% │ %s │\n",
			bin.MinPower, bin.MaxPower, bin.Count, bin.Percent, bar)
	}

	fmt.Printf("└─────────────────┴─────────────────┴─────────┴──────────┴─────────────────────┘\n\n")
}

// displaySamples shows sample data
func displaySamples(filename string, offset, count int) error {
	fmt.Printf("📋 Sample Data (showing %d samples starting at offset %d):\n", count, offset)

	samples, err := filewriter.ReadSamples(filename, uint32(offset), uint32(count))
	if err != nil {
		return err
	}

	fmt.Printf("┌─────────┬─────────────────┬─────────────────┬─────────────────┐\n")
	fmt.Printf("│ Index   │ I (Real)        │ Q (Imaginary)   │ Magnitude       │\n")
	fmt.Printf("├─────────┼─────────────────┼─────────────────┼─────────────────┤\n")

	for i, sample := range samples {
		magnitude := math.Sqrt(float64(real(sample)*real(sample) + imag(sample)*imag(sample)))
		fmt.Printf("│ %7d │ %13.6f   │ %13.6f   │ %13.6f   │\n",
			offset+i, real(sample), imag(sample), magnitude)
	}

	fmt.Printf("└─────────┴─────────────────┴─────────────────┴─────────────────┘\n\n")
	return nil
}

// exportSampleData exports sample data to CSV
func exportSampleData(filename, outputFile string) error {
	if outputFile == "" {
		outputFile = strings.TrimSuffix(filename, ".dat") + "_samples.csv"
	}

	fmt.Printf("📤 Exporting sample data to: %s\n", outputFile)

	// Read all samples
	_, samples, err := filewriter.ReadFile(filename)
	if err != nil {
		return err
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write CSV header
	fmt.Fprintf(file, "Index,I_Real,Q_Imaginary,Magnitude,Power_dBm\n")

	// Write sample data
	for i, sample := range samples {
		magnitude := math.Sqrt(float64(real(sample)*real(sample) + imag(sample)*imag(sample)))
		power := float64(real(sample)*real(sample) + imag(sample)*imag(sample))
		powerDBm := 10.0 * math.Log10(power + 1e-12)

		fmt.Fprintf(file, "%d,%.6f,%.6f,%.6f,%.2f\n",
			i, real(sample), imag(sample), magnitude, powerDBm)
	}

	fmt.Printf("   ✓ Exported %d samples\n\n", len(samples))
	return nil
}

// exportMetadataJSON exports metadata to JSON
func exportMetadataJSON(metadata *filewriter.Metadata, outputFile string) error {
	if outputFile == "" {
		outputFile = "metadata.json"
	}

	fmt.Printf("📤 Exporting metadata to: %s\n", outputFile)

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Simple JSON export (could use encoding/json for more complex cases)
	fmt.Fprintf(file, "{\n")
	fmt.Fprintf(file, "  \"collection_id\": \"%s\",\n", metadata.CollectionID)
	fmt.Fprintf(file, "  \"file_format_version\": %d,\n", metadata.FileFormatVersion)
	fmt.Fprintf(file, "  \"frequency\": %d,\n", metadata.Frequency)
	fmt.Fprintf(file, "  \"sample_rate\": %d,\n", metadata.SampleRate)
	fmt.Fprintf(file, "  \"collection_time\": \"%s\",\n", metadata.CollectionTime.Format("2006-01-02T15:04:05.000Z"))
	fmt.Fprintf(file, "  \"gps_timestamp\": \"%s\",\n", metadata.GPSTimestamp.Format("2006-01-02T15:04:05.000Z"))
	fmt.Fprintf(file, "  \"device_info\": \"%s\",\n", metadata.DeviceInfo)
	fmt.Fprintf(file, "  \"gps_location\": {\n")
	fmt.Fprintf(file, "    \"latitude\": %.8f,\n", metadata.GPSLocation.Latitude)
	fmt.Fprintf(file, "    \"longitude\": %.8f,\n", metadata.GPSLocation.Longitude)
	fmt.Fprintf(file, "    \"altitude\": %.2f\n", metadata.GPSLocation.Altitude)
	fmt.Fprintf(file, "  }\n")
	fmt.Fprintf(file, "}\n")

	fmt.Printf("   ✓ Metadata exported\n\n")
	return nil
}

// Utility functions for formatting

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(seconds float64) string {
	if seconds < 1 {
		return fmt.Sprintf("%.1f ms", seconds*1000)
	} else if seconds < 60 {
		return fmt.Sprintf("%.1f s", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%.1f min", seconds/60)
	} else {
		return fmt.Sprintf("%.1f hr", seconds/3600)
	}
}

func formatDataRate(bytesPerSecond float64) string {
	const unit = 1024
	if bytesPerSecond < unit {
		return fmt.Sprintf("%.1f B/s", bytesPerSecond)
	}
	div, exp := float64(unit), 0
	for n := bytesPerSecond / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB/s", bytesPerSecond/div, "KMGTPE"[exp])
}

// main is the entry point of the application
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}