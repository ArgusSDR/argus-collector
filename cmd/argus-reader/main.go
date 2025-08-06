// Argus Reader - Utility to display contents of Argus Collector data files
// This program reads and displays the metadata and sample information from .dat files
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"argus-collector/internal/filewriter"
	"argus-collector/internal/version"

	"github.com/spf13/cobra"
)

var (
	showSamples        bool
	showStats          bool
	outputFormat       string
	showHex            bool
	showGraph          bool
	graphWidth         int
	graphHeight        int
	graphSamples       int
	showVersion        bool
	showDeviceAnalysis bool
)

// DeviceSettings contains parsed device configuration information
type DeviceSettings struct {
	Name     string
	Gain     string
	GainMode string
	BiasTee  string
}

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "argus-reader [file.dat]",
	Short: "Display contents of Argus Collector data files",
	Long: `Argus Reader displays the metadata and sample data from Argus Collector .dat files.
Useful for analyzing collected RF data and verifying collection parameters.

Display modes:
  --samples    Show all decoded IQ sample values (magnitude, phase)
  --hex        Show complete raw hexadecimal dump of sample data bytes
  --stats      Show statistical analysis of sample data
  --graph      Generate ASCII graph of signal magnitude over time`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Handle version flag
		if showVersion {
			fmt.Println(version.GetVersionInfo("Argus Reader"))
			return
		}

		// Require filename if not showing version
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Error: filename required\n")
			cmd.Usage()
			os.Exit(1)
		}

		if err := displayFile(args[0], cmd); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "show version information")
	rootCmd.Flags().BoolVarP(&showSamples, "samples", "s", false, "display all IQ sample data")
	rootCmd.Flags().BoolVar(&showStats, "stats", false, "show statistical analysis of samples")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "output format (table, json, csv)")
	rootCmd.Flags().BoolVar(&showHex, "hex", false, "display all raw sample data as hexadecimal dump")
	rootCmd.Flags().BoolVarP(&showGraph, "graph", "g", false, "generate ASCII graph of signal magnitude over time")
	rootCmd.Flags().IntVar(&graphWidth, "graph-width", 80, "width of the ASCII graph in characters")
	rootCmd.Flags().IntVar(&graphHeight, "graph-height", 20, "height of the ASCII graph in lines")
	rootCmd.Flags().IntVar(&graphSamples, "graph-samples", 1000, "number of samples to include in graph")

	// Add a device info analysis flag
	rootCmd.Flags().BoolVar(&showDeviceAnalysis, "device-analysis", false, "show detailed device configuration analysis")
}

// displayFile reads and displays the contents of an Argus data file
func displayFile(filename string, cmd *cobra.Command) error {
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
	fmt.Printf("ARGUS DATA FILE READER %s\n\n", version.GetFullVersion())

	// Display file info
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return err
	}

	fmt.Printf("📁 File Information:\n")
	fmt.Printf("Name: %s\n", filepath.Base(filename))
	fmt.Printf("Size: %.2f MB (%d bytes)\n", float64(fileInfo.Size())/(1024*1024), fileInfo.Size())
	fmt.Printf("Modified: %s\n\n", fileInfo.ModTime().Format("2006-01-02 15:04:05"))

	// Display metadata
	displayMetadata(metadata)

	// Display device analysis if requested
	if showDeviceAnalysis {
		deviceSettings := parseDeviceInfo(metadata.DeviceInfo)
		displayDeviceAnalysis(deviceSettings)
	}

	// Display sample information (using count only)
	displaySampleInfo(int(sampleCount), metadata.SampleRate)

	// Handle sample data display if requested
	if showSamples || showStats || showHex || showGraph {
		// For samples and hex, use streaming display
		if showSamples {
			if err := displaySamplesStreaming(filename, metadata, int(sampleCount)); err != nil {
				return fmt.Errorf("failed to display samples: %w", err)
			}
		}

		if showHex {
			if err := displayHexStreaming(filename, metadata, int(sampleCount)); err != nil {
				return fmt.Errorf("failed to display hex dump: %w", err)
			}
		}
		
		// For graph and stats, load samples into memory (these need all data for analysis)
		if showGraph || showStats {
			maxSamplesNeeded := 100000 // Default for stats
			if showGraph {
				actualGraphSamples := graphSamples
				if !cmd.Flags().Changed("graph-samples") {
					actualGraphSamples = min(int(sampleCount), 10000)
				}
				maxSamplesNeeded = max(maxSamplesNeeded, actualGraphSamples)
			}
			
			fmt.Printf("⏳ Loading %d samples for analysis...\n", maxSamplesNeeded)
			samples, err := readLimitedSamples(filename, maxSamplesNeeded)
			if err != nil {
				return fmt.Errorf("failed to read samples: %w", err)
			}

			if showGraph {
				actualGraphSamples := graphSamples
				if !cmd.Flags().Changed("graph-samples") {
					actualGraphSamples = min(int(sampleCount), 10000)
				}
				
				graphSampleData := samples
				if len(samples) > actualGraphSamples {
					// Use evenly spaced samples
					step := len(samples) / actualGraphSamples
					graphSampleData = make([]complex64, 0, actualGraphSamples)
					for i := 0; i < len(samples); i += step {
						if len(graphSampleData) < actualGraphSamples {
							graphSampleData = append(graphSampleData, samples[i])
						}
					}
				}
				displayGraph(graphSampleData, metadata.SampleRate)
			}

			if showStats {
				statsSamples := samples
				if len(samples) > 100000 {
					// Use every Nth sample for large datasets
					step := len(samples) / 50000
					statsSamples = make([]complex64, 0, 50000)
					for i := 0; i < len(samples); i += step {
						statsSamples = append(statsSamples, samples[i])
					}
					fmt.Printf("📊 Statistics calculated from %d representative samples\n", len(statsSamples))
				}
				displayStatistics(statsSamples)
			}
		}
	}

	return nil
}

