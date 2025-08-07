//go:build !rtlsdr

// Package rtlsdr provides stub implementations when RTL-SDR support is not compiled in
// This file is compiled when the "rtlsdr" build tag is NOT specified
package rtlsdr

import (
	"fmt"
	"time"
)

// Device represents a stub RTL-SDR device (no actual hardware access)
type Device struct {
	frequency  uint32 // Stored frequency setting
	sampleRate uint32 // Stored sample rate setting
	gain       int    // Stored gain setting
	gainMode   string // Stored gain mode setting
	biasTee    bool   // Stored bias tee setting
	
	// Software AGC state (stub)
	agcEnabled     bool    // Software AGC enabled (stub)
	agcTargetPower float64 // Target signal power (stub)
	agcFinalGain   float64 // Final AGC gain (stub)
	
	// Logging control (stub)
	verbose        bool    // Enable verbose logging (stub)
}

// IQSample represents a stub IQ sample structure (matches real implementation)
type IQSample struct {
	Timestamp time.Time   // Time when collection would have started
	Data      []complex64 // Empty sample data
}

// NewDevice creates a stub RTL-SDR device for testing
func NewDevice(deviceIndex int) (*Device, error) {
	return &Device{
		frequency:      433920000, // Default frequency
		sampleRate:     2048000,   // Default sample rate
		gain:           207,       // Default gain (20.7 dB in tenths)
		gainMode:       "manual",  // Default to manual gain
		biasTee:        false,     // Default bias tee off
		agcTargetPower: 0.7,       // Target 70% of full scale
		agcFinalGain:   20.7,      // Default final gain
	}, nil
}

// NewDeviceBySerial creates a stub RTL-SDR device by serial number for testing
func NewDeviceBySerial(serialNumber string) (*Device, error) {
	return &Device{
		frequency:      433920000, // Default frequency
		sampleRate:     2048000,   // Default sample rate
		gain:           207,       // Default gain (20.7 dB in tenths)
		gainMode:       "manual",  // Default to manual gain
		biasTee:        false,     // Default bias tee off
		agcTargetPower: 0.7,       // Target 70% of full scale
		agcFinalGain:   20.7,      // Default final gain
	}, nil
}

// ListDevices returns stub device information for testing
func ListDevices() ([]DeviceInfo, error) {
	return []DeviceInfo{
		{
			Index:        0,
			Name:         "RTL-SDR Stub Device #0",
			Manufacturer: "Stub Corp",
			Product:      "RTL-SDR Stub",
			SerialNumber: "00000001",
		},
		{
			Index:        1,
			Name:         "RTL-SDR Stub Device #1",
			Manufacturer: "Stub Corp",
			Product:      "RTL-SDR Stub",
			SerialNumber: "00000002",
		},
	}, nil
}

// DeviceInfo contains information about a stub RTL-SDR device
type DeviceInfo struct {
	Index        int    // Device index (0-based)
	Name         string // Device name
	Manufacturer string // USB manufacturer string
	Product      string // USB product string
	SerialNumber string // USB serial number string
}

// SetFrequency stub method - stores frequency setting
func (d *Device) SetFrequency(freq uint32) error {
	d.frequency = freq
	return nil
}

// SetSampleRate stub method - stores sample rate setting with validation
func (d *Device) SetSampleRate(rate uint32) error {
	// Simulate the same validation as real implementation
	validRates := []uint32{
		250000, 1024000, 1536000, 1792000, 1920000,
		2048000, 2160000, 2560000, 2880000, 3200000,
	}

	// Check if requested rate is valid
	isValid := false
	for _, validRate := range validRates {
		if rate == validRate {
			isValid = true
			break
		}
	}

	if !isValid {
		// Find closest valid rate
		bestRate, err := d.findValidSampleRate(rate)
		if err != nil {
			return fmt.Errorf("failed to set sample rate to %d Hz: %w", rate, err)
		}
		fmt.Printf("Warning: Requested sample rate %d Hz not supported, using %d Hz instead\n", rate, bestRate)
		d.sampleRate = bestRate
	} else {
		d.sampleRate = rate
	}

	return nil
}

// findValidSampleRate finds a valid sample rate close to the requested rate (stub version)
func (d *Device) findValidSampleRate(requestedRate uint32) (uint32, error) {
	validRates := []uint32{
		250000, 1024000, 1536000, 1792000, 1920000,
		2048000, 2160000, 2560000, 2880000, 3200000,
	}

	var bestRate uint32
	var minDiff uint32 = ^uint32(0)

	for _, rate := range validRates {
		var diff uint32
		if rate > requestedRate {
			diff = rate - requestedRate
		} else {
			diff = requestedRate - rate
		}

		if diff < minDiff {
			minDiff = diff
			bestRate = rate
		}
	}

	if bestRate == 0 {
		return 0, fmt.Errorf("no valid sample rate found")
	}

	return bestRate, nil
}

