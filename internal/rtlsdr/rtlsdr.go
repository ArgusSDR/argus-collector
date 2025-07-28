//go:build rtlsdr

// Package rtlsdr provides RTL-SDR device interface for signal collection
// This file is only compiled when the "rtlsdr" build tag is specified
package rtlsdr

import (
	"context"
	"fmt"
	"time"

	"github.com/jpoirier/gortlsdr"
)

// Device represents an RTL-SDR device and its configuration
type Device struct {
	dev        *rtlsdr.Context // RTL-SDR device context
	frequency  uint32          // Current tuned frequency in Hz
	sampleRate uint32          // Current sample rate in Hz
	gain       int             // Current gain in tenths of dB
}

// IQSample represents a collected set of IQ samples with timestamp
type IQSample struct {
	Timestamp time.Time    // Time when collection started
	Data      []complex64  // IQ sample data (I=real, Q=imaginary)
}

// NewDevice creates a new RTL-SDR device instance
// deviceIndex: 0-based index of RTL-SDR device to open
func NewDevice(deviceIndex int) (*Device, error) {
	// Check if any RTL-SDR devices are connected
	count := rtlsdr.GetDeviceCount()
	if count == 0 {
		return nil, fmt.Errorf("no RTL-SDR devices found")
	}
	
	// Validate device index is within range
	if deviceIndex >= count {
		return nil, fmt.Errorf("device index %d out of range (found %d devices)", deviceIndex, count)
	}

	// Open the specified RTL-SDR device
	dev, err := rtlsdr.Open(deviceIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to open RTL-SDR device: %w", err)
	}

	return &Device{dev: dev}, nil
}

// SetFrequency sets the center frequency of the RTL-SDR device
// freq: frequency in Hz
func (d *Device) SetFrequency(freq uint32) error {
	if err := d.dev.SetCenterFreq(int(freq)); err != nil {
		return fmt.Errorf("failed to set frequency to %d Hz: %w", freq, err)
	}
	d.frequency = freq
	return nil
}

// SetSampleRate sets the sample rate of the RTL-SDR device
// rate: sample rate in Hz (samples per second)
func (d *Device) SetSampleRate(rate uint32) error {
	if err := d.dev.SetSampleRate(int(rate)); err != nil {
		return fmt.Errorf("failed to set sample rate to %d Hz: %w", rate, err)
	}
	d.sampleRate = rate
	return nil
}

// SetGain sets the tuner gain of the RTL-SDR device
// gain: gain in dB (decibels)
func (d *Device) SetGain(gain float64) error {
	// Convert gain from dB to tenths of dB (RTL-SDR API requirement)
	gainTenthsDB := int(gain * 10)
	if err := d.dev.SetTunerGain(gainTenthsDB); err != nil {
		return fmt.Errorf("failed to set gain to %.1f dB: %w", gain, err)
	}
	d.gain = gainTenthsDB
	return nil
}

// EnableAGC enables or disables automatic gain control
// enable: true to enable AGC, false to use manual gain
func (d *Device) EnableAGC(enable bool) error {
	// Note: SetTunerGainMode expects false for AGC enabled
	if err := d.dev.SetTunerGainMode(!enable); err != nil {
		return fmt.Errorf("failed to set AGC mode: %w", err)
	}
	return nil
}

// GetDeviceInfo returns a formatted string with device information
func (d *Device) GetDeviceInfo() (string, error) {
	// Get device name from USB strings
	name, _, _, err := rtlsdr.GetDeviceUsbStrings(0)
	if err != nil {
		return "", fmt.Errorf("failed to get device info: %w", err)
	}
	// Format device info with current settings
	return fmt.Sprintf("%s (freq: %d Hz, rate: %d Hz, gain: %.1f dB)", 
		name, d.frequency, d.sampleRate, float64(d.gain)/10), nil
}

// StartCollection collects IQ samples from RTL-SDR for specified duration
// duration: how long to collect samples
// samplesChan: channel to send collected samples to
func (d *Device) StartCollection(duration time.Duration, samplesChan chan<- IQSample) error {
	// Create context with timeout to ensure collection stops
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	// Reset RTL-SDR buffer to ensure clean start
	if err := d.dev.ResetBuffer(); err != nil {
		return fmt.Errorf("failed to reset buffer: %w", err)
	}

	// Calculate total samples needed (2 bytes per complex sample)
	totalSamples := int(d.sampleRate * uint32(duration.Seconds()))
	chunkSize := 262144 // 256KB chunks for memory efficiency
	if chunkSize > totalSamples*2 {
		chunkSize = totalSamples * 2
	}

	// Pre-allocate slice for all samples
	allSamples := make([]complex64, 0, totalSamples)
	buffer := make([]uint8, chunkSize)
	
	startTime := time.Now()
	totalRead := 0
	
	// Read samples in chunks to manage memory usage
	for totalRead < totalSamples*2 {
		// Check if context has been cancelled (timeout reached)
		select {
		case <-ctx.Done():
			// Context cancelled, stop collection
			break
		default:
		}
		
		remaining := totalSamples*2 - totalRead
		readSize := chunkSize
		if readSize > remaining {
			readSize = remaining
		}
		
		// Read raw IQ data from RTL-SDR with timeout protection
		// Note: ReadSync is blocking, but we check context between calls
		nRead, err := d.dev.ReadSync(buffer[:readSize], readSize)
		if err != nil {
			return fmt.Errorf("failed to read samples: %w", err)
		}
		
		if nRead == 0 {
			break
		}

		// Convert raw bytes to complex64 samples
		// RTL-SDR provides unsigned 8-bit IQ pairs (I,Q,I,Q...)
		for i := 0; i < nRead; i += 2 {
			if i+1 < nRead {
				// Convert unsigned 8-bit to signed float [-1.0, 1.0]
				i_val := (float32(buffer[i]) - 127.5) / 127.5
				q_val := (float32(buffer[i+1]) - 127.5) / 127.5
				allSamples = append(allSamples, complex(i_val, q_val))
			}
		}
		
		totalRead += nRead
	}

	// Send collected samples through channel
	select {
	case samplesChan <- IQSample{
		Timestamp: startTime,
		Data:      allSamples,
	}:
	default:
		return fmt.Errorf("samples channel is full")
	}

	return nil
}

// Close properly closes the RTL-SDR device and releases resources
func (d *Device) Close() error {
	if d.dev != nil {
		return d.dev.Close()
	}
	return nil
}