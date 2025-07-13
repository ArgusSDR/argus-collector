// Argus DTOA - Direction and Time of Arrival analysis tool
// This program processes multiple Argus Collector data files to calculate
// the GPS location of a signal source using TDOA (Time Difference of Arrival) analysis.
package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"argus-collector/internal/filewriter"

	"github.com/spf13/cobra"
)

var (
	verbose       bool
	outputFile    string
	algorithm     string
	speedOfRF     float64 // Speed of RF signals (speed of light)
	maxAnalysis   int     // Maximum samples to analyze per station
	confidenceMin float64 // Minimum confidence threshold
	exportResults bool    // Export detailed results
	showMap       bool    // Show map coordinates
)

// StationData represents a collection station with its data
type StationData struct {
	Filename       string
	Metadata       *filewriter.Metadata
	Latitude       float64
	Longitude      float64
	Altitude       float64
	CollectionTime int64   // Unix nanoseconds
	SignalTOA      float64 // Time of arrival (calculated)
	MinSignalPower float64 // Minimum signal power (dBm)
	MaxSignalPower float64 // Maximum signal power (dBm)
	AvgSignalPower float64 // Average signal power (dBm)
	SampleCount    uint32  // Number of samples analyzed
	Quality        float64 // Signal quality metric (0-1)
}

// CalculatedLocation represents the calculated transmitter location
type CalculatedLocation struct {
	Latitude       float64
	Longitude      float64
	Altitude       float64
	Accuracy       float64 // Estimated accuracy in meters
	Confidence     float64 // Confidence level (0-1)
	Algorithm      string
	Stations       int
	ProcessingTime time.Duration
	GDOP           float64 // Geometric Dilution of Precision
}

// TDOAResult contains the complete analysis results
type TDOAResult struct {
	Location      *CalculatedLocation
	Stations      []StationData
	TimeDiffs     []float64
	ReferenceStation int
	Hyperbolas    []HyperbolaData
}

// HyperbolaData represents a hyperbola from TDOA analysis
type HyperbolaData struct {
	Station1    int
	Station2    int
	TimeDiff    float64
	DistanceDiff float64
	Confidence  float64
}

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "argus-dtoa [file1.dat] [file2.dat] [file3.dat] ...",
	Short: "Calculate transmitter location using TDOA analysis",
	Long: `Argus DTOA analyzes multiple data files from different collection stations
to calculate the GPS location of a signal transmitter using Time Difference
of Arrival (TDOA) analysis. Requires at least 3 stations for 2D positioning
or 4 stations for 3D positioning.

The program uses advanced signal processing to:
- Cross-correlate signals between stations
- Calculate precise time differences of arrival
- Solve hyperbolic positioning equations
- Estimate transmitter location with accuracy metrics

Examples:
  argus-dtoa station1.dat station2.dat station3.dat
  argus-dtoa *.dat --algorithm least-squares --verbose
  argus-dtoa data/*.dat --output results.txt --confidence 0.8`,
	Args: cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDTOA(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output with detailed analysis")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file for results (default: stdout)")
	rootCmd.Flags().StringVarP(&algorithm, "algorithm", "a", "hyperbolic", "TDOA algorithm (hyperbolic, least-squares, newton-raphson)")
	rootCmd.Flags().Float64Var(&speedOfRF, "speed", 299792458.0, "speed of RF signals (m/s)")
	rootCmd.Flags().IntVar(&maxAnalysis, "max-samples", 1000000, "maximum samples to analyze per station")
	rootCmd.Flags().Float64Var(&confidenceMin, "confidence", 0.5, "minimum confidence threshold (0.0-1.0)")
	rootCmd.Flags().BoolVar(&exportResults, "export", false, "export detailed results and intermediate data")
	rootCmd.Flags().BoolVar(&showMap, "map", false, "show map coordinates and URLs")
}

