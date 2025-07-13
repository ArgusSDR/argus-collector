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
}

// IQSample represents a stub IQ sample structure (matches real implementation)
type IQSample struct {
	Timestamp time.Time    // Time when collection would have started
	Data      []complex64  // Empty sample data
}

// NewDevice returns an error indicating RTL-SDR support is not compiled in
func NewDevice(deviceIndex int) (*Device, error) {
	return nil, fmt.Errorf("RTL-SDR support not compiled in. Install librtlsdr-dev and rebuild with '-tags rtlsdr'")
}

// SetFrequency stub method - returns error indicating RTL-SDR not available
func (d *Device) SetFrequency(freq uint32) error {
	return fmt.Errorf("RTL-SDR not available")
}

// SetSampleRate stub method - returns error indicating RTL-SDR not available
func (d *Device) SetSampleRate(rate uint32) error {
	return fmt.Errorf("RTL-SDR not available")
}

// SetGain stub method - returns error indicating RTL-SDR not available
func (d *Device) SetGain(gain float64) error {
	return fmt.Errorf("RTL-SDR not available")
}

// EnableAGC stub method - returns error indicating RTL-SDR not available
func (d *Device) EnableAGC(enable bool) error {
	return fmt.Errorf("RTL-SDR not available")
}

// GetDeviceInfo stub method - returns error indicating RTL-SDR not available
func (d *Device) GetDeviceInfo() (string, error) {
	return "", fmt.Errorf("RTL-SDR not available")
}

// StartCollection stub method - returns error indicating RTL-SDR not available
func (d *Device) StartCollection(duration time.Duration, samplesChan chan<- IQSample) error {
	return fmt.Errorf("RTL-SDR not available")
}

// Close stub method - no-op for stub implementation
func (d *Device) Close() error {
	return nil
}