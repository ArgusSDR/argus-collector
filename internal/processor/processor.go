// Package processor implements TDOA signal processing for transmitter localization
package processor

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"argus-collector/internal/filewriter"
)

// Config holds the configuration for TDOA processing
type Config struct {
	Algorithm      string   // TDOA algorithm to use
	Confidence     float64  // Minimum confidence threshold
	MaxDistance    float64  // Maximum expected transmitter distance (km)  
	FrequencyRange []string // Frequency ranges to analyze
	Verbose        bool     // Enable verbose logging
}

// Location represents a geographic coordinate
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

// ReceiverInfo contains information about a receiver station
type ReceiverInfo struct {
	ID       string    `json:"id"`
	Location Location  `json:"location"`
	Filename string    `json:"filename"`
	SNR      float64   `json:"snr"`
	Metadata *filewriter.Metadata `json:"-"`
	Samples  []complex64 `json:"-"`
}

// TDOAMeasurement represents a time difference measurement between two receivers
type TDOAMeasurement struct {
	Receiver1ID    string  `json:"receiver1_id"`
	Receiver2ID    string  `json:"receiver2_id"`
	TimeDiff       float64 `json:"time_diff_ns"`      // Time difference in nanoseconds
	DistanceDiff   float64 `json:"distance_diff_m"`   // Distance difference in meters
	Confidence     float64 `json:"confidence"`        // Measurement confidence (0-1)
	CorrelationPeak float64 `json:"correlation_peak"` // Cross-correlation peak value
}

// Result holds the complete TDOA processing results
type Result struct {
	Location           Location           `json:"location"`
	Confidence         float64            `json:"confidence"`
	ErrorRadius        float64            `json:"error_radius_m"`
	Algorithm          string             `json:"algorithm"`
	Frequency          float64            `json:"frequency_hz"`
	ProcessingTime     time.Time          `json:"processing_time"`
	ReceiverLocations  []ReceiverInfo     `json:"receivers"`
	TDOAMeasurements   []TDOAMeasurement  `json:"tdoa_measurements"`
	HeatmapPoints      []HeatmapPoint     `json:"heatmap_points,omitempty"`
}

// HeatmapPoint represents a point in the probability heatmap
type HeatmapPoint struct {
	Location    Location `json:"location"`
	Probability float64  `json:"probability"`
}

// Processor handles TDOA signal processing
type Processor struct {
	config *Config
}

// NewProcessor creates a new TDOA processor with the given configuration
func NewProcessor(config *Config) (*Processor, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate configuration
	if config.Confidence < 0.0 || config.Confidence > 1.0 {
		return nil, fmt.Errorf("confidence must be between 0.0 and 1.0")
	}

	if config.MaxDistance <= 0 {
		return nil, fmt.Errorf("max distance must be positive")
	}

	// Set default algorithm if not specified
	if config.Algorithm == "" {
		config.Algorithm = "basic"
	}

	return &Processor{config: config}, nil
}

// ProcessFiles processes multiple argus data files to calculate transmitter location
func (p *Processor) ProcessFiles(filenames []string) (*Result, error) {
	if len(filenames) < 3 {
		return nil, fmt.Errorf("TDOA requires at least 3 files, got %d", len(filenames))
	}

	fmt.Printf("📊 Loading and validating %d data files...\n", len(filenames))

	// Load all files and extract receiver information
	receivers, err := p.loadReceivers(filenames)
	if err != nil {
		return nil, fmt.Errorf("failed to load receivers: %w", err)
	}

	// Validate that all files have compatible parameters
	if err := p.validateReceivers(receivers); err != nil {
		return nil, fmt.Errorf("receiver validation failed: %w", err)
	}

	fmt.Printf("✅ Loaded %d receivers with compatible parameters\n", len(receivers))
	fmt.Printf("⚙️  Performing cross-correlation analysis...\n")

	// Perform cross-correlation analysis between all receiver pairs
	measurements, err := p.performTDOAAnalysis(receivers)
	if err != nil {
		return nil, fmt.Errorf("TDOA analysis failed: %w", err)
	}

	fmt.Printf("📈 Generated %d TDOA measurements\n", len(measurements))
	fmt.Printf("🧮 Calculating transmitter location...\n")

	// Calculate transmitter location using TDOA measurements
	location, confidence, errorRadius, err := p.calculateLocation(receivers, measurements)
	if err != nil {
		return nil, fmt.Errorf("location calculation failed: %w", err)
	}

	// Generate heatmap if requested
	var heatmapPoints []HeatmapPoint
	if p.config.Algorithm == "heatmap" || p.config.Verbose {
		fmt.Printf("🗺️  Generating probability heatmap...\n")
		heatmapPoints = p.generateHeatmap(receivers, measurements, *location, errorRadius)
		fmt.Printf("   📍 Generated %d heatmap points\n", len(heatmapPoints))
	}

	result := &Result{
		Location:          *location,
		Confidence:        confidence,
		ErrorRadius:       errorRadius,
		Algorithm:         p.config.Algorithm,
		Frequency:         float64(receivers[0].Metadata.Frequency),
		ProcessingTime:    time.Now(),
		ReceiverLocations: receivers,
		TDOAMeasurements:  measurements,
		HeatmapPoints:     heatmapPoints,
	}

	fmt.Printf("🎯 Location calculated: %.6f°, %.6f° (±%.1fm, confidence: %.2f)\n",
		location.Latitude, location.Longitude, errorRadius, confidence)

	return result, nil
}