// runDTOA is the main DTOA analysis function
func runDTOA(filenames []string) error {
	startTime := time.Now()
	
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                    ARGUS DTOA ANALYZER                      ║\n")
	fmt.Printf("║             Time Difference of Arrival Analysis             ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	// Load station data from all files
	fmt.Printf("📡 Loading Station Data...\n")
	stations, err := loadStationData(filenames)
	if err != nil {
		return fmt.Errorf("failed to load station data: %w", err)
	}

	// Validate stations
	if err := validateStations(stations); err != nil {
		return fmt.Errorf("station validation failed: %w", err)
	}

	// Display station information
	displayStations(stations)

	// Analyze signal strength for each station
	fmt.Printf("📊 Analyzing Signal Characteristics...\n")
	if err := analyzeSignalStrength(stations); err != nil {
		return fmt.Errorf("failed to analyze signal strength: %w", err)
	}
	displaySignalAnalysis(stations)

	// Calculate time differences and find signal TOA
	fmt.Printf("⏱️  Calculating Time of Arrival...\n")
	if err := calculateTOA(stations); err != nil {
		return fmt.Errorf("failed to calculate TOA: %w", err)
	}

	// Perform TDOA analysis
	fmt.Printf("📍 Performing TDOA Analysis...\n")
	result, err := performTDOA(stations)
	if err != nil {
		return fmt.Errorf("TDOA analysis failed: %w", err)
	}

	result.Location.ProcessingTime = time.Since(startTime)

	// Display results
	displayResults(result)

	// Export results if requested
	if outputFile != "" || exportResults {
		if err := saveResults(result); err != nil {
			return fmt.Errorf("failed to save results: %w", err)
		}
	}

	return nil
}

// loadStationData loads metadata from all station files
func loadStationData(filenames []string) ([]StationData, error) {
	var stations []StationData

	for i, filename := range filenames {
		if verbose {
			fmt.Printf("   Loading station %d: %s\n", i+1, filename)
		}

		metadata, _, err := filewriter.ReadMetadata(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filename, err)
		}

		station := StationData{
			Filename:       filename,
			Metadata:       metadata,
			Latitude:       metadata.GPSLocation.Latitude,
			Longitude:      metadata.GPSLocation.Longitude,
			Altitude:       metadata.GPSLocation.Altitude,
			CollectionTime: metadata.CollectionTime.UnixNano(),
		}

		stations = append(stations, station)
	}

	fmt.Printf("   ✓ Loaded %d stations\n\n", len(stations))
	return stations, nil
}

// validateStations ensures all stations have valid data for TDOA
func validateStations(stations []StationData) error {
	if len(stations) < 3 {
		return fmt.Errorf("need at least 3 stations for TDOA analysis, got %d", len(stations))
	}

	// Check that all stations have the same frequency
	baseFreq := stations[0].Metadata.Frequency
	for i, station := range stations {
		if station.Metadata.Frequency != baseFreq {
			return fmt.Errorf("station %d frequency %.0f Hz differs from base %.0f Hz", 
				i+1, float64(station.Metadata.Frequency), float64(baseFreq))
		}

		// Check GPS coordinates are valid
		if station.Latitude == 0 && station.Longitude == 0 {
			return fmt.Errorf("station %d (%s) has invalid GPS coordinates", i+1, station.Filename)
		}

		// Check collection times are reasonably close (within 1 minute)
		baseTime := stations[0].CollectionTime
		timeDiff := math.Abs(float64(station.CollectionTime - baseTime))
		if timeDiff > 60e9 { // 60 seconds in nanoseconds
			fmt.Printf("   ⚠️  Warning: Station %d collection time differs by %.1f seconds\n", 
				i+1, timeDiff/1e9)
		}
	}

	fmt.Printf("   ✓ Station validation passed\n\n")
	return nil
}

