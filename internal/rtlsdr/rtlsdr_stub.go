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
	}, nil
}

// SetFrequency stub method - stores frequency setting
func (d *Device) SetFrequency(freq uint32) error {
	d.frequency = freq
	return nil
}

// SetSampleRate stub method - stores sample rate setting
func (d *Device) SetSampleRate(rate uint32) error {
	d.sampleRate = rate
	return nil
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

// GetDeviceInfo stub method - returns mock device info
func (d *Device) GetDeviceInfo() (string, error) {
	return fmt.Sprintf("RTL-SDR Stub Device (freq: %d Hz, rate: %d Hz, gain: %.1f dB)", 
		d.frequency, d.sampleRate, float64(d.gain)/10), nil
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