// loadReceivers loads data from all input files and creates receiver information
func (p *Processor) loadReceivers(filenames []string) ([]ReceiverInfo, error) {
	receivers := make([]ReceiverInfo, len(filenames))

	for i, filename := range filenames {
		// Get file size for progress estimation
		if fileInfo, err := os.Stat(filename); err == nil {
			sizeMB := float64(fileInfo.Size()) / (1024 * 1024)
			fmt.Printf("   📁 Loading %s (%.1f MB) (%d/%d)...\n", 
				filepath.Base(filename), sizeMB, i+1, len(filenames))
		} else {
			fmt.Printf("   📁 Loading %s (%d/%d)...\n", filepath.Base(filename), i+1, len(filenames))
		}
		
		// Use progress-aware file reading for large files
		metadata, samples, err := p.readFileWithProgress(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
		}
		
		fmt.Printf("      ✅ Loaded %d samples\n", len(samples))

		// Calculate basic signal metrics
		snr := p.calculateSNR(samples)

		receivers[i] = ReceiverInfo{
			ID:       fmt.Sprintf("R%d", i+1),
			Location: Location{
				Latitude:  metadata.GPSLocation.Latitude,
				Longitude: metadata.GPSLocation.Longitude,
				Altitude:  metadata.GPSLocation.Altitude,
			},
			Filename: filename,
			SNR:      snr,
			Metadata: metadata,
			Samples:  samples,
		}

		if p.config.Verbose {
			fmt.Printf("   %s: %.6f°, %.6f° (SNR: %.1f dB, %d samples)\n",
				receivers[i].ID, receivers[i].Location.Latitude, receivers[i].Location.Longitude,
				snr, len(samples))
		}
	}

	return receivers, nil
}

// validateReceivers ensures all receivers have compatible parameters for TDOA
func (p *Processor) validateReceivers(receivers []ReceiverInfo) error {
	if len(receivers) < 3 {
		return fmt.Errorf("need at least 3 receivers")
	}

	// Check that all receivers have the same frequency and sample rate
	refFreq := receivers[0].Metadata.Frequency
	refSampleRate := receivers[0].Metadata.SampleRate

	for i, receiver := range receivers[1:] {
		if receiver.Metadata.Frequency != refFreq {
			return fmt.Errorf("frequency mismatch: receiver %d has %.0f Hz, expected %.0f Hz",
				i+2, float64(receiver.Metadata.Frequency), float64(refFreq))
		}
		if receiver.Metadata.SampleRate != refSampleRate {
			return fmt.Errorf("sample rate mismatch: receiver %d has %d Hz, expected %d Hz",
				i+2, receiver.Metadata.SampleRate, refSampleRate)
		}
	}

	// Check that receivers have different locations
	for i := 0; i < len(receivers); i++ {
		for j := i + 1; j < len(receivers); j++ {
			dist := p.distanceBetweenLocations(receivers[i].Location, receivers[j].Location)
			if dist < 10.0 { // Less than 10 meters apart
				return fmt.Errorf("receivers %s and %s are too close (%.1f m apart)",
					receivers[i].ID, receivers[j].ID, dist)
			}
		}
	}

	// Check time synchronization (collection times should be very close)
	refTime := receivers[0].Metadata.CollectionTime
	for i, receiver := range receivers[1:] {
		timeDiff := receiver.Metadata.CollectionTime.Sub(refTime)
		if math.Abs(timeDiff.Seconds()) > 1.0 { // More than 1 second difference
			return fmt.Errorf("time sync issue: receiver %d collection time differs by %.1f seconds",
				i+2, timeDiff.Seconds())
		}
	}

	return nil
}