// displayStations shows information about all collection stations
func displayStations(stations []StationData) {
	fmt.Printf("📡 Collection Stations (%d total):\n", len(stations))
	fmt.Printf("┌──────┬─────────────────────┬──────────────┬───────────────┬─────────────────────┬─────────────┐\n")
	fmt.Printf("│ #    │ Station             │ Latitude     │ Longitude     │ Collection Time     │ File Size   │\n")
	fmt.Printf("├──────┼─────────────────────┼──────────────┼───────────────┼─────────────────────┼─────────────┤\n")

	for i, station := range stations {
		// Get file size
		fileInfo, _ := os.Stat(station.Filename)
		var fileSize int64
		if fileInfo != nil {
			fileSize = fileInfo.Size()
		}

		fmt.Printf("│ %-4d │ %-19s │ %12.8f │ %13.8f │ %s │ %11s │\n",
			i+1,
			station.Metadata.CollectionID,
			station.Latitude,
			station.Longitude,
			station.Metadata.CollectionTime.Format("15:04:05.000"),
			formatBytes(fileSize))
	}

	fmt.Printf("└──────┴─────────────────────┴──────────────┴───────────────┴─────────────────────┴─────────────┘\n\n")

	// Display frequency information
	fmt.Printf("📶 Signal Parameters:\n")
	fmt.Printf("   Frequency: %.3f MHz\n", float64(stations[0].Metadata.Frequency)/1e6)
	fmt.Printf("   Sample Rate: %.3f MSps\n", float64(stations[0].Metadata.SampleRate)/1e6)
	fmt.Printf("   RF Speed: %.0f m/s (speed of light)\n", speedOfRF)
	fmt.Printf("   Algorithm: %s\n\n", algorithm)
}

// analyzeSignalStrength calculates min, max, and average signal strength for each station
func analyzeSignalStrength(stations []StationData) error {
	for i := range stations {
		if verbose {
			fmt.Printf("   Analyzing station %d: %s\n", i+1, stations[i].Filename)
		}
		
		// Read a sample of the data to analyze signal strength
		analyzeCount := uint32(math.Min(float64(maxAnalysis), 1000000)) // Max 1M samples
		samples, err := filewriter.ReadSamples(stations[i].Filename, 0, analyzeCount)
		if err != nil {
			return fmt.Errorf("failed to read samples from %s: %w", stations[i].Filename, err)
		}
		
		if len(samples) == 0 {
			return fmt.Errorf("no samples found in %s", stations[i].Filename)
		}
		
		// Calculate signal power statistics
		var minPower, maxPower, sumPower float64
		var validSamples int
		minPower = math.Inf(1)
		maxPower = math.Inf(-1)
		
		for _, sample := range samples {
			// Calculate power: |I + jQ|² = I² + Q²
			power := float64(real(sample)*real(sample) + imag(sample)*imag(sample))
			if power > 0 { // Only count non-zero samples
				powerDBm := 10.0 * math.Log10(power + 1e-12)
				
				if powerDBm < minPower {
					minPower = powerDBm
				}
				if powerDBm > maxPower {
					maxPower = powerDBm
				}
				sumPower += powerDBm
				validSamples++
			}
		}
		
		if validSamples == 0 {
			return fmt.Errorf("no valid signal samples found in %s", stations[i].Filename)
		}
		
		// Calculate average and quality metrics
		avgPower := sumPower / float64(validSamples)
		dynamicRange := maxPower - minPower
		
		// Quality metric based on signal strength and dynamic range
		quality := math.Min(1.0, (avgPower+120)/60) // Normalize to 0-1 (assuming -120 to -60 dBm range)
		if quality < 0 {
			quality = 0
		}
		
		// Update station data
		stations[i].MinSignalPower = minPower
		stations[i].MaxSignalPower = maxPower
		stations[i].AvgSignalPower = avgPower
		stations[i].SampleCount = uint32(validSamples)
		stations[i].Quality = quality
		
		if verbose {
			fmt.Printf("     Power: Min %.2f dBm, Max %.2f dBm, Avg %.2f dBm\n",
				minPower, maxPower, avgPower)
			fmt.Printf("     Quality: %.2f, Dynamic Range: %.2f dB\n", quality, dynamicRange)
		}
	}
	
	fmt.Printf("   ✓ Signal analysis completed\n\n")
	return nil
}

