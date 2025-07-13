// Argus Reader - Utility to display contents of Argus Collector data files
// This program reads and displays the metadata and sample information from .dat files
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"argus-collector/internal/filewriter"

	"github.com/spf13/cobra"
)

var (
	showSamples   bool
	sampleLimit   int
	showStats     bool
	outputFormat  string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "argus-reader [file.dat]",
	Short: "Display contents of Argus Collector data files",
	Long: `Argus Reader displays the metadata and sample data from Argus Collector .dat files.
Useful for analyzing collected RF data and verifying collection parameters.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := displayFile(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&showSamples, "samples", "s", false, "display IQ sample data")
	rootCmd.Flags().IntVarP(&sampleLimit, "limit", "l", 10, "limit number of samples to display")
	rootCmd.Flags().BoolVar(&showStats, "stats", false, "show statistical analysis of samples")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "output format (table, json, csv)")
}

// displayFile reads and displays the contents of an Argus data file
func displayFile(filename string) error {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filename)
	}

	// Read metadata only (fast)
	metadata, sampleCount, err := filewriter.ReadMetadata(filename)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// We only need the sample count for metadata display
	// Actual samples will be loaded only if requested

	// Display file information
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                    ARGUS DATA FILE READER                   ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	// Display file info
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return err
	}

	fmt.Printf("📁 File Information:\n")
	fmt.Printf("   Name: %s\n", filepath.Base(filename))
	fmt.Printf("   Size: %.2f MB (%d bytes)\n", float64(fileInfo.Size())/(1024*1024), fileInfo.Size())
	fmt.Printf("   Modified: %s\n\n", fileInfo.ModTime().Format("2006-01-02 15:04:05"))

	// Display metadata
	displayMetadata(metadata)

	// Display sample information (using count only)
	displaySampleInfo(int(sampleCount), metadata.SampleRate)

	// Load and display sample data if requested
	if showSamples || showStats {
		fmt.Printf("⏳ Loading sample data...\n")
		
		// Read all samples (this is slow but accurate)
		_, allSamples, err := filewriter.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read samples: %w", err)
		}
		
		if showSamples {
			// Show limited samples for display
			limitedSamples := allSamples
			if len(allSamples) > sampleLimit {
				limitedSamples = allSamples[:sampleLimit]
			}
			displaySampleData(limitedSamples)
		}

		if showStats {
			// Use subset for stats if file is very large
			statsSamples := allSamples
			if len(allSamples) > 100000 {
				// Use every Nth sample for large files
				step := len(allSamples) / 50000
				statsSamples = make([]complex64, 0, 50000)
				for i := 0; i < len(allSamples); i += step {
					statsSamples = append(statsSamples, allSamples[i])
				}
				fmt.Printf("📊 Statistics calculated from %d representative samples\n", len(statsSamples))
			}
			displayStatistics(statsSamples)
		}
	}

	return nil
}

// displayMetadata shows the file metadata in a formatted table
func displayMetadata(metadata *filewriter.Metadata) {
	fmt.Printf("📊 Collection Metadata:\n")
	fmt.Printf("┌─────────────────────────┬─────────────────────────────────────────┐\n")
	fmt.Printf("│ Parameter               │ Value                                   │\n")
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ File Format Version     │ %d                                       │\n", metadata.FileFormatVersion)
	fmt.Printf("│ Collection ID           │ %-39s │\n", metadata.CollectionID)
	fmt.Printf("│ Device Info             │ %-39s │\n", metadata.DeviceInfo)
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Frequency               │ %.3f MHz                              │\n", float64(metadata.Frequency)/1e6)
	fmt.Printf("│ Sample Rate             │ %.3f MSps                             │\n", float64(metadata.SampleRate)/1e6)
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Collection Time         │ %s │\n", metadata.CollectionTime.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("│ GPS Timestamp           │ %s │\n", metadata.GPSTimestamp.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ GPS Latitude            │ %14.8f°                        │\n", metadata.GPSLocation.Latitude)
	fmt.Printf("│ GPS Longitude           │ %14.8f°                        │\n", metadata.GPSLocation.Longitude)
	fmt.Printf("│ GPS Altitude            │ %14.2f m                         │\n", metadata.GPSLocation.Altitude)
	fmt.Printf("└─────────────────────────┴─────────────────────────────────────────┘\n\n")
}

// displaySampleInfo shows information about the IQ samples
func displaySampleInfo(sampleCount int, sampleRate uint32) {
	duration := float64(sampleCount) / float64(sampleRate)
	
	fmt.Printf("📡 Sample Information:\n")
	fmt.Printf("┌─────────────────────────┬─────────────────────────────────────────┐\n")
	fmt.Printf("│ Parameter               │ Value                                   │\n")
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Total Samples           │ %d                                    │\n", sampleCount)
	fmt.Printf("│ Sample Type             │ Complex64 (32-bit I + 32-bit Q)        │\n")
	fmt.Printf("│ Data Size               │ %.2f MB                               │\n", float64(sampleCount*8)/(1024*1024))
	fmt.Printf("│ Collection Duration     │ %.3f seconds                          │\n", duration)
	fmt.Printf("└─────────────────────────┴─────────────────────────────────────────┘\n\n")
}

// displaySampleData shows the actual IQ sample values
func displaySampleData(samples []complex64) {
	fmt.Printf("📈 IQ Sample Data (first %d samples):\n", sampleLimit)
	fmt.Printf("┌──────┬──────────────┬──────────────┬──────────────┬────────────┐\n")
	fmt.Printf("│ #    │ I (Real)     │ Q (Imag)     │ Magnitude    │ Phase (°)  │\n")
	fmt.Printf("├──────┼──────────────┼──────────────┼──────────────┼────────────┤\n")

	limit := sampleLimit
	if limit > len(samples) {
		limit = len(samples)
	}

	for i := 0; i < limit; i++ {
		sample := samples[i]
		magnitude := math.Sqrt(float64(real(sample)*real(sample) + imag(sample)*imag(sample)))
		phase := math.Atan2(float64(imag(sample)), float64(real(sample))) * 180 / math.Pi

		fmt.Printf("│ %-4d │ %12.6f │ %12.6f │ %12.6f │ %10.2f │\n", 
			i, real(sample), imag(sample), magnitude, phase)
	}

	fmt.Printf("└──────┴──────────────┴──────────────┴──────────────┴────────────┘\n")
	
	if len(samples) > sampleLimit {
		fmt.Printf("... (%d more samples not shown)\n", len(samples)-sampleLimit)
	}
	fmt.Println()
}

// displayStatistics shows statistical analysis of the samples
func displayStatistics(samples []complex64) {
	if len(samples) == 0 {
		fmt.Printf("📊 Statistics: No samples to analyze\n\n")
		return
	}

	// Calculate statistics
	var sumI, sumQ, sumMag float64
	var minMag, maxMag float64 = math.Inf(1), math.Inf(-1)
	var sumPower float64

	for _, sample := range samples {
		i := float64(real(sample))
		q := float64(imag(sample))
		mag := math.Sqrt(i*i + q*q)
		power := i*i + q*q

		sumI += i
		sumQ += q
		sumMag += mag
		sumPower += power

		if mag < minMag {
			minMag = mag
		}
		if mag > maxMag {
			maxMag = mag
		}
	}

	count := float64(len(samples))
	meanI := sumI / count
	meanQ := sumQ / count
	meanMag := sumMag / count
	meanPower := sumPower / count

	// Calculate variance for I and Q
	var varI, varQ float64
	for _, sample := range samples {
		i := float64(real(sample))
		q := float64(imag(sample))
		varI += (i - meanI) * (i - meanI)
		varQ += (q - meanQ) * (q - meanQ)
	}
	varI /= count
	varQ /= count

	fmt.Printf("📊 Statistical Analysis:\n")
	fmt.Printf("┌─────────────────────────┬─────────────────────────────────────────┐\n")
	fmt.Printf("│ Statistic               │ Value                                   │\n")
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Mean I (Real)           │ %12.6f                           │\n", meanI)
	fmt.Printf("│ Mean Q (Imaginary)      │ %12.6f                           │\n", meanQ)
	fmt.Printf("│ I Variance              │ %12.6f                           │\n", varI)
	fmt.Printf("│ Q Variance              │ %12.6f                           │\n", varQ)
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Mean Magnitude          │ %12.6f                           │\n", meanMag)
	fmt.Printf("│ Min Magnitude           │ %12.6f                           │\n", minMag)
	fmt.Printf("│ Max Magnitude           │ %12.6f                           │\n", maxMag)
	fmt.Printf("│ Mean Power              │ %12.6f                           │\n", meanPower)
	fmt.Printf("│ Power (dB)              │ %12.2f dB                        │\n", 10*math.Log10(meanPower))
	fmt.Printf("└─────────────────────────┴─────────────────────────────────────────┘\n\n")
}

// main is the entry point of the application
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}