// calculateSNR estimates the signal-to-noise ratio of the samples
func (p *Processor) calculateSNR(samples []complex64) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	// Calculate power of all samples
	var totalPower float64
	for _, sample := range samples {
		power := float64(real(sample)*real(sample) + imag(sample)*imag(sample))
		totalPower += power
	}
	avgPower := totalPower / float64(len(samples))

	// Estimate noise floor (use lower 25% of power measurements)
	powers := make([]float64, len(samples))
	for i, sample := range samples {
		powers[i] = float64(real(sample)*real(sample) + imag(sample)*imag(sample))
	}
	sort.Float64s(powers)
	
	noiseFloor := 0.0
	noiseCount := len(powers) / 4 // Bottom 25%
	for i := 0; i < noiseCount; i++ {
		noiseFloor += powers[i]
	}
	noiseFloor /= float64(noiseCount)

	// Calculate SNR in dB
	if noiseFloor > 0 {
		snr := 10 * math.Log10(avgPower/noiseFloor)
		return snr
	}
	return 0.0
}

// distanceBetweenLocations calculates the distance between two GPS coordinates in meters
func (p *Processor) distanceBetweenLocations(loc1, loc2 Location) float64 {
	const R = 6371000 // Earth radius in meters

	lat1Rad := loc1.Latitude * math.Pi / 180
	lat2Rad := loc2.Latitude * math.Pi / 180
	deltaLatRad := (loc2.Latitude - loc1.Latitude) * math.Pi / 180
	deltaLonRad := (loc2.Longitude - loc1.Longitude) * math.Pi / 180

	a := math.Sin(deltaLatRad/2)*math.Sin(deltaLatRad/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLonRad/2)*math.Sin(deltaLonRad/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// performTDOAAnalysis performs cross-correlation analysis between all receiver pairs
func (p *Processor) performTDOAAnalysis(receivers []ReceiverInfo) ([]TDOAMeasurement, error) {
	var measurements []TDOAMeasurement
	var allMeasurements []TDOAMeasurement // Keep all measurements for output

	// Calculate total number of pairs
	totalPairs := len(receivers) * (len(receivers) - 1) / 2
	pairCount := 0

	// Process all unique pairs of receivers
	for i := 0; i < len(receivers); i++ {
		for j := i + 1; j < len(receivers); j++ {
			pairCount++
			fmt.Printf("   🔗 Correlating %s ↔ %s (%d/%d)...\n", 
				receivers[i].ID, receivers[j].ID, pairCount, totalPairs)
				
			measurement, err := p.crossCorrelate(receivers[i], receivers[j])
			if err != nil {
				if p.config.Verbose {
					fmt.Printf("⚠️  Cross-correlation failed for %s-%s: %v\n", 
						receivers[i].ID, receivers[j].ID, err)
				}
				continue
			}

			// Always keep all measurements for output files
			allMeasurements = append(allMeasurements, *measurement)

			// Filter measurements by confidence threshold for location calculation
			if measurement.Confidence >= p.config.Confidence {
				measurements = append(measurements, *measurement)
				fmt.Printf("      ✅ Δt=%.1fns, Δd=%.1fm, confidence=%.3f\n",
					measurement.TimeDiff, measurement.DistanceDiff, measurement.Confidence)
			} else {
				fmt.Printf("      ⚠️  Low confidence: %.3f (threshold: %.3f) - included in output\n",
					measurement.Confidence, p.config.Confidence)
			}
		}
	}

	// If no high-confidence measurements, use all measurements but warn user
	if len(measurements) == 0 {
		fmt.Printf("⚠️  No TDOA measurements met confidence threshold of %.2f\n", p.config.Confidence)
		if len(allMeasurements) > 0 {
			fmt.Printf("   📍 Using %d low-confidence measurements for approximate location\n", len(allMeasurements))
			measurements = allMeasurements
		} else {
			return nil, fmt.Errorf("no valid TDOA measurements could be calculated")
		}
	}

	return measurements, nil
}

// crossCorrelate performs cross-correlation between two receiver signals
func (p *Processor) crossCorrelate(r1, r2 ReceiverInfo) (*TDOAMeasurement, error) {
	// For now, implement a simplified correlation method
	// In a full implementation, this would use FFT-based cross-correlation
	
	// Ensure we have enough samples
	minLen := len(r1.Samples)
	if len(r2.Samples) < minLen {
		minLen = len(r2.Samples)
	}
	
	if minLen < 1000 {
		return nil, fmt.Errorf("insufficient samples for correlation")
	}

	// Use first 10000 samples for correlation (for performance)
	corrLen := minLen
	if corrLen > 10000 {
		corrLen = 10000
	}

	samples1 := r1.Samples[:corrLen]
	samples2 := r2.Samples[:corrLen]

	// Calculate cross-correlation using simplified time-domain method
	maxCorr := 0.0
	bestDelay := 0
	maxSearchDelay := corrLen / 10 // Search within 10% of signal length

	if p.config.Verbose {
		fmt.Printf("         🔍 Searching %d delay positions...\n", 2*maxSearchDelay+1)
	}

	for delay := -maxSearchDelay; delay <= maxSearchDelay; delay++ {
		corr := p.calculateCorrelation(samples1, samples2, delay)
		if math.Abs(corr) > math.Abs(maxCorr) {
			maxCorr = corr
			bestDelay = delay
		}
		
		// Show progress for long searches
		if p.config.Verbose && maxSearchDelay > 1000 && (delay-(-maxSearchDelay))%(maxSearchDelay/5) == 0 {
			progress := float64(delay+maxSearchDelay) / float64(2*maxSearchDelay) * 100
			fmt.Printf("         📊 Progress: %.0f%%\n", progress)
		}
	}

	// Convert sample delay to time delay
	sampleRate := float64(r1.Metadata.SampleRate)
	timeDiffNs := float64(bestDelay) * 1e9 / sampleRate

	// Convert time delay to distance difference (speed of light)
	const speedOfLight = 299792458.0 // m/s
	distanceDiffM := timeDiffNs * speedOfLight / 1e9

	// Calculate confidence based on correlation peak strength
	confidence := math.Min(math.Abs(maxCorr), 1.0)

	return &TDOAMeasurement{
		Receiver1ID:     r1.ID,
		Receiver2ID:     r2.ID,
		TimeDiff:        timeDiffNs,
		DistanceDiff:    distanceDiffM,
		Confidence:      confidence,
		CorrelationPeak: maxCorr,
	}, nil
}

// calculateCorrelation calculates normalized cross-correlation at a specific delay
func (p *Processor) calculateCorrelation(sig1, sig2 []complex64, delay int) float64 {
	if len(sig1) == 0 || len(sig2) == 0 {
		return 0.0
	}

	// Determine overlap region
	start1 := 0
	start2 := 0

	if delay > 0 {
		start2 = delay
		if start2 >= len(sig2) {
			return 0.0
		}
	} else if delay < 0 {
		start1 = -delay
		if start1 >= len(sig1) {
			return 0.0
		}
	}

	// Calculate overlap length
	overlapLen := len(sig1) - start1
	if len(sig2)-start2 < overlapLen {
		overlapLen = len(sig2) - start2
	}

	if overlapLen <= 0 {
		return 0.0
	}

	// Calculate correlation
	var sum1, sum2, sum1Sq, sum2Sq, sumProduct complex128

	for i := 0; i < overlapLen; i++ {
		s1 := complex128(sig1[start1+i])
		s2 := complex128(sig2[start2+i])

		sum1 += s1
		sum2 += s2
		sum1Sq += s1 * s1
		sum2Sq += s2 * s2
		sumProduct += s1 * complex(real(s2), -imag(s2)) // Complex conjugate
	}

	n := float64(overlapLen)
	mean1 := sum1 / complex(n, 0)
	mean2 := sum2 / complex(n, 0)

	// Calculate normalized correlation coefficient
	num := sumProduct - complex(n, 0)*mean1*complex(real(mean2), -imag(mean2))
	
	var1 := sum1Sq - complex(n, 0)*mean1*complex(real(mean1), -imag(mean1))
	var2 := sum2Sq - complex(n, 0)*mean2*complex(real(mean2), -imag(mean2))

	denom := complex(math.Sqrt(real(var1) * real(var2)), 0)
	
	if real(denom) == 0 {
		return 0.0
	}

	correlation := num / denom
	return real(correlation) // Return real part of normalized correlation
}

// calculateLocation calculates transmitter location using TDOA measurements
func (p *Processor) calculateLocation(receivers []ReceiverInfo, measurements []TDOAMeasurement) (*Location, float64, float64, error) {
	if len(measurements) == 0 {
		return nil, 0, 0, fmt.Errorf("no TDOA measurements available")
	}
	
	// Warn if we have fewer than optimal measurements
	if len(measurements) < 3 {
		fmt.Printf("⚠️  Only %d TDOA measurements available (optimal: 3+) - accuracy may be limited\n", len(measurements))
	}

	// For this implementation, use a simplified least-squares approach
	// In practice, this would use more sophisticated algorithms like Newton-Raphson
	
	// Start with centroid of receivers as initial guess
	var sumLat, sumLon float64
	for _, r := range receivers {
		sumLat += r.Location.Latitude
		sumLon += r.Location.Longitude
	}
	
	initialLat := sumLat / float64(len(receivers))
	initialLon := sumLon / float64(len(receivers))

	// For now, return the centroid with estimated error
	// TODO: Implement proper hyperbolic positioning algorithm
	location := &Location{
		Latitude:  initialLat,
		Longitude: initialLon,
		Altitude:  0.0, // Ground level assumed
	}

	// Calculate average confidence from measurements
	var avgConfidence float64
	for _, m := range measurements {
		avgConfidence += m.Confidence
	}
	avgConfidence /= float64(len(measurements))

	// Estimate error radius based on geometry and confidence
	errorRadius := p.estimateErrorRadius(receivers, measurements, avgConfidence)

	return location, avgConfidence, errorRadius, nil
}

// estimateErrorRadius estimates the positioning error radius
func (p *Processor) estimateErrorRadius(receivers []ReceiverInfo, measurements []TDOAMeasurement, confidence float64) float64 {
	// Calculate geometric dilution of precision (GDOP) approximation
	// For now, use a simple estimate based on receiver spacing and confidence
	
	var avgSpacing float64
	count := 0
	
	for i := 0; i < len(receivers); i++ {
		for j := i + 1; j < len(receivers); j++ {
			dist := p.distanceBetweenLocations(receivers[i].Location, receivers[j].Location)
			avgSpacing += dist
			count++
		}
	}
	
	if count > 0 {
		avgSpacing /= float64(count)
	}

	// Error radius inversely related to confidence and receiver spacing
	baseError := 100.0 / confidence // Base error in meters
	gdopFactor := 1000.0 / avgSpacing // GDOP approximation
	
	errorRadius := baseError * gdopFactor
	
	// Clamp to reasonable bounds
	if errorRadius < 10.0 {
		errorRadius = 10.0
	}
	if errorRadius > 5000.0 {
		errorRadius = 5000.0
	}
	
	return errorRadius
}

// generateHeatmap generates probability heatmap points around the calculated location
func (p *Processor) generateHeatmap(receivers []ReceiverInfo, measurements []TDOAMeasurement, center Location, errorRadius float64) []HeatmapPoint {
	var points []HeatmapPoint
	
	// Generate grid of points around the center location
	gridSize := 20 // 20x20 grid
	stepSize := errorRadius * 2 / float64(gridSize) // Grid step in meters
	
	for i := 0; i < gridSize; i++ {
		for j := 0; j < gridSize; j++ {
			// Calculate offset from center
			offsetX := (float64(i) - float64(gridSize)/2) * stepSize
			offsetY := (float64(j) - float64(gridSize)/2) * stepSize
			
			// Convert meter offsets to lat/lon offsets (approximate)
			latOffset := offsetY / 111000.0 // Approximate meters per degree latitude
			lonOffset := offsetX / (111000.0 * math.Cos(center.Latitude * math.Pi / 180))
			
			point := Location{
				Latitude:  center.Latitude + latOffset,
				Longitude: center.Longitude + lonOffset,
				Altitude:  center.Altitude,
			}
			
			// Calculate probability based on distance from center
			distance := p.distanceBetweenLocations(center, point)
			probability := math.Exp(-distance * distance / (2 * errorRadius * errorRadius))
			
			if probability > 0.01 { // Only include points with meaningful probability
				points = append(points, HeatmapPoint{
					Location:    point,
					Probability: probability,
				})
			}
		}
	}
	
	return points
}

// readFileWithProgress reads an argus data file with progress reporting
func (p *Processor) readFileWithProgress(filename string) (*filewriter.Metadata, []complex64, error) {
	// Get file size for progress calculation
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := fileInfo.Size()
	
	// For small files, just use regular reading
	if fileSize < 10*1024*1024 { // Less than 10MB
		return filewriter.ReadFile(filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read header with progress reporting
	fmt.Printf("      📊 Reading header...")
	
	// Read magic header
	magic := make([]byte, 5)
	if _, err := file.Read(magic); err != nil {
		return nil, nil, fmt.Errorf("failed to read magic: %w", err)
	}
	if string(magic) != "ARGUS" {
		return nil, nil, fmt.Errorf("invalid file format")
	}

	var metadata filewriter.Metadata
	
	// Read metadata fields in order (same as filewriter.ReadFile)
	if err := binary.Read(file, binary.LittleEndian, &metadata.FileFormatVersion); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &metadata.Frequency); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &metadata.SampleRate); err != nil {
		return nil, nil, err
	}

	var collectionTimeUnix int64
	var collectionTimeNano int32
	if err := binary.Read(file, binary.LittleEndian, &collectionTimeUnix); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &collectionTimeNano); err != nil {
		return nil, nil, err
	}
	metadata.CollectionTime = time.Unix(collectionTimeUnix, int64(collectionTimeNano))

	if err := binary.Read(file, binary.LittleEndian, &metadata.GPSLocation.Latitude); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &metadata.GPSLocation.Longitude); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &metadata.GPSLocation.Altitude); err != nil {
		return nil, nil, err
	}

	var gpsTimeUnix int64
	var gpsTimeNano int32
	if err := binary.Read(file, binary.LittleEndian, &gpsTimeUnix); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &gpsTimeNano); err != nil {
		return nil, nil, err
	}
	metadata.GPSTimestamp = time.Unix(gpsTimeUnix, int64(gpsTimeNano))

	var deviceInfoLen uint8
	if err := binary.Read(file, binary.LittleEndian, &deviceInfoLen); err != nil {
		return nil, nil, err
	}
	deviceInfoBytes := make([]byte, deviceInfoLen)
	if _, err := file.Read(deviceInfoBytes); err != nil {
		return nil, nil, err
	}
	metadata.DeviceInfo = string(deviceInfoBytes)

	var collectionIDLen uint8
	if err := binary.Read(file, binary.LittleEndian, &collectionIDLen); err != nil {
		return nil, nil, err
	}
	collectionIDBytes := make([]byte, collectionIDLen)
	if _, err := file.Read(collectionIDBytes); err != nil {
		return nil, nil, err
	}
	metadata.CollectionID = string(collectionIDBytes)

	var sampleCount uint32
	if err := binary.Read(file, binary.LittleEndian, &sampleCount); err != nil {
		return nil, nil, err
	}

	fmt.Printf(" ✅ Complete\n")
	fmt.Printf("      📊 Reading %d samples...\n", sampleCount)
	
	// Read samples with progress reporting
	samples := make([]complex64, sampleCount)
	const chunkSize = 1024 * 1024 // 1MB chunks
	samplesPerChunk := chunkSize / 8 // 8 bytes per complex64 (2 float32s)
	
	var samplesRead uint32
	lastProgress := -1
	
	for samplesRead < sampleCount {
		// Calculate how many samples to read in this chunk
		samplesToRead := samplesPerChunk
		if samplesRead + uint32(samplesToRead) > sampleCount {
			samplesToRead = int(sampleCount - samplesRead)
		}
		
		// Read chunk of samples
		for i := 0; i < samplesToRead; i++ {
			var real, imag float32
			if err := binary.Read(file, binary.LittleEndian, &real); err != nil {
				return nil, nil, err
			}
			if err := binary.Read(file, binary.LittleEndian, &imag); err != nil {
				return nil, nil, err
			}
			samples[samplesRead + uint32(i)] = complex(real, imag)
		}
		
		samplesRead += uint32(samplesToRead)
		
		// Calculate and display progress
		progress := int((float64(samplesRead) / float64(sampleCount)) * 100)
		if progress != lastProgress && progress%10 == 0 {
			fmt.Printf("         Progress: %d%%\n", progress)
			lastProgress = progress
		}
	}
	
	fmt.Printf("         Progress: 100%%\n")

	return &metadata, samples, nil
}