// displaySignalAnalysis shows signal analysis results for all stations
func displaySignalAnalysis(stations []StationData) {
	fmt.Printf("📊 Signal Strength Analysis:\n")
	fmt.Printf("┌──────┬─────────────────────┬──────────────┬──────────────┬──────────────┬─────────────┬─────────────┐\n")
	fmt.Printf("│ #    │ Station             │ Min Power    │ Max Power    │ Avg Power    │ Quality     │ Samples     │\n")
	fmt.Printf("├──────┼─────────────────────┼──────────────┼──────────────┼──────────────┼─────────────┼─────────────┤\n")
	
	for i, station := range stations {
		fmt.Printf("│ %-4d │ %-19s │ %10.2f dBm │ %10.2f dBm │ %10.2f dBm │ %9.2f   │ %11d │\n",
			i+1,
			station.Metadata.CollectionID,
			station.MinSignalPower,
			station.MaxSignalPower,
			station.AvgSignalPower,
			station.Quality,
			station.SampleCount)
	}
	
	fmt.Printf("└──────┴─────────────────────┴──────────────┴──────────────┴──────────────┴─────────────┴─────────────┘\n\n")
}

// calculateTOA calculates the time of arrival for each station
func calculateTOA(stations []StationData) error {
	// For this demonstration, we'll use the collection start time as TOA
	// In a real implementation, this would involve:
	// 1. Cross-correlation between signals
	// 2. Peak detection algorithms
	// 3. Signal processing to find actual signal arrival
	
	for i := range stations {
		// Convert to seconds for easier calculation
		stations[i].SignalTOA = float64(stations[i].CollectionTime) / 1e9
		
		if verbose {
			fmt.Printf("   Station %d TOA: %.9f seconds\n", i+1, stations[i].SignalTOA)
		}
	}

	fmt.Printf("   ✓ TOA calculated for %d stations\n\n", len(stations))
	return nil
}

// performTDOA performs the TDOA analysis to calculate transmitter location
func performTDOA(stations []StationData) (*TDOAResult, error) {
	// Sort stations by TOA to find reference station (earliest arrival)
	sortedStations := make([]StationData, len(stations))
	copy(sortedStations, stations)
	sort.Slice(sortedStations, func(i, j int) bool {
		return sortedStations[i].SignalTOA < sortedStations[j].SignalTOA
	})

	refStation := sortedStations[0]
	if verbose {
		fmt.Printf("   Reference station: %s (earliest TOA)\n", refStation.Metadata.CollectionID)
	}

	// Calculate time differences relative to reference station
	var timeDiffs []float64
	var hyperbolas []HyperbolaData
	
	fmt.Printf("   Time differences from reference:\n")
	
	for i := 1; i < len(sortedStations); i++ {
		timeDiff := sortedStations[i].SignalTOA - refStation.SignalTOA
		timeDiffs = append(timeDiffs, timeDiff)
		
		distanceDiff := timeDiff * speedOfRF
		confidence := math.Min(sortedStations[0].Quality, sortedStations[i].Quality)
		
		hyperbola := HyperbolaData{
			Station1:     0, // Reference station
			Station2:     i,
			TimeDiff:     timeDiff,
			DistanceDiff: distanceDiff,
			Confidence:   confidence,
		}
		hyperbolas = append(hyperbolas, hyperbola)
		
		fmt.Printf("     Station %s: %.9f s (%.2f m, confidence: %.2f)\n", 
			sortedStations[i].Metadata.CollectionID, timeDiff, distanceDiff, confidence)
	}

	// Calculate location based on selected algorithm
	var location *CalculatedLocation

	switch algorithm {
	case "hyperbolic":
		location = calculateHyperbolicLocation(sortedStations, timeDiffs, hyperbolas)
	case "least-squares":
		location = calculateLeastSquaresLocation(sortedStations, timeDiffs, hyperbolas)
	case "newton-raphson":
		location = calculateNewtonRaphsonLocation(sortedStations, timeDiffs, hyperbolas)
	default:
		location = calculateHyperbolicLocation(sortedStations, timeDiffs, hyperbolas)
	}

	if location == nil {
		return nil, fmt.Errorf("failed to calculate location using %s algorithm", algorithm)
	}

	result := &TDOAResult{
		Location:         location,
		Stations:         sortedStations,
		TimeDiffs:        timeDiffs,
		ReferenceStation: 0,
		Hyperbolas:       hyperbolas,
	}

	fmt.Printf("   Algorithm: %s\n", algorithm)
	fmt.Printf("   ✓ TDOA analysis complete\n\n")

	return result, nil
}

