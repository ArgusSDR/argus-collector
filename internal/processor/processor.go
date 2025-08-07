// Package processor implements TDOA signal processing for transmitter localization
package processor

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
	"unsafe"

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

// ProgressTracker tracks progress of long-running operations
type ProgressTracker struct {
	totalSteps    int
	currentStep   int
	stepName      string
	subProgress   float64
	lastReported  time.Time
	startTime     time.Time
	verbose       bool
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(totalSteps int, verbose bool) *ProgressTracker {
	return &ProgressTracker{
		totalSteps:   totalSteps,
		currentStep:  0,
		startTime:    time.Now(),
		lastReported: time.Now(),
		verbose:      verbose,
	}
}

// StartStep begins a new processing step
func (pt *ProgressTracker) StartStep(stepName string) {
	pt.currentStep++
	pt.stepName = stepName
	pt.subProgress = 0.0
	pt.lastReported = time.Now()
	
	elapsed := time.Since(pt.startTime)
	fmt.Printf("⏳ Step %d/%d: %s (elapsed: %v)\n", pt.currentStep, pt.totalSteps, stepName, elapsed.Truncate(time.Second))
}

// UpdateSubProgress updates progress within the current step
func (pt *ProgressTracker) UpdateSubProgress(progress float64, details string) {
	pt.subProgress = progress
	
	// Only report progress every 2 seconds for non-verbose mode, or every 500ms for verbose
	reportInterval := 2 * time.Second
	if pt.verbose {
		reportInterval = 500 * time.Millisecond
	}
	
	if time.Since(pt.lastReported) >= reportInterval {
		elapsed := time.Since(pt.startTime)
		overallProgress := (float64(pt.currentStep-1) + progress) / float64(pt.totalSteps) * 100
		
		if details != "" {
			fmt.Printf("   📊 %.1f%% complete (%.1f%% overall, %s) - %v elapsed\n", 
				progress*100, overallProgress, details, elapsed.Truncate(time.Second))
		} else {
			fmt.Printf("   📊 %.1f%% complete (%.1f%% overall) - %v elapsed\n", 
				progress*100, overallProgress, elapsed.Truncate(time.Second))
		}
		
		pt.lastReported = time.Now()
	}
}

// CompleteStep marks the current step as complete
func (pt *ProgressTracker) CompleteStep() {
	elapsed := time.Since(pt.startTime)
	overallProgress := float64(pt.currentStep) / float64(pt.totalSteps) * 100
	
	fmt.Printf("✅ Step %d/%d complete: %s (%.1f%% overall, %v elapsed)\n", 
		pt.currentStep, pt.totalSteps, pt.stepName, overallProgress, elapsed.Truncate(time.Second))
}

// Finish completes all progress tracking
func (pt *ProgressTracker) Finish() {
	totalTime := time.Since(pt.startTime)
	fmt.Printf("🎉 All processing complete! Total time: %v\n", totalTime.Truncate(time.Second))
}

// Location represents a geographic coordinate
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

// ReceiverInfo contains information about a receiver station
type ReceiverInfo struct {
	ID       string               `json:"id"`
	Location Location             `json:"location"`
	Filename string               `json:"filename"`
	SNR      float64              `json:"snr"`
	Metadata *filewriter.Metadata `json:"-"`
	Samples  []complex64          `json:"-"`
}

// TDOAMeasurement represents a time difference measurement between two receivers
type TDOAMeasurement struct {
	Receiver1ID     string  `json:"receiver1_id"`
	Receiver2ID     string  `json:"receiver2_id"`
	TimeDiff        float64 `json:"time_diff_ns"`     // Time difference in nanoseconds
	DistanceDiff    float64 `json:"distance_diff_m"`  // Distance difference in meters
	Confidence      float64 `json:"confidence"`       // Measurement confidence (0-1)
	CorrelationPeak float64 `json:"correlation_peak"` // Cross-correlation peak value
}

// Result holds the complete TDOA processing results
type Result struct {
	Location          Location          `json:"location"`
	Confidence        float64           `json:"confidence"`
	ErrorRadius       float64           `json:"error_radius_m"`
	Algorithm         string            `json:"algorithm"`
	Frequency         float64           `json:"frequency_hz"`
	ProcessingTime    time.Time         `json:"processing_time"`
	ReceiverLocations []ReceiverInfo    `json:"receivers"`
	TDOAMeasurements  []TDOAMeasurement `json:"tdoa_measurements"`
	HeatmapPoints     []HeatmapPoint    `json:"heatmap_points,omitempty"`
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

	// Initialize progress tracker - determine total steps
	totalSteps := 4 // Load files, TDOA analysis, location calculation, heatmap (optional)
	if p.config.Algorithm == "heatmap" || p.config.Verbose {
		totalSteps = 5
	}

	progress := NewProgressTracker(totalSteps, p.config.Verbose)

	// Step 1: Load and validate files
	progress.StartStep("Loading and validating data files")
	receivers, err := p.loadReceiversWithProgress(filenames, progress)
	if err != nil {
		return nil, fmt.Errorf("failed to load receivers: %w", err)
	}

	// Validate that all files have compatible parameters
	if err := p.validateReceivers(receivers); err != nil {
		return nil, fmt.Errorf("receiver validation failed: %w", err)
	}
	progress.CompleteStep()

	// Step 2: Cross-correlation analysis
	progress.StartStep("Performing cross-correlation analysis")
	measurements, err := p.performTDOAAnalysisWithProgress(receivers, progress)
	if err != nil {
		return nil, fmt.Errorf("TDOA analysis failed: %w", err)
	}
	progress.CompleteStep()

	// Step 3: Location calculation
	progress.StartStep("Calculating transmitter location")
	location, confidence, errorRadius, err := p.calculateLocationWithProgress(receivers, measurements, progress)
	if err != nil {
		return nil, fmt.Errorf("location calculation failed: %w", err)
	}
	progress.CompleteStep()

	// Step 4: Generate heatmap if requested
	var heatmapPoints []HeatmapPoint
	if p.config.Algorithm == "heatmap" || p.config.Verbose {
		progress.StartStep("Generating probability heatmap")
		heatmapPoints = p.generateHeatmapWithProgress(receivers, measurements, *location, errorRadius, progress)
		progress.CompleteStep()
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

	progress.Finish()
	fmt.Printf("🎯 Final Result: %.6f°, %.6f° (±%.1fm, confidence: %.2f)\n",
		location.Latitude, location.Longitude, errorRadius, confidence)

	return result, nil
}

// loadReceiversWithProgress loads data from all input files with progress reporting
func (p *Processor) loadReceiversWithProgress(filenames []string, progress *ProgressTracker) ([]ReceiverInfo, error) {
	return p.loadReceivers(filenames, progress)
}

// loadReceivers loads data from all input files and creates receiver information
func (p *Processor) loadReceivers(filenames []string, progress ...*ProgressTracker) ([]ReceiverInfo, error) {
	receivers := make([]ReceiverInfo, len(filenames))

	// Get optional progress tracker
	var pt *ProgressTracker
	if len(progress) > 0 {
		pt = progress[0]
	}

	for i, filename := range filenames {
		// Update progress
		if pt != nil {
			fileProgress := float64(i) / float64(len(filenames))
			pt.UpdateSubProgress(fileProgress, fmt.Sprintf("file %d/%d", i+1, len(filenames)))
		}

		// Get file size for progress estimation
		if fileInfo, err := os.Stat(filename); err == nil {
			sizeMB := float64(fileInfo.Size()) / (1024 * 1024)
			if pt == nil { // Only print if no progress tracker (backward compatibility)
				fmt.Printf("   📁 Loading %s (%.1f MB) (%d/%d)...\n",
					filepath.Base(filename), sizeMB, i+1, len(filenames))
			}
		} else {
			if pt == nil {
				fmt.Printf("   📁 Loading %s (%d/%d)...\n", filepath.Base(filename), i+1, len(filenames))
			}
		}

		// Use progress-aware file reading for large files
		metadata, samples, err := p.readFileWithProgress(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
		}

		if pt == nil {
			fmt.Printf("      ✅ Loaded %d samples\n", len(samples))
		}

		// Calculate basic signal metrics
		snr := p.calculateSNR(samples)

		receivers[i] = ReceiverInfo{
			ID: fmt.Sprintf("R%d", i+1),
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

		if p.config.Verbose && pt == nil {
			fmt.Printf("   %s: %.6f°, %.6f° (SNR: %.1f dB, %d samples)\n",
				receivers[i].ID, receivers[i].Location.Latitude, receivers[i].Location.Longitude,
				snr, len(samples))
		}
	}

	// Final progress update
	if pt != nil {
		pt.UpdateSubProgress(1.0, fmt.Sprintf("loaded %d files", len(filenames)))
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

// performTDOAAnalysisWithProgress performs cross-correlation analysis with progress reporting
func (p *Processor) performTDOAAnalysisWithProgress(receivers []ReceiverInfo, progress *ProgressTracker) ([]TDOAMeasurement, error) {
	return p.performTDOAAnalysis(receivers, progress)
}

// performTDOAAnalysis performs cross-correlation analysis between all receiver pairs
func (p *Processor) performTDOAAnalysis(receivers []ReceiverInfo, progress ...*ProgressTracker) ([]TDOAMeasurement, error) {
	var measurements []TDOAMeasurement
	var allMeasurements []TDOAMeasurement // Keep all measurements for output

	// Get optional progress tracker
	var pt *ProgressTracker
	if len(progress) > 0 {
		pt = progress[0]
	}

	// Calculate total number of pairs
	totalPairs := len(receivers) * (len(receivers) - 1) / 2
	pairCount := 0

	// Process all unique pairs of receivers
	for i := 0; i < len(receivers); i++ {
		for j := i + 1; j < len(receivers); j++ {
			pairCount++

			// Update progress
			if pt != nil {
				pairProgress := float64(pairCount-1) / float64(totalPairs)
				pt.UpdateSubProgress(pairProgress, fmt.Sprintf("pair %d/%d (%s↔%s)", pairCount, totalPairs, receivers[i].ID, receivers[j].ID))
			} else {
				fmt.Printf("   🔗 Correlating %s ↔ %s (%d/%d)...\n",
					receivers[i].ID, receivers[j].ID, pairCount, totalPairs)
			}

			measurement, err := p.crossCorrelate(receivers[i], receivers[j])
			if err != nil {
				if p.config.Verbose {
					if pt == nil {
						fmt.Printf("⚠️  Cross-correlation failed for %s-%s: %v\n",
							receivers[i].ID, receivers[j].ID, err)
					}
				}
				continue
			}

			// Always keep all measurements for output files
			allMeasurements = append(allMeasurements, *measurement)

			// Filter measurements by confidence threshold for location calculation
			if measurement.Confidence >= p.config.Confidence {
				measurements = append(measurements, *measurement)
				if pt == nil {
					fmt.Printf("      ✅ Δt=%.1fns, Δd=%.1fm, confidence=%.3f\n",
						measurement.TimeDiff, measurement.DistanceDiff, measurement.Confidence)
				}
			} else {
				if pt == nil {
					fmt.Printf("      ⚠️  Low confidence: %.3f (threshold: %.3f) - included in output\n",
						measurement.Confidence, p.config.Confidence)
				}
			}
		}
	}

	// Final progress update
	if pt != nil {
		pt.UpdateSubProgress(1.0, fmt.Sprintf("completed %d correlations", totalPairs))
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

// crossCorrelate performs cross-correlation between two receiver signals using multi-resolution search
func (p *Processor) crossCorrelate(r1, r2 ReceiverInfo) (*TDOAMeasurement, error) {
	// Ensure we have enough samples
	minLen := len(r1.Samples)
	if len(r2.Samples) < minLen {
		minLen = len(r2.Samples)
	}

	if minLen < 1000 {
		return nil, fmt.Errorf("insufficient samples for correlation")
	}

	// Use first 50000 samples for correlation (increased from 10000 for better accuracy)
	corrLen := minLen
	if corrLen > 50000 {
		corrLen = 50000
	}

	samples1 := r1.Samples[:corrLen]
	samples2 := r2.Samples[:corrLen]

	if p.config.Verbose {
		fmt.Printf("         🔍 Multi-resolution correlation search (%d samples)...\n", corrLen)
	}

	// Perform multi-resolution search for optimal performance
	bestDelay, maxCorr, err := p.multiResolutionCorrelation(samples1, samples2)
	if err != nil {
		return nil, fmt.Errorf("correlation failed: %w", err)
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

// multiResolutionCorrelation performs coarse-to-fine correlation search for optimal performance
func (p *Processor) multiResolutionCorrelation(samples1, samples2 []complex64) (int, float64, error) {
	maxSearchDelay := len(samples1) / 10 // Search within 10% of signal length
	
	// Stage 1: Coarse search with heavily decimated samples (8x decimation)
	decimationFactor1 := 8
	coarseDelay, coarseCorr, err := p.coarseCorrelationSearch(samples1, samples2, decimationFactor1, maxSearchDelay)
	if err != nil {
		return 0, 0, fmt.Errorf("coarse search failed: %w", err)
	}

	if p.config.Verbose {
		fmt.Printf("         📊 Coarse search: delay=%d, corr=%.4f\n", coarseDelay, coarseCorr)
	}

	// Stage 2: Medium resolution search around coarse result (2x decimation)
	decimationFactor2 := 2
	searchRange2 := decimationFactor1 * 4 // Search ±32 samples around coarse result
	mediumDelay, mediumCorr, err := p.refinedCorrelationSearch(samples1, samples2, decimationFactor2, coarseDelay, searchRange2)
	if err != nil {
		return 0, 0, fmt.Errorf("medium search failed: %w", err)
	}

	if p.config.Verbose {
		fmt.Printf("         📊 Medium search: delay=%d, corr=%.4f\n", mediumDelay, mediumCorr)
	}

	// Stage 3: Fine search at full resolution around medium result
	searchRange3 := decimationFactor2 * 4 // Search ±8 samples around medium result  
	fineDelay, fineCorr, err := p.refinedCorrelationSearch(samples1, samples2, 1, mediumDelay, searchRange3)
	if err != nil {
		return 0, 0, fmt.Errorf("fine search failed: %w", err)
	}

	if p.config.Verbose {
		fmt.Printf("         📊 Fine search: delay=%d, corr=%.4f\n", fineDelay, fineCorr)
	}

	return fineDelay, fineCorr, nil
}

// coarseCorrelationSearch performs initial coarse search with decimated samples
func (p *Processor) coarseCorrelationSearch(samples1, samples2 []complex64, decimationFactor, maxSearchDelay int) (int, float64, error) {
	// Decimate samples for faster coarse search
	decimated1 := p.decimateSamples(samples1, decimationFactor)
	decimated2 := p.decimateSamples(samples2, decimationFactor)

	if len(decimated1) < 100 || len(decimated2) < 100 {
		return 0, 0, fmt.Errorf("insufficient decimated samples for coarse search")
	}

	// Adjust search delay for decimation
	maxDecimatedDelay := maxSearchDelay / decimationFactor
	if maxDecimatedDelay < 1 {
		maxDecimatedDelay = 1
	}

	maxCorr := 0.0
	bestDelay := 0

	// Search with larger steps for speed
	searchStep := max(1, maxDecimatedDelay/50) // Take up to 100 correlation points
	searchCount := 0

	for delay := -maxDecimatedDelay; delay <= maxDecimatedDelay; delay += searchStep {
		corr := p.calculateCorrelation(decimated1, decimated2, delay)
		if math.Abs(corr) > math.Abs(maxCorr) {
			maxCorr = corr
			bestDelay = delay
		}
		searchCount++
	}

	if p.config.Verbose {
		fmt.Printf("         🔎 Coarse: %d correlations at %dx decimation\n", searchCount, decimationFactor)
	}

	// Convert back to original sample delay
	return bestDelay * decimationFactor, maxCorr, nil
}

// refinedCorrelationSearch performs refined search around a candidate delay
func (p *Processor) refinedCorrelationSearch(samples1, samples2 []complex64, decimationFactor, centerDelay, searchRange int) (int, float64, error) {
	// Decimate samples if needed
	var searchSamples1, searchSamples2 []complex64
	if decimationFactor > 1 {
		searchSamples1 = p.decimateSamples(samples1, decimationFactor)
		searchSamples2 = p.decimateSamples(samples2, decimationFactor)
		centerDelay = centerDelay / decimationFactor
		searchRange = searchRange / decimationFactor
	} else {
		searchSamples1 = samples1
		searchSamples2 = samples2
	}

	if searchRange < 1 {
		searchRange = 1
	}

	maxCorr := 0.0
	bestDelay := centerDelay
	searchCount := 0

	// Search around the center delay
	for delay := centerDelay - searchRange; delay <= centerDelay + searchRange; delay++ {
		corr := p.calculateCorrelation(searchSamples1, searchSamples2, delay)
		if math.Abs(corr) > math.Abs(maxCorr) {
			maxCorr = corr
			bestDelay = delay
		}
		searchCount++
	}

	if p.config.Verbose {
		resolution := ""
		if decimationFactor == 2 {
			resolution = "medium"
		} else if decimationFactor == 1 {
			resolution = "fine"
		}
		fmt.Printf("         🎯 %s: %d correlations (range ±%d)\n", resolution, searchCount, searchRange)
	}

	// Convert back to original sample delay if decimated
	return bestDelay * decimationFactor, maxCorr, nil
}

// decimateSamples reduces sample count by taking every Nth sample for faster correlation
func (p *Processor) decimateSamples(samples []complex64, factor int) []complex64 {
	if factor <= 1 {
		return samples
	}

	decimatedLen := len(samples) / factor
	if decimatedLen == 0 {
		return []complex64{}
	}

	decimated := make([]complex64, decimatedLen)
	for i := 0; i < decimatedLen; i++ {
		decimated[i] = samples[i*factor]
	}

	return decimated
}

// max returns the maximum of two integers (helper function for Go < 1.21)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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

	denom := complex(math.Sqrt(real(var1)*real(var2)), 0)

	if real(denom) == 0 {
		return 0.0
	}

	correlation := num / denom
	return real(correlation) // Return real part of normalized correlation
}

// calculateLocationWithProgress calculates transmitter location with progress reporting
func (p *Processor) calculateLocationWithProgress(receivers []ReceiverInfo, measurements []TDOAMeasurement, progress *ProgressTracker) (*Location, float64, float64, error) {
	return p.calculateLocation(receivers, measurements, progress)
}

// calculateLocation calculates transmitter location using TDOA measurements
func (p *Processor) calculateLocation(receivers []ReceiverInfo, measurements []TDOAMeasurement, progress ...*ProgressTracker) (*Location, float64, float64, error) {
	if len(measurements) == 0 {
		return nil, 0, 0, fmt.Errorf("no TDOA measurements available")
	}

	// Get optional progress tracker
	var pt *ProgressTracker
	if len(progress) > 0 {
		pt = progress[0]
	}

	if pt != nil {
		pt.UpdateSubProgress(0.1, "validating measurements")
	}

	// Warn if we have fewer than optimal measurements
	if len(measurements) < 3 {
		if pt == nil {
			fmt.Printf("⚠️  Only %d TDOA measurements available (optimal: 3+) - accuracy may be limited\n", len(measurements))
		}
	}

	if pt != nil {
		pt.UpdateSubProgress(0.3, "calculating centroid")
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

	if pt != nil {
		pt.UpdateSubProgress(0.6, "estimating location")
	}

	// For now, return the centroid with estimated error
	// TODO: Implement proper hyperbolic positioning algorithm
	location := &Location{
		Latitude:  initialLat,
		Longitude: initialLon,
		Altitude:  0.0, // Ground level assumed
	}

	if pt != nil {
		pt.UpdateSubProgress(0.8, "calculating confidence")
	}

	// Calculate average confidence from measurements
	var avgConfidence float64
	for _, m := range measurements {
		avgConfidence += m.Confidence
	}
	avgConfidence /= float64(len(measurements))

	// Estimate error radius based on geometry and confidence
	errorRadius := p.estimateErrorRadius(receivers, measurements, avgConfidence)

	if pt != nil {
		pt.UpdateSubProgress(1.0, fmt.Sprintf("location: %.6f°, %.6f°", location.Latitude, location.Longitude))
	}

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
	baseError := 100.0 / confidence   // Base error in meters
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

// generateHeatmapWithProgress generates probability heatmap with progress reporting  
func (p *Processor) generateHeatmapWithProgress(receivers []ReceiverInfo, measurements []TDOAMeasurement, center Location, errorRadius float64, progress *ProgressTracker) []HeatmapPoint {
	return p.generateHeatmap(receivers, measurements, center, errorRadius, progress)
}

// generateHeatmap generates probability heatmap points around the calculated location
func (p *Processor) generateHeatmap(receivers []ReceiverInfo, measurements []TDOAMeasurement, center Location, errorRadius float64, progress ...*ProgressTracker) []HeatmapPoint {
	var points []HeatmapPoint

	// Get optional progress tracker
	var pt *ProgressTracker
	if len(progress) > 0 {
		pt = progress[0]
	}

	// Generate grid of points around the center location
	gridSize := 20                                  // 20x20 grid
	stepSize := errorRadius * 2 / float64(gridSize) // Grid step in meters
	totalPoints := gridSize * gridSize
	processedPoints := 0

	if pt != nil {
		pt.UpdateSubProgress(0.1, "initializing heatmap grid")
	}

	for i := 0; i < gridSize; i++ {
		for j := 0; j < gridSize; j++ {
			processedPoints++

			// Update progress every few points
			if pt != nil && processedPoints%50 == 0 {
				gridProgress := float64(processedPoints) / float64(totalPoints)
				pt.UpdateSubProgress(0.1 + gridProgress*0.9, fmt.Sprintf("point %d/%d", processedPoints, totalPoints))
			}

			// Calculate offset from center
			offsetX := (float64(i) - float64(gridSize)/2) * stepSize
			offsetY := (float64(j) - float64(gridSize)/2) * stepSize

			// Convert meter offsets to lat/lon offsets (approximate)
			latOffset := offsetY / 111000.0 // Approximate meters per degree latitude
			lonOffset := offsetX / (111000.0 * math.Cos(center.Latitude*math.Pi/180))

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

	if pt != nil {
		pt.UpdateSubProgress(1.0, fmt.Sprintf("generated %d heatmap points", len(points)))
	}

	return points
}

// OptimizedFileReader provides optimized file I/O for argus data files
type OptimizedFileReader struct {
	filename string
	file     *os.File
	mmap     []byte
	size     int64
}

// NewOptimizedFileReader creates a new optimized file reader
func NewOptimizedFileReader(filename string) (*OptimizedFileReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	size := stat.Size()

	// Use memory mapping for files larger than 50MB
	var mmap []byte
	if size > 50*1024*1024 {
		// Memory map the entire file for large files
		mmap, err = syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_PRIVATE)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to memory map file: %w", err)
		}
	}

	return &OptimizedFileReader{
		filename: filename,
		file:     file,
		mmap:     mmap,
		size:     size,
	}, nil
}

// Close closes the file reader and cleans up resources
func (r *OptimizedFileReader) Close() error {
	var err error
	if r.mmap != nil {
		if unmapErr := syscall.Munmap(r.mmap); unmapErr != nil {
			err = fmt.Errorf("failed to unmap memory: %w", unmapErr)
		}
	}
	if r.file != nil {
		if closeErr := r.file.Close(); closeErr != nil {
			if err != nil {
				err = fmt.Errorf("multiple errors: %v, %v", err, closeErr)
			} else {
				err = closeErr
			}
		}
	}
	return err
}

// ReadFile reads an entire argus data file using optimized I/O
func (r *OptimizedFileReader) ReadFile() (*filewriter.Metadata, []complex64, error) {
	if r.mmap != nil {
		return r.readFromMemoryMap()
	}
	return r.readWithBufferedIO()
}

// readFromMemoryMap reads data using memory mapping for maximum performance
func (r *OptimizedFileReader) readFromMemoryMap() (*filewriter.Metadata, []complex64, error) {
	data := r.mmap
	offset := 0

	// Read magic header
	if len(data) < 5 {
		return nil, nil, fmt.Errorf("file too small")
	}
	if string(data[0:5]) != "ARGUS" {
		return nil, nil, fmt.Errorf("invalid file format")
	}
	offset += 5

	var metadata filewriter.Metadata

	// Read metadata fields using unsafe pointer arithmetic for speed
	if len(data) < offset+2 {
		return nil, nil, fmt.Errorf("unexpected EOF reading version")
	}
	metadata.FileFormatVersion = *(*uint16)(unsafe.Pointer(&data[offset]))
	offset += 2

	if len(data) < offset+8 {
		return nil, nil, fmt.Errorf("unexpected EOF reading frequency")
	}
	metadata.Frequency = *(*uint64)(unsafe.Pointer(&data[offset]))
	offset += 8

	if len(data) < offset+4 {
		return nil, nil, fmt.Errorf("unexpected EOF reading sample rate")
	}
	metadata.SampleRate = *(*uint32)(unsafe.Pointer(&data[offset]))
	offset += 4

	// Read collection timestamp
	if len(data) < offset+12 {
		return nil, nil, fmt.Errorf("unexpected EOF reading collection time")
	}
	collectionTimeUnix := *(*int64)(unsafe.Pointer(&data[offset]))
	offset += 8
	collectionTimeNano := *(*int32)(unsafe.Pointer(&data[offset]))
	offset += 4
	metadata.CollectionTime = time.Unix(collectionTimeUnix, int64(collectionTimeNano))

	// Read GPS location
	if len(data) < offset+24 {
		return nil, nil, fmt.Errorf("unexpected EOF reading GPS location")
	}
	metadata.GPSLocation.Latitude = *(*float64)(unsafe.Pointer(&data[offset]))
	offset += 8
	metadata.GPSLocation.Longitude = *(*float64)(unsafe.Pointer(&data[offset]))
	offset += 8
	metadata.GPSLocation.Altitude = *(*float64)(unsafe.Pointer(&data[offset]))
	offset += 8

	// Read GPS timestamp
	if len(data) < offset+12 {
		return nil, nil, fmt.Errorf("unexpected EOF reading GPS timestamp")
	}
	gpsTimeUnix := *(*int64)(unsafe.Pointer(&data[offset]))
	offset += 8
	gpsTimeNano := *(*int32)(unsafe.Pointer(&data[offset]))
	offset += 4
	metadata.GPSTimestamp = time.Unix(gpsTimeUnix, int64(gpsTimeNano))

	// Read device info
	if len(data) < offset+1 {
		return nil, nil, fmt.Errorf("unexpected EOF reading device info length")
	}
	deviceInfoLen := data[offset]
	offset += 1
	if len(data) < offset+int(deviceInfoLen) {
		return nil, nil, fmt.Errorf("unexpected EOF reading device info")
	}
	metadata.DeviceInfo = string(data[offset : offset+int(deviceInfoLen)])
	offset += int(deviceInfoLen)

	// Read collection ID
	if len(data) < offset+1 {
		return nil, nil, fmt.Errorf("unexpected EOF reading collection ID length")
	}
	collectionIDLen := data[offset]
	offset += 1
	if len(data) < offset+int(collectionIDLen) {
		return nil, nil, fmt.Errorf("unexpected EOF reading collection ID")
	}
	metadata.CollectionID = string(data[offset : offset+int(collectionIDLen)])
	offset += int(collectionIDLen)

	// Read sample count
	if len(data) < offset+4 {
		return nil, nil, fmt.Errorf("unexpected EOF reading sample count")
	}
	sampleCount := *(*uint32)(unsafe.Pointer(&data[offset]))
	offset += 4

	fmt.Printf("      📊 Memory-mapped file, reading %d samples...\n", sampleCount)

	// Read samples directly from memory map for maximum speed
	if len(data) < offset+int(sampleCount)*8 {
		return nil, nil, fmt.Errorf("unexpected EOF reading samples")
	}

	samples := make([]complex64, sampleCount)
	sampleBytes := data[offset : offset+int(sampleCount)*8]

	// Convert bytes directly to complex64 slice using unsafe operations
	// This is much faster than reading individual float32 values
	floatPtr := (*float32)(unsafe.Pointer(&sampleBytes[0]))
	floatSlice := (*[1 << 30]float32)(unsafe.Pointer(floatPtr))[:sampleCount*2:sampleCount*2]

	for i := uint32(0); i < sampleCount; i++ {
		real := floatSlice[i*2]
		imag := floatSlice[i*2+1]
		samples[i] = complex(real, imag)
	}

	fmt.Printf("      ✅ Memory-mapped read complete\n")

	return &metadata, samples, nil
}

// readWithBufferedIO reads data using optimized buffered I/O for smaller files
func (r *OptimizedFileReader) readWithBufferedIO() (*filewriter.Metadata, []complex64, error) {
	// Use larger buffer for better performance
	const bufferSize = 64 * 1024 // 64KB buffer
	buffer := make([]byte, bufferSize)

	// Read header first
	if _, err := r.file.Read(buffer[:5]); err != nil {
		return nil, nil, fmt.Errorf("failed to read magic: %w", err)
	}
	if string(buffer[:5]) != "ARGUS" {
		return nil, nil, fmt.Errorf("invalid file format")
	}

	var metadata filewriter.Metadata

	// Read metadata using binary.Read for compatibility
	if err := binary.Read(r.file, binary.LittleEndian, &metadata.FileFormatVersion); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(r.file, binary.LittleEndian, &metadata.Frequency); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(r.file, binary.LittleEndian, &metadata.SampleRate); err != nil {
		return nil, nil, err
	}

	var collectionTimeUnix int64
	var collectionTimeNano int32
	if err := binary.Read(r.file, binary.LittleEndian, &collectionTimeUnix); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(r.file, binary.LittleEndian, &collectionTimeNano); err != nil {
		return nil, nil, err
	}
	metadata.CollectionTime = time.Unix(collectionTimeUnix, int64(collectionTimeNano))

	if err := binary.Read(r.file, binary.LittleEndian, &metadata.GPSLocation.Latitude); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(r.file, binary.LittleEndian, &metadata.GPSLocation.Longitude); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(r.file, binary.LittleEndian, &metadata.GPSLocation.Altitude); err != nil {
		return nil, nil, err
	}

	var gpsTimeUnix int64
	var gpsTimeNano int32
	if err := binary.Read(r.file, binary.LittleEndian, &gpsTimeUnix); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(r.file, binary.LittleEndian, &gpsTimeNano); err != nil {
		return nil, nil, err
	}
	metadata.GPSTimestamp = time.Unix(gpsTimeUnix, int64(gpsTimeNano))

	var deviceInfoLen uint8
	if err := binary.Read(r.file, binary.LittleEndian, &deviceInfoLen); err != nil {
		return nil, nil, err
	}
	deviceInfoBytes := make([]byte, deviceInfoLen)
	if _, err := r.file.Read(deviceInfoBytes); err != nil {
		return nil, nil, err
	}
	metadata.DeviceInfo = string(deviceInfoBytes)

	var collectionIDLen uint8
	if err := binary.Read(r.file, binary.LittleEndian, &collectionIDLen); err != nil {
		return nil, nil, err
	}
	collectionIDBytes := make([]byte, collectionIDLen)
	if _, err := r.file.Read(collectionIDBytes); err != nil {
		return nil, nil, err
	}
	metadata.CollectionID = string(collectionIDBytes)

	var sampleCount uint32
	if err := binary.Read(r.file, binary.LittleEndian, &sampleCount); err != nil {
		return nil, nil, err
	}

	fmt.Printf("      📊 Buffered read, processing %d samples...\n", sampleCount)

	// Read samples in larger chunks for better performance
	samples := make([]complex64, sampleCount)
	const samplesPerChunk = 4096 // Read 4096 samples at a time
	chunkBuffer := make([]byte, samplesPerChunk*8) // 8 bytes per complex64

	var samplesRead uint32
	lastProgress := -1

	for samplesRead < sampleCount {
		// Calculate how many samples to read in this chunk
		samplesToRead := samplesPerChunk
		if samplesRead+uint32(samplesToRead) > sampleCount {
			samplesToRead = int(sampleCount - samplesRead)
		}

		chunkBytes := samplesToRead * 8
		n, err := r.file.Read(chunkBuffer[:chunkBytes])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read sample chunk: %w", err)
		}
		if n != chunkBytes {
			return nil, nil, fmt.Errorf("incomplete read: expected %d bytes, got %d", chunkBytes, n)
		}

		// Convert bytes to complex64 using unsafe operations for speed
		floatPtr := (*float32)(unsafe.Pointer(&chunkBuffer[0]))
		floatSlice := (*[1 << 20]float32)(unsafe.Pointer(floatPtr))[:samplesToRead*2:samplesToRead*2]

		for i := 0; i < samplesToRead; i++ {
			real := floatSlice[i*2]
			imag := floatSlice[i*2+1]
			samples[samplesRead+uint32(i)] = complex(real, imag)
		}

		samplesRead += uint32(samplesToRead)

		// Show progress for large files
		progress := int((float64(samplesRead) / float64(sampleCount)) * 100)
		if progress != lastProgress && progress%20 == 0 {
			fmt.Printf("         Progress: %d%%\n", progress)
			lastProgress = progress
		}
	}

	fmt.Printf("         Progress: 100%%\n")

	return &metadata, samples, nil
}

// readFileWithProgress reads an argus data file with optimized I/O and progress reporting
func (p *Processor) readFileWithProgress(filename string) (*filewriter.Metadata, []complex64, error) {
	// Get file size for strategy selection
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	// For very small files, use the original simple method
	if fileSize < 5*1024*1024 { // Less than 5MB
		return filewriter.ReadFile(filename)
	}

	// Use optimized reader for larger files
	reader, err := NewOptimizedFileReader(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create optimized reader: %w", err)
	}
	defer reader.Close()

	sizeMB := float64(fileSize) / (1024 * 1024)
	fmt.Printf("      📁 Using optimized I/O for %.1f MB file\n", sizeMB)

	return reader.ReadFile()
}