// GetTunerGains stub method - returns typical RTL-SDR gains in tenths of dB
func (d *Device) GetTunerGains() ([]int, error) {
	// Typical RTL-SDR gains in tenths of dB
	return []int{0, 9, 14, 27, 37, 77, 87, 125, 144, 157, 166, 197, 207, 229, 254, 280, 297, 328, 338, 364, 372, 386, 402, 421, 434, 439, 445, 480, 496}, nil
}

// GetTunerGainsFloat stub method - returns typical RTL-SDR gains in dB as floats
func (d *Device) GetTunerGainsFloat() ([]float64, error) {
	gains, err := d.GetTunerGains()
	if err != nil {
		return nil, err
	}

	gainsFloat := make([]float64, len(gains))
	for i, gain := range gains {
		gainsFloat[i] = float64(gain) / 10.0
	}
	return gainsFloat, nil
}

// SetGain stub method - stores gain setting
func (d *Device) SetGain(gain float64) error {
	d.gain = int(gain * 10) // Store in tenths of dB
	d.gainMode = "manual"
	return nil
}

// SetGainMode stub method - stores gain mode setting
func (d *Device) SetGainMode(mode string) error {
	switch mode {
	case "auto":
		d.gainMode = mode
		d.agcEnabled = true
		d.agcFinalGain = 24.8 // Simulate AGC final gain for stub
		if d.verbose {
			fmt.Printf("Software AGC enabled (stub mode - target: %.1f%%, simulating gain adjustments)\n", d.agcTargetPower*100)
		}
		return nil
	case "manual":
		d.gainMode = mode
		d.agcEnabled = false
		return nil
	default:
		return fmt.Errorf("invalid gain mode: %s (must be 'auto' or 'manual')", mode)
	}
}

// EnableAGC stub method - compatibility wrapper
// Deprecated: Use SetGainMode instead
func (d *Device) EnableAGC(enable bool) error {
	if enable {
		return d.SetGainMode("auto")
	}
	return d.SetGainMode("manual")
}

// GetGain stub method - returns current gain in dB
func (d *Device) GetGain() float64 {
	return float64(d.gain) / 10.0
}

// GetGainMode stub method - returns current gain mode
func (d *Device) GetGainMode() string {
	return d.gainMode
}

// SetVerbose stub method - enables or disables verbose logging
func (d *Device) SetVerbose(verbose bool) {
	d.verbose = verbose
}

// GetFinalAGCGain stub method - returns the final gain value determined by AGC
func (d *Device) GetFinalAGCGain() float64 {
	return d.agcFinalGain
}

// ReportAGCResult stub method - reports the final AGC result
func (d *Device) ReportAGCResult() {
	if d.agcEnabled && d.gainMode == "auto" {
		fmt.Printf("AGC converged to %.1f dB gain\n", d.agcFinalGain)
	}
}

// SetBiasTee stub method - stores bias tee setting
func (d *Device) SetBiasTee(enable bool) error {
	d.biasTee = enable
	return nil
}

// GetDeviceInfo stub method - returns mock device info
func (d *Device) GetDeviceInfo() (string, error) {
	biasStatus := "off"
	if d.biasTee {
		biasStatus = "on"
	}

	gainInfo := fmt.Sprintf("%.1f dB (%s)", float64(d.gain)/10, d.gainMode)

	return fmt.Sprintf("RTL-SDR Stub Device (freq: %d Hz, rate: %d Hz, gain: %s, bias-tee: %s)",
		d.frequency, d.sampleRate, gainInfo, biasStatus), nil
}

// StartCollection stub method - simulates collection for testing with proper timeout handling
func (d *Device) StartCollection(duration time.Duration, samplesChan chan<- IQSample) error {
	startTime := time.Now()

	// Generate fake sample data for testing
	totalSamples := int(d.sampleRate * uint32(duration.Seconds()))
	fakeSamples := make([]complex64, totalSamples)

	// Fill with simple test pattern
	for i := range fakeSamples {
		fakeSamples[i] = complex(0.1, 0.1) // Simple test signal
	}

	// Simulate the real hardware behavior: collect for the duration, then send data
	// This matches how the real RTL-SDR works - it collects samples over time
	time.Sleep(duration)

	// Send the fake samples after collection completes (like real hardware)
	select {
	case samplesChan <- IQSample{
		Timestamp: startTime,
		Data:      fakeSamples,
	}:
		return nil
	default:
		return fmt.Errorf("samples channel is full")
	}
}

// Close stub method - no-op for stub implementation
func (d *Device) Close() error {
	return nil
}