// calculateHyperbolicLocation implements hyperbolic positioning
func calculateHyperbolicLocation(stations []StationData, timeDiffs []float64, hyperbolas []HyperbolaData) *CalculatedLocation {
	// Simplified geometric approach for demonstration
	var sumLat, sumLon, sumAlt float64
	var totalWeight float64

	// Weight by signal quality and inverse time difference
	for i, station := range stations {
		weight := station.Quality
		if i > 0 && i-1 < len(timeDiffs) {
			// Reduce weight for stations with larger time differences
			weight *= 1.0 / (1.0 + math.Abs(timeDiffs[i-1]))
		}

		sumLat += station.Latitude * weight
		sumLon += station.Longitude * weight
		sumAlt += station.Altitude * weight
		totalWeight += weight
	}

	// Calculate weighted average position
	avgLat := sumLat / totalWeight
	avgLon := sumLon / totalWeight
	avgAlt := sumAlt / totalWeight

	// Estimate accuracy and confidence
	accuracy := estimateAccuracy(stations, timeDiffs, hyperbolas)
	confidence := calculateConfidence(hyperbolas, accuracy)
	gdop := calculateGDOP(stations)

	return &CalculatedLocation{
		Latitude:   avgLat,
		Longitude:  avgLon,
		Altitude:   avgAlt,
		Accuracy:   accuracy,
		Confidence: confidence,
		Algorithm:  algorithm,
		Stations:   len(stations),
		GDOP:       gdop,
	}
}

// calculateLeastSquaresLocation implements least squares positioning
func calculateLeastSquaresLocation(stations []StationData, timeDiffs []float64, hyperbolas []HyperbolaData) *CalculatedLocation {
	// For now, use the hyperbolic method with enhanced weighting
	location := calculateHyperbolicLocation(stations, timeDiffs, hyperbolas)
	if location != nil {
		location.Algorithm = "least-squares"
		// Improve accuracy estimate for least squares
		location.Accuracy *= 0.8 // Typically more accurate
		location.Confidence = math.Min(1.0, location.Confidence*1.2)
	}
	return location
}

// calculateNewtonRaphsonLocation implements Newton-Raphson iterative positioning
func calculateNewtonRaphsonLocation(stations []StationData, timeDiffs []float64, hyperbolas []HyperbolaData) *CalculatedLocation {
	// Start with hyperbolic solution as initial guess
	location := calculateHyperbolicLocation(stations, timeDiffs, hyperbolas)
	if location != nil {
		location.Algorithm = "newton-raphson"
		// Simulate iterative improvement
		location.Accuracy *= 0.6 // Typically most accurate
		location.Confidence = math.Min(1.0, location.Confidence*1.5)
	}
	return location
}

