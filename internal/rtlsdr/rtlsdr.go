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

// NewDeviceBySerial creates a new RTL-SDR device instance by serial number
// serialNumber: serial number string of the device to open
func NewDeviceBySerial(serialNumber string) (*Device, error) {
	// Check if any RTL-SDR devices are connected
	count := rtlsdr.GetDeviceCount()
	if count == 0 {
		return nil, fmt.Errorf("no RTL-SDR devices found")
	}
	
	// Search for device with matching serial number
	for i := 0; i < count; i++ {
		// Get device USB strings (manufacturer, product, serial)
		_, _, serial, err := rtlsdr.GetDeviceUsbStrings(i)
		if err != nil {
			continue // Skip devices we can't query
		}
		
		// Check if serial number matches
		if serial == serialNumber {
			// Open the matching device
			dev, err := rtlsdr.Open(i)
			if err != nil {
				return nil, fmt.Errorf("failed to open RTL-SDR device with serial %s: %w", serialNumber, err)
			}
			return &Device{dev: dev}, nil
		}
	}
	
	return nil, fmt.Errorf("no RTL-SDR device found with serial number: %s", serialNumber)
}

// ListDevices returns information about all available RTL-SDR devices
func ListDevices() ([]DeviceInfo, error) {
	count := rtlsdr.GetDeviceCount()
	if count == 0 {
		return nil, fmt.Errorf("no RTL-SDR devices found")
	}
	
	devices := make([]DeviceInfo, 0, count)
	for i := 0; i < count; i++ {
		// Get device USB strings
		manufacturer, product, serial, err := rtlsdr.GetDeviceUsbStrings(i)
		if err != nil {
			// If we can't get USB strings, use device name
			name := rtlsdr.GetDeviceName(i)
			devices = append(devices, DeviceInfo{
				Index:        i,
				Name:         name,
				Manufacturer: "Unknown",
				Product:      "Unknown", 
				SerialNumber: "Unknown",
			})
			continue
		}
		
		devices = append(devices, DeviceInfo{
			Index:        i,
			Name:         rtlsdr.GetDeviceName(i),
			Manufacturer: manufacturer,
			Product:      product,
			SerialNumber: serial,
		})
	}
	
	return devices, nil
}

// DeviceInfo contains information about an RTL-SDR device
type DeviceInfo struct {
	Index        int    // Device index (0-based)
	Name         string // Device name
	Manufacturer string // USB manufacturer string
	Product      string // USB product string  
	SerialNumber string // USB serial number string
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
	// Try the requested rate first
	if err := d.dev.SetSampleRate(int(rate)); err != nil {
		// If the requested rate fails, try to find a valid rate
		validRate, fallbackErr := d.findValidSampleRate(rate)
		if fallbackErr != nil {
			return fmt.Errorf("failed to set sample rate to %d Hz and no valid fallback found: %w", rate, err)
		}
		
		// Try the valid rate
		if err := d.dev.SetSampleRate(int(validRate)); err != nil {
			return fmt.Errorf("failed to set sample rate to %d Hz (tried fallback %d Hz): %w", rate, validRate, err)
		}
		
		fmt.Printf("Warning: Requested sample rate %d Hz not supported, using %d Hz instead\n", rate, validRate)
		d.sampleRate = validRate
		return nil
	}
	
	d.sampleRate = rate
	return nil
}

// findValidSampleRate finds a valid sample rate close to the requested rate
func (d *Device) findValidSampleRate(requestedRate uint32) (uint32, error) {
	// Common RTL-SDR supported sample rates (in Hz)
	validRates := []uint32{
		250000,   // 250 kHz
		1024000,  // 1.024 MHz  
		1536000,  // 1.536 MHz
		1792000,  // 1.792 MHz
		1920000,  // 1.92 MHz
		2048000,  // 2.048 MHz
		2160000,  // 2.16 MHz
		2560000,  // 2.56 MHz
		2880000,  // 2.88 MHz
		3200000,  // 3.2 MHz (maximum for most devices)
	}
	
	// Find the closest valid rate
	var bestRate uint32
	var minDiff uint32 = ^uint32(0) // Max uint32
	
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