// readSamplesRobust reads samples from the file with error handling for truncated files
func readSamplesRobust(filename string) (*filewriter.Metadata, []complex64, error) {
	// First try the normal read method
	metadata, samples, err := filewriter.ReadFile(filename)
	if err == nil {
		return metadata, samples, nil
	}

	// If we get EOF, the file might be truncated. Try to read what we can.
	fmt.Printf("⚠️  File appears truncated, attempting partial read...\n")

	// Read metadata to get the header info
	metadataOnly, sampleCountFromHeader, err := filewriter.ReadMetadata(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Calculate actual available samples based on file size
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Estimate header size (this is approximate but safer than trusting header count)
	// Magic(5) + FileFormatVersion(2) + Frequency(8) + SampleRate(4) + CollectionTime(12) +
	// GPS(24) + GPSTime(12) + DeviceInfoLen(1) + DeviceInfo + CollectionIDLen(1) + CollectionID + SampleCount(4)
	estimatedHeaderSize := int64(5 + 2 + 8 + 4 + 12 + 24 + 12 + 1 + len(metadataOnly.DeviceInfo) + 1 + len(metadataOnly.CollectionID) + 4)

	availableDataBytes := fileInfo.Size() - estimatedHeaderSize
	availableSamples := availableDataBytes / 8 // Each complex64 is 8 bytes (4 bytes real + 4 bytes imag)

	fmt.Printf("📊 File analysis:\n")
	fmt.Printf("   Header claims: %d samples (%.2f MB)\n", sampleCountFromHeader, float64(sampleCountFromHeader*8)/(1024*1024))
	fmt.Printf("   File size: %d bytes (%.2f MB)\n", fileInfo.Size(), float64(fileInfo.Size())/(1024*1024))
	fmt.Printf("   Estimated header size: %d bytes\n", estimatedHeaderSize)
	fmt.Printf("   Available for samples: %d bytes\n", availableDataBytes)
	fmt.Printf("   Actual readable samples: %d\n", availableSamples)

	if availableSamples <= 0 {
		return metadataOnly, []complex64{}, nil
	}

	// Read only what's actually available
	samples, err = readSamplesFromFile(filename, int(availableSamples))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read available samples: %w", err)
	}

	return metadataOnly, samples, nil
}

