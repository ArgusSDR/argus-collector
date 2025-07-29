//go:build !rtlsdr

// Package rtlsdr provides stub implementations when RTL-SDR support is not compiled in
// This file is compiled when the "rtlsdr" build tag is NOT specified
package rtlsdr

import (
	"context"
	"fmt"
	"time"
)

// Device represents a stub RTL-SDR device (no actual hardware access)
type Device struct {
	frequency  uint32 // Stored frequency setting
	sampleRate uint32 // Stored sample rate setting
	gain       int    // Stored gain setting
	biasTee    bool   // Stored bias tee setting
}

// IQSample represents a stub IQ sample structure (matches real implementation)
type IQSample struct {
	Timestamp time.Time    // Time when collection would have started
	Data      []complex64  // Empty sample data
}

// NewDevice creates a stub RTL-SDR device for testing
func NewDevice(deviceIndex int) (*Device, error) {
	return &Device{
		frequency:  433920000, // Default frequency
		sampleRate: 2048000,   // Default sample rate
		gain:       20,        // Default gain
		biasTee:    false,     // Default bias tee off
	}, nil
}

// NewDeviceBySerial creates a stub RTL-SDR device by serial number for testing
func NewDeviceBySerial(serialNumber string) (*Device, error) {
	return &Device{
		frequency:  433920000, // Default frequency
		sampleRate: 2048000,   // Default sample rate
		gain:       20,        // Default gain
		biasTee:    false,     // Default bias tee off
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

// SetGain stub method - stores gain setting
func (d *Device) SetGain(gain float64) error {
	d.gain = int(gain * 10) // Store in tenths of dB
	return nil
}

// EnableAGC stub method - no-op for stub implementation
func (d *Device) EnableAGC(enable bool) error {
	return nil
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
	return fmt.Sprintf("RTL-SDR Stub Device (freq: %d Hz, rate: %d Hz, gain: %.1f dB, bias-tee: %s)", 
		d.frequency, d.sampleRate, float64(d.gain)/10, biasStatus), nil
}

// StartCollection stub method - simulates collection for testing with proper timeout handling
func (d *Device) StartCollection(duration time.Duration, samplesChan chan<- IQSample) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	startTime := time.Now()
	
	// Generate fake sample data for testing  
	totalSamples := int(d.sampleRate * uint32(duration.Seconds()))
	fakeSamples := make([]complex64, totalSamples)
	
	// Fill with simple test pattern
	for i := range fakeSamples {
		fakeSamples[i] = complex(0.1, 0.1) // Simple test signal
	}
	
	// Wait for the requested duration or until cancelled
	select {
	case <-ctx.Done():
		// Duration expired - this is the normal case
	case <-time.After(duration + time.Second):
		// Safety timeout in case context doesn't work
		return fmt.Errorf("stub collection timeout exceeded")
	}
	
	// Send the fake samples
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