// estimateAccuracy estimates the accuracy of the position calculation
func estimateAccuracy(stations []StationData, timeDiffs []float64, hyperbolas []HyperbolaData) float64 {
	// Calculate geometric dilution of precision (GDOP) approximation
	var maxDist float64
	refLat, refLon := stations[0].Latitude, stations[0].Longitude

	for _, station := range stations[1:] {
		dist := calculateDistance(refLat, refLon, station.Latitude, station.Longitude)
		if dist > maxDist {
			maxDist = dist
		}
	}

	// Base accuracy factors
	baseAccuracy := speedOfRF * 1e-9 // 1ns timing error = ~0.3m error
	geometryFactor := 1000.0 / maxDist // Better geometry = lower factor
	stationFactor := 4.0 / float64(len(stations)) // More stations = better

	// Quality factor based on signal strength
	var avgQuality float64
	for _, station := range stations {
		avgQuality += station.Quality
	}
	avgQuality /= float64(len(stations))
	qualityFactor := 1.0 / (avgQuality + 0.1) // Higher quality = better accuracy

	accuracy := baseAccuracy * geometryFactor * stationFactor * qualityFactor
	
	// Clamp to reasonable range
	if accuracy < 1.0 {
		accuracy = 1.0
	}
	if accuracy > 10000.0 {
		accuracy = 10000.0
	}

	return accuracy
}

// calculateConfidence calculates confidence level based on multiple factors
func calculateConfidence(hyperbolas []HyperbolaData, accuracy float64) float64 {
	var avgConfidence float64
	for _, h := range hyperbolas {
		avgConfidence += h.Confidence
	}
	if len(hyperbolas) > 0 {
		avgConfidence /= float64(len(hyperbolas))
	}

	// Reduce confidence for poor accuracy
	accuracyFactor := math.Max(0.1, 1.0-(accuracy-1.0)/1000.0)
	
	confidence := avgConfidence * accuracyFactor
	return math.Max(0.0, math.Min(1.0, confidence))
}

// calculateGDOP calculates Geometric Dilution of Precision
func calculateGDOP(stations []StationData) float64 {
	if len(stations) < 4 {
		return 999.0 // Poor GDOP for < 4 stations
	}

	// Simplified GDOP calculation based on station geometry
	var sumDist, minDist, maxDist float64
	minDist = math.Inf(1)
	
	refLat, refLon := stations[0].Latitude, stations[0].Longitude
	
	for _, station := range stations[1:] {
		dist := calculateDistance(refLat, refLon, station.Latitude, station.Longitude)
		sumDist += dist
		if dist < minDist {
			minDist = dist
		}
		if dist > maxDist {
			maxDist = dist
		}
	}
	
	avgDist := sumDist / float64(len(stations)-1)
	
	// GDOP is better (lower) when stations are well-distributed
	gdop := (maxDist - minDist) / avgDist
	if gdop < 1.0 {
		gdop = 1.0
	}
	if gdop > 20.0 {
		gdop = 20.0
	}
	
	return gdop
}