// readSamplesFromFile reads a specific number of samples from the file
func readSamplesFromFile(filename string, maxSamples int) ([]complex64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Skip to after the header by reading metadata first
	_, _, err = filewriter.ReadMetadata(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// We need to reopen and seek past the header manually
	// This is a simplified approach - read until we get to sample data
	file.Seek(0, 0) // Start from beginning

	// Skip magic
	file.Seek(5, 0)

	// Skip metadata fields in the same order as ReadFile
	var skipBuffer [8]byte

	// Skip FileFormatVersion (2 bytes)
	file.Read(skipBuffer[:2])
	// Skip Frequency (8 bytes)
	file.Read(skipBuffer[:8])
	// Skip SampleRate (4 bytes)
	file.Read(skipBuffer[:4])
	// Skip CollectionTime (12 bytes)
	file.Read(skipBuffer[:8])
	file.Read(skipBuffer[:4])
	// Skip GPS Location (24 bytes)
	file.Read(skipBuffer[:8])
	file.Read(skipBuffer[:8])
	file.Read(skipBuffer[:8])
	// Skip GPS timestamp (12 bytes)
	file.Read(skipBuffer[:8])
	file.Read(skipBuffer[:4])

	// Skip device info
	var deviceInfoLen uint8
	binary.Read(file, binary.LittleEndian, &deviceInfoLen)
	file.Seek(int64(deviceInfoLen), 1)

	// Skip collection ID
	var collectionIDLen uint8
	binary.Read(file, binary.LittleEndian, &collectionIDLen)
	file.Seek(int64(collectionIDLen), 1)

	// Skip sample count field
	file.Seek(4, 1)

	// Now read samples until EOF or maxSamples
	samples := make([]complex64, 0, maxSamples)

	for len(samples) < maxSamples {
		var real, imag float32
		if err := binary.Read(file, binary.LittleEndian, &real); err != nil {
			break // EOF or error
		}
		if err := binary.Read(file, binary.LittleEndian, &imag); err != nil {
			break // EOF or error
		}
		samples = append(samples, complex(real, imag))
	}

	return samples, nil
}

// readLimitedSamples reads only a limited number of samples from the beginning of the file
func readLimitedSamples(filename string, maxSamples int) ([]complex64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Skip to after the header by reading metadata first to get header size
	metadata, _, err := filewriter.ReadMetadata(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Seek to start of sample data (after header)
	// Magic(5) + FileFormatVersion(2) + Frequency(8) + SampleRate(4) + CollectionTime(12) +
	// GPS(24) + GPSTime(12) + DeviceInfoLen(1) + DeviceInfo + CollectionIDLen(1) + CollectionID + SampleCount(4)
	headerSize := int64(5 + 2 + 8 + 4 + 12 + 24 + 12 + 1 + len(metadata.DeviceInfo) + 1 + len(metadata.CollectionID) + 4)
	
	_, err = file.Seek(headerSize, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to sample data: %w", err)
	}

	// Read samples directly
	samples := make([]complex64, 0, maxSamples)
	for len(samples) < maxSamples {
		var real, imag float32
		if err := binary.Read(file, binary.LittleEndian, &real); err != nil {
			break // EOF or error
		}
		if err := binary.Read(file, binary.LittleEndian, &imag); err != nil {
			break // EOF or error
		}
		samples = append(samples, complex(real, imag))
	}

	return samples, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers  
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// parseDeviceInfo extracts device settings from the device info string
func parseDeviceInfo(deviceInfo string) DeviceSettings {
	settings := DeviceSettings{
		Name:     deviceInfo, // Fallback to full string
		Gain:     "Unknown",
		GainMode: "Unknown",
		BiasTee:  "Unknown",
	}

	// Extract device name (everything before the first parenthesis)
	if nameMatch := regexp.MustCompile(`^([^(]+)`).FindStringSubmatch(deviceInfo); nameMatch != nil {
		settings.Name = strings.TrimSpace(nameMatch[1])
	}

	// Extract gain information: "gain: 20.7 dB (manual)"
	gainRegex := regexp.MustCompile(`gain:\s*([0-9.]+)\s*dB\s*\(([^)]+)\)`)
	if gainMatch := gainRegex.FindStringSubmatch(deviceInfo); gainMatch != nil {
		settings.Gain = gainMatch[1] + " dB"
		settings.GainMode = gainMatch[2]
	}

	// Extract bias tee information: "bias-tee: off"
	biasRegex := regexp.MustCompile(`bias-tee:\s*(\w+)`)
	if biasMatch := biasRegex.FindStringSubmatch(deviceInfo); biasMatch != nil {
		settings.BiasTee = biasMatch[1]
	}

	return settings
}

// displayMetadata shows the file metadata in a formatted table
func displayMetadata(metadata *filewriter.Metadata) {
	// Parse device information to extract gain control settings
	deviceSettings := parseDeviceInfo(metadata.DeviceInfo)

	fmt.Printf("📊 Collection Metadata:\n")
	fmt.Printf("File Format Version: %d\n", metadata.FileFormatVersion)
	fmt.Printf("Collection ID: %s\n", metadata.CollectionID)
	fmt.Printf("Frequency: %.3f MHz\n", float64(metadata.Frequency)/1e6)
	fmt.Printf("Sample Rate: %.3f MSps\n", float64(metadata.SampleRate)/1e6)
	fmt.Printf("Collection Time: %s\n", metadata.CollectionTime.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("GPS Timestamp: %s\n", metadata.GPSTimestamp.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("GPS Latitude: %14.8f°\n", metadata.GPSLocation.Latitude)
	fmt.Printf("GPS Longitude: %14.8f°\n", metadata.GPSLocation.Longitude)
	fmt.Printf("GPS Altitude: %14.2f m\n\n", metadata.GPSLocation.Altitude)

	// Display device configuration prominently
	fmt.Printf("📻 Device Configuration:\n")
	fmt.Printf("Device Name: %s\n", deviceSettings.Name)
	fmt.Printf("Gain Setting: %s\n", deviceSettings.Gain)
	fmt.Printf("Gain Mode: %s\n", deviceSettings.GainMode)
	fmt.Printf("Bias Tee: %s\n\n", deviceSettings.BiasTee)
}

// displayDeviceAnalysis shows detailed analysis of device configuration
func displayDeviceAnalysis(deviceSettings DeviceSettings) {
	fmt.Printf("🔧 Device Configuration Analysis:\n")
	fmt.Printf("┌─────────────────────────┬─────────────────────────────────────────┐\n")
	fmt.Printf("│ Analysis                │ Information                             │\n")
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")

	// Analyze gain mode
	var gainAnalysis string
	switch deviceSettings.GainMode {
	case "auto":
		gainAnalysis = "AGC enabled - automatic gain adjustment"
	case "manual":
		gainAnalysis = "Manual gain control - fixed gain setting"
	default:
		gainAnalysis = "Unknown gain mode"
	}
	fmt.Printf("Gain Control: %s\n", gainAnalysis)

	// Analyze gain setting if available
	if deviceSettings.Gain != "Unknown" && deviceSettings.GainMode == "manual" {
		fmt.Printf("Gain Impact: Higher values increase sensitivity but may introduce noise\n")
	}

	// Analyze bias tee
	var biasAnalysis string
	switch deviceSettings.BiasTee {
	case "on":
		biasAnalysis = "Powering external LNA via antenna port"
	case "off":
		biasAnalysis = "No power supplied to antenna port"
	default:
		biasAnalysis = "Bias tee status unknown"
	}
	fmt.Printf("Bias Tee Status: %s\n", biasAnalysis)

	// Recommendations
	fmt.Printf("\nRecommendations:\n")
	if deviceSettings.GainMode == "auto" {
		fmt.Printf("• AGC may cause gain variations\n")
		fmt.Printf("• Consider manual gain for consistency\n")
	} else if deviceSettings.GainMode == "manual" {
		fmt.Printf("• Manual gain provides consistency\n")
		fmt.Printf("• Monitor for clipping or noise\n")
	}

	if deviceSettings.BiasTee == "on" {
		fmt.Printf("• Bias tee active - check LNA power\n")
	}

	fmt.Println()

	// Show typical RTL-SDR gain values for reference
	fmt.Printf("📊 RTL-SDR Gain Reference:\n")
	fmt.Printf("0.0 - 10.0 dB: Strong signals, prevent overload\n")
	fmt.Printf("10.0 - 30.0 dB: Medium signals, general purpose\n")
	fmt.Printf("30.0 - 50.0 dB: Weak signals, maximum sensitivity\n")
	fmt.Printf("AUTO (AGC): Automatic adjustment based on signal\n\n")
}

// displaySampleInfo shows information about the IQ samples
func displaySampleInfo(sampleCount int, sampleRate uint32) {
	duration := float64(sampleCount) / float64(sampleRate)

	fmt.Printf("📡 Sample Information:\n")
	fmt.Printf("Total Samples: %d\n", sampleCount)
	fmt.Printf("Sample Type: Complex64 (32-bit I + 32-bit Q)\n")
	fmt.Printf("Data Size: %.2f MB\n", float64(sampleCount*8)/(1024*1024))
	fmt.Printf("Collection Duration: %.3f seconds\n\n", duration)
}

// displaySamplesStreaming reads and displays samples as they're read from file
func displaySamplesStreaming(filename string, metadata *filewriter.Metadata, totalSamples int) error {
	fmt.Printf("📈 IQ Sample Data (streaming all %d samples):\n", totalSamples)
	fmt.Printf("%-8s %-14s %-14s %-14s %-12s\n", "#", "I (Real)", "Q (Imag)", "Magnitude", "Phase (°)")

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Seek to start of sample data
	headerSize := int64(5 + 2 + 8 + 4 + 12 + 24 + 12 + 1 + len(metadata.DeviceInfo) + 1 + len(metadata.CollectionID) + 4)
	_, err = file.Seek(headerSize, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to sample data: %w", err)
	}

	// Stream samples and display them
	const rad2deg = 180.0 / math.Pi
	const batchSize = 1000 // Process in batches for better performance
	var batch strings.Builder
	batch.Grow(batchSize * 80) // Estimate 80 chars per row
	
	index := 0
	for {
		var real, imag float32
		if err := binary.Read(file, binary.LittleEndian, &real); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read real part: %w", err)
		}
		if err := binary.Read(file, binary.LittleEndian, &imag); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read imaginary part: %w", err)
		}

		realPart := float64(real)
		imagPart := float64(imag)
		magnitude := math.Sqrt(realPart*realPart + imagPart*imagPart)
		phase := math.Atan2(imagPart, realPart) * rad2deg

		batch.WriteString(fmt.Sprintf("%-8d %-14.6f %-14.6f %-14.6f %-12.2f\n",
			index, realPart, imagPart, magnitude, phase))
		
		index++
		
		// Output batch when it's full
		if index%batchSize == 0 {
			fmt.Print(batch.String())
			batch.Reset()
		}
	}
	
	// Output remaining batch
	if batch.Len() > 0 {
		fmt.Print(batch.String())
	}

	fmt.Println()
	
	return nil
}

// displayHexStreaming reads and displays hex dump as samples are read from file
func displayHexStreaming(filename string, metadata *filewriter.Metadata, totalSamples int) error {
	totalBytes := totalSamples * 8
	fmt.Printf("🔍 Hex Dump of Raw Sample Data (streaming all %d bytes):\n", totalBytes)
	fmt.Printf("Each complex64 sample = 8 bytes (4-byte float I + 4-byte float Q)\n")
	fmt.Printf("%-9s %-48s %s\n", "Address", "00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F", "ASCII")

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Seek to start of sample data
	headerSize := int64(5 + 2 + 8 + 4 + 12 + 24 + 12 + 1 + len(metadata.DeviceInfo) + 1 + len(metadata.CollectionID) + 4)
	_, err = file.Seek(headerSize, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to sample data: %w", err)
	}

	// Read and display in 16-byte rows
	var buffer [16]byte
	offset := 0
	showInterpretation := true
	interpretCount := 0
	
	for {
		n, err := file.Read(buffer[:])
		if n == 0 {
			break
		}
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read data: %w", err)
		}

		// Build hex representation
		var hexPart strings.Builder
		var asciiPart strings.Builder
		
		for i := 0; i < 16; i++ {
			if i < n {
				b := buffer[i]
				hexPart.WriteString(fmt.Sprintf("%02x ", b))
				
				// ASCII representation
				if b >= 32 && b <= 126 {
					asciiPart.WriteByte(b)
				} else {
					asciiPart.WriteByte('.')
				}
			} else {
				hexPart.WriteString("   ")
				asciiPart.WriteByte(' ')
			}
		}

		fmt.Printf("%08x %-48s %s\n", offset, hexPart.String(), asciiPart.String())
		
		// Show sample interpretation for first few complete samples
		if showInterpretation && offset%8 == 0 && n >= 8 && interpretCount < 4 {
			// We have at least one complete sample, interpret it
			for i := 0; i+7 < n && interpretCount < 4; i += 8 {
				// Extract real and imaginary parts from bytes
				realBits := binary.LittleEndian.Uint32(buffer[i:i+4])
				imagBits := binary.LittleEndian.Uint32(buffer[i+4:i+8])
				
				realVal := math.Float32frombits(realBits)
				imagVal := math.Float32frombits(imagBits)
				
				sampleNum := (offset + i) / 8
				if interpretCount == 0 {
					fmt.Printf("\nSample Interpretation (first few samples):\n")
				}
				
				fmt.Printf("Sample %d: I=%f Q=%f | I bytes: %02x %02x %02x %02x | Q bytes: %02x %02x %02x %02x\n",
					sampleNum, realVal, imagVal,
					buffer[i], buffer[i+1], buffer[i+2], buffer[i+3],
					buffer[i+4], buffer[i+5], buffer[i+6], buffer[i+7])
				
				interpretCount++
			}
			
			if interpretCount >= 4 {
				showInterpretation = false
				fmt.Println()
			}
		}
		
		offset += n
		
		if err == io.EOF {
			break
		}
	}
	
	fmt.Println()
	return nil
}

// displayGraph creates an ASCII graph of signal magnitude over time
func displayGraph(samples []complex64, sampleRate uint32) {
	if len(samples) == 0 {
		fmt.Printf("📈 Signal Graph: No samples to display\n\n")
		return
	}

	// Calculate magnitudes
	magnitudes := make([]float64, len(samples))
	minMag, maxMag := math.Inf(1), math.Inf(-1)

	for i, sample := range samples {
		mag := math.Sqrt(float64(real(sample)*real(sample) + imag(sample)*imag(sample)))
		magnitudes[i] = mag
		if mag < minMag {
			minMag = mag
		}
		if mag > maxMag {
			maxMag = mag
		}
	}

	// Handle edge case where all magnitudes are the same
	if maxMag == minMag {
		maxMag = minMag + 1e-6
	}

	// Calculate time span
	totalTime := float64(len(samples)) / float64(sampleRate)

	fmt.Printf("📈 Signal Magnitude Over Time:\n")
	fmt.Printf("Samples: %d | Duration: %.3f seconds | Sample Rate: %.3f MSps\n",
		len(samples), totalTime, float64(sampleRate)/1e6)
	fmt.Printf("Magnitude Range: %.6f to %.6f\n", minMag, maxMag)
	fmt.Println()

	// Create graph grid
	graph := make([][]rune, graphHeight)
	for i := range graph {
		graph[i] = make([]rune, graphWidth)
		for j := range graph[i] {
			graph[i][j] = ' '
		}
	}

	// Plot data points
	for i, mag := range magnitudes {
		// Map sample index to x position
		x := int(float64(i) * float64(graphWidth-1) / float64(len(magnitudes)-1))
		if x >= graphWidth {
			x = graphWidth - 1
		}

		// Map magnitude to y position (inverted because we draw top to bottom)
		normalizedMag := (mag - minMag) / (maxMag - minMag)
		y := int(float64(graphHeight-1) * (1.0 - normalizedMag))
		if y >= graphHeight {
			y = graphHeight - 1
		}
		if y < 0 {
			y = 0
		}

		// Plot the point
		if graph[y][x] == ' ' {
			graph[y][x] = '*'
		} else {
			graph[y][x] = '#' // Multiple points at same location
		}
	}

	// Display the graph with y-axis labels
	fmt.Printf("Magnitude\n")
	for i, row := range graph {
		// Calculate the magnitude value for this row
		normalizedY := float64(graphHeight-1-i) / float64(graphHeight-1)
		magValue := minMag + normalizedY*(maxMag-minMag)

		// Print y-axis label and graph row
		fmt.Printf("%8.4f |", magValue)
		for _, char := range row {
			fmt.Print(string(char))
		}
		fmt.Println("|")
	}

	// Print x-axis
	fmt.Printf("         +")
	fmt.Print(strings.Repeat("-", graphWidth))
	fmt.Println("+")

	// Print time labels
	fmt.Printf("         0")
	midTime := totalTime / 2
	endTime := totalTime

	// Calculate spacing for time labels
	midPos := graphWidth / 2
	endPos := graphWidth

	// Print middle time label
	midLabel := fmt.Sprintf("%.3fs", midTime)
	fmt.Print(strings.Repeat(" ", midPos-len(midLabel)/2))
	fmt.Print(midLabel)

	// Print end time label
	endLabel := fmt.Sprintf("%.3fs", endTime)
	fmt.Print(strings.Repeat(" ", endPos-midPos-len(endLabel)))
	fmt.Print(endLabel)
	fmt.Println()

	fmt.Printf("\nLegend: * = data point, # = multiple points, Time →\n\n")

	// Additional analysis
	fmt.Printf("📊 Signal Analysis:\n")
	avgMag := 0.0
	for _, mag := range magnitudes {
		avgMag += mag
	}
	avgMag /= float64(len(magnitudes))

	fmt.Printf("   Average Magnitude: %.6f\n", avgMag)
	fmt.Printf("   Peak Magnitude: %.6f\n", maxMag)
	fmt.Printf("   Dynamic Range: %.2f dB\n", 20*math.Log10(maxMag/minMag))
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
	var magnitudes []float64

	for _, sample := range samples {
		i := float64(real(sample))
		q := float64(imag(sample))
		mag := math.Sqrt(i*i + q*q)
		power := i*i + q*q

		sumI += i
		sumQ += q
		sumMag += mag
		sumPower += power
		magnitudes = append(magnitudes, mag)

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

	// Calculate signal strength metrics
	rmsAmplitude := math.Sqrt(meanPower)
	signalStrengthDbm := 10*math.Log10(meanPower) - 30 // Convert to dBm (assuming 50-ohm impedance)
	
	// Calculate noise floor estimation using lowest 10% of magnitude samples
	// Sort magnitudes to find noise floor
	sortedMags := make([]float64, len(magnitudes))
	copy(sortedMags, magnitudes)
	
	// Simple bubble sort for the bottom 10%
	n := len(sortedMags)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if sortedMags[j] > sortedMags[j+1] {
				sortedMags[j], sortedMags[j+1] = sortedMags[j+1], sortedMags[j]
			}
		}
	}
	
	// Use bottom 10% for noise floor estimation
	noiseFloorSamples := int(float64(len(sortedMags)) * 0.1)
	if noiseFloorSamples < 10 {
		noiseFloorSamples = 10
	}
	if noiseFloorSamples > len(sortedMags) {
		noiseFloorSamples = len(sortedMags)
	}
	
	var noiseFloorSum float64
	for i := 0; i < noiseFloorSamples; i++ {
		noiseFloorSum += sortedMags[i] * sortedMags[i] // Convert to power
	}
	noiseFloorPower := noiseFloorSum / float64(noiseFloorSamples)
	noiseFloorDb := 10 * math.Log10(noiseFloorPower)
	
	// Calculate Signal-to-Noise Ratio
	signalPowerDb := 10 * math.Log10(meanPower)
	snrDb := signalPowerDb - noiseFloorDb

	fmt.Printf("📊 Statistical Analysis:\n")
	fmt.Printf("Mean I (Real): %12.6f\n", meanI)
	fmt.Printf("Mean Q (Imaginary): %12.6f\n", meanQ)
	fmt.Printf("I Variance: %12.6f\n", varI)
	fmt.Printf("Q Variance: %12.6f\n", varQ)
	fmt.Printf("Mean Magnitude: %12.6f\n", meanMag)
	fmt.Printf("Min Magnitude: %12.6f\n", minMag)
	fmt.Printf("Max Magnitude: %12.6f\n", maxMag)
	fmt.Printf("RMS Amplitude: %12.6f\n", rmsAmplitude)
	fmt.Printf("Mean Power: %12.6f\n", meanPower)
	fmt.Printf("Signal Power (dB): %12.2f dB\n", signalPowerDb)
	fmt.Printf("Signal Strength (dBm): %12.2f dBm\n", signalStrengthDbm)
	fmt.Printf("Noise Floor (dB): %12.2f dB\n", noiseFloorDb)
	fmt.Printf("Signal-to-Noise Ratio: %12.2f dB\n\n", snrDb)
}

// main is the entry point of the application
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