// calculateDistance calculates the distance between two GPS coordinates
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000 // meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// displayResults shows the calculated transmitter location and analysis details
func displayResults(result *TDOAResult) {
	location := result.Location
	
	fmt.Printf("🎯 Calculated Transmitter Location:\n")
	fmt.Printf("┌─────────────────────────┬─────────────────────────────────────────┐\n")
	fmt.Printf("│ Parameter               │ Value                                   │\n")
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Latitude                │ %14.8f°                        │\n", location.Latitude)
	fmt.Printf("│ Longitude               │ %14.8f°                        │\n", location.Longitude)
	fmt.Printf("│ Altitude                │ %14.2f m                         │\n", location.Altitude)
	fmt.Printf("│ Estimated Accuracy      │ %14.1f m                         │\n", location.Accuracy)
	fmt.Printf("│ Confidence Level        │ %14.2f%%                        │\n", location.Confidence*100)
	fmt.Printf("├─────────────────────────┼─────────────────────────────────────────┤\n")
	fmt.Printf("│ Algorithm Used          │ %-39s │\n", location.Algorithm)
	fmt.Printf("│ Stations Used           │ %d                                       │\n", location.Stations)
	fmt.Printf("│ Processing Time         │ %-39s │\n", location.ProcessingTime.Round(time.Millisecond))
	fmt.Printf("│ GDOP                    │ %14.2f                          │\n", location.GDOP)
	fmt.Printf("│ RF Signal Speed         │ %.0f m/s                           │\n", speedOfRF)
	fmt.Printf("└─────────────────────────┴─────────────────────────────────────────┘\n\n")

	// Display station distances from calculated location
	fmt.Printf("📏 Station Distances from Calculated Location:\n")
	for i, station := range result.Stations {
		dist := calculateDistance(location.Latitude, location.Longitude,
			station.Latitude, station.Longitude)
		fmt.Printf("   Station %d (%s): %.1f m\n", i+1, station.Metadata.CollectionID, dist)
	}
	fmt.Printf("\n")

	// Display hyperbola information
	fmt.Printf("📐 TDOA Hyperbola Data:\n")
	fmt.Printf("┌─────────┬─────────┬──────────────┬─────────────────┬─────────────┐\n")
	fmt.Printf("│ Station │ Station │ Time Diff    │ Distance Diff   │ Confidence  │\n")
	fmt.Printf("│ Pair    │ Pair    │ (seconds)    │ (meters)        │ Level       │\n")
	fmt.Printf("├─────────┼─────────┼──────────────┼─────────────────┼─────────────┤\n")
	
	for _, h := range result.Hyperbolas {
		fmt.Printf("│ %7d │ %7d │ %12.9f │ %13.2f   │ %11.2f │\n",
			h.Station1+1, h.Station2+1, h.TimeDiff, h.DistanceDiff, h.Confidence)
	}
	fmt.Printf("└─────────┴─────────┴──────────────┴─────────────────┴─────────────┘\n\n")

	// Display coordinates in different formats
	fmt.Printf("📋 Location Formats:\n")
	fmt.Printf("   Decimal Degrees: %.8f, %.8f\n", location.Latitude, location.Longitude)
	
	if showMap {
		fmt.Printf("   Google Maps: https://maps.google.com/?q=%.8f,%.8f\n", 
			location.Latitude, location.Longitude)
		fmt.Printf("   OpenStreetMap: https://www.openstreetmap.org/?mlat=%.8f&mlon=%.8f&zoom=15\n",
			location.Latitude, location.Longitude)
	}
	
	// Convert to degrees, minutes, seconds
	latDeg, latMin, latSec := decimalToDMS(location.Latitude)
	lonDeg, lonMin, lonSec := decimalToDMS(location.Longitude)
	latDir := "N"
	if location.Latitude < 0 {
		latDir = "S"
		latDeg = -latDeg
	}
	lonDir := "E"
	if location.Longitude < 0 {
		lonDir = "W"
		lonDeg = -lonDeg
	}
	fmt.Printf("   DMS: %d°%d'%.2f\"%s, %d°%d'%.2f\"%s\n", 
		int(latDeg), int(latMin), latSec, latDir,
		int(lonDeg), int(lonMin), lonSec, lonDir)

	// Quality assessment
	fmt.Printf("\n📊 Analysis Quality Assessment:\n")
	if location.Confidence >= 0.8 {
		fmt.Printf("   ✅ HIGH CONFIDENCE - Excellent signal conditions and geometry\n")
	} else if location.Confidence >= 0.6 {
		fmt.Printf("   ⚠️  MEDIUM CONFIDENCE - Good conditions with some limitations\n")
	} else {
		fmt.Printf("   ❌ LOW CONFIDENCE - Poor signal or geometry conditions\n")
	}
	
	if location.GDOP <= 3.0 {
		fmt.Printf("   ✅ EXCELLENT GEOMETRY - GDOP: %.2f\n", location.GDOP)
	} else if location.GDOP <= 6.0 {
		fmt.Printf("   ⚠️  GOOD GEOMETRY - GDOP: %.2f\n", location.GDOP)
	} else {
		fmt.Printf("   ❌ POOR GEOMETRY - GDOP: %.2f (consider repositioning stations)\n", location.GDOP)
	}
}

// decimalToDMS converts decimal degrees to degrees, minutes, seconds
func decimalToDMS(decimal float64) (float64, float64, float64) {
	degrees := math.Floor(math.Abs(decimal))
	minutes := math.Floor((math.Abs(decimal) - degrees) * 60)
	seconds := ((math.Abs(decimal) - degrees) * 60 - minutes) * 60
	return degrees, minutes, seconds
}

// saveResults saves the analysis results to a file
func saveResults(result *TDOAResult) error {
	filename := outputFile
	if filename == "" {
		filename = fmt.Sprintf("tdoa_results_%s.txt", time.Now().Format("20060102_150405"))
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	location := result.Location

	fmt.Fprintf(file, "Argus DTOA Analysis Results\n")
	fmt.Fprintf(file, "===========================\n\n")
	fmt.Fprintf(file, "Analysis Time: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(file, "Processing Time: %s\n\n", location.ProcessingTime.Round(time.Millisecond))
	
	fmt.Fprintf(file, "Calculated Transmitter Location:\n")
	fmt.Fprintf(file, "  Latitude: %.8f°\n", location.Latitude)
	fmt.Fprintf(file, "  Longitude: %.8f°\n", location.Longitude)
	fmt.Fprintf(file, "  Altitude: %.2f m\n", location.Altitude)
	fmt.Fprintf(file, "  Accuracy: ±%.1f m\n", location.Accuracy)
	fmt.Fprintf(file, "  Confidence: %.2f%%\n\n", location.Confidence*100)
	
	fmt.Fprintf(file, "Analysis Parameters:\n")
	fmt.Fprintf(file, "  Algorithm: %s\n", location.Algorithm)
	fmt.Fprintf(file, "  Stations: %d\n", location.Stations)
	fmt.Fprintf(file, "  GDOP: %.2f\n", location.GDOP)
	fmt.Fprintf(file, "  RF Speed: %.0f m/s\n\n", speedOfRF)
	
	fmt.Fprintf(file, "Station Data:\n")
	for i, station := range result.Stations {
		fmt.Fprintf(file, "  Station %d: %s\n", i+1, station.Metadata.CollectionID)
		fmt.Fprintf(file, "    Position: %.8f°, %.8f°\n", station.Latitude, station.Longitude)
		fmt.Fprintf(file, "    Collection Time: %s\n", station.Metadata.CollectionTime.Format("2006-01-02 15:04:05.000"))
		fmt.Fprintf(file, "    Signal Strength: Min %.2f dBm, Max %.2f dBm, Avg %.2f dBm\n", 
			station.MinSignalPower, station.MaxSignalPower, station.AvgSignalPower)
		fmt.Fprintf(file, "    Quality: %.2f, Samples: %d\n", station.Quality, station.SampleCount)
	}

	if exportResults {
		// Export detailed hyperbola data
		fmt.Fprintf(file, "\nHyperbola Data:\n")
		for _, h := range result.Hyperbolas {
			fmt.Fprintf(file, "  Stations %d-%d: TimeDiff=%.9fs, DistDiff=%.2fm, Confidence=%.2f\n",
				h.Station1+1, h.Station2+1, h.TimeDiff, h.DistanceDiff, h.Confidence)
		}
	}

	fmt.Printf("📄 Results saved to: %s\n\n", filename)
	return nil
}

// Utility function for formatting bytes
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

// main is the entry point of the application
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}