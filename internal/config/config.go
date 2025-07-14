// Package config provides configuration structures and defaults for Argus Collector
package config

import (
	"time"
)

// Config represents the complete application configuration
type Config struct {
	RTLSDR     RTLSDRConfig     `yaml:"rtlsdr"`     // RTL-SDR device settings
	GPS        GPSConfig        `yaml:"gps"`        // GPS receiver settings
	Collection CollectionConfig `yaml:"collection"` // Data collection settings
	Logging    LoggingConfig    `yaml:"logging"`    // Logging configuration
}

// RTLSDRConfig contains RTL-SDR device configuration parameters
type RTLSDRConfig struct {
	Frequency   float64 `yaml:"frequency"`    // RF frequency in Hz
	SampleRate  uint32  `yaml:"sample_rate"`  // Sample rate in Hz
	Gain        float64 `yaml:"gain"`         // RF gain in dB
	DeviceIndex int     `yaml:"device_index"` // RTL-SDR device index (0-based)
}

// GPSConfig contains GPS receiver configuration parameters
type GPSConfig struct {
	Port            string        `yaml:"port"`             // Serial port device path
	BaudRate        int           `yaml:"baud_rate"`        // Serial communication baud rate
	Timeout         time.Duration `yaml:"timeout"`          // Timeout for GPS fix acquisition
	Disable         bool          `yaml:"disable"`          // Disable GPS hardware and use manual coordinates
	ManualLatitude  float64       `yaml:"manual_latitude"`  // Manual latitude in decimal degrees
	ManualLongitude float64       `yaml:"manual_longitude"` // Manual longitude in decimal degrees
	ManualAltitude  float64       `yaml:"manual_altitude"`  // Manual altitude in meters
}

// CollectionConfig contains data collection configuration parameters
type CollectionConfig struct {
	Duration    time.Duration `yaml:"duration"`     // Collection duration
	OutputDir   string        `yaml:"output_dir"`   // Output directory for data files
	FilePrefix  string        `yaml:"file_prefix"`  // Prefix for output filenames
	SyncedStart bool          `yaml:"synced_start"` // Enable synchronized start timing
}

// LoggingConfig contains logging configuration parameters
type LoggingConfig struct {
	Level string `yaml:"level"` // Log level (debug, info, warn, error)
	File  string `yaml:"file"`  // Log file path
}

// DefaultConfig returns a configuration with sensible default values
func DefaultConfig() *Config {
	return &Config{
		RTLSDR: RTLSDRConfig{
			Frequency:   433.92e6, // 433.92 MHz ISM band
			SampleRate:  2048000,  // 2.048 MSps
			Gain:        20.7,     // 20.7 dB gain
			DeviceIndex: 0,        // First RTL-SDR device
		},
		GPS: GPSConfig{
			Port:            "/dev/ttyUSB0",    // Common USB GPS device path
			BaudRate:        9600,             // Standard NMEA baud rate
			Timeout:         30 * time.Second, // 30 second GPS fix timeout
			Disable:         false,            // GPS enabled by default
			ManualLatitude:  0.0,              // Default latitude (equator)
			ManualLongitude: 0.0,              // Default longitude (prime meridian)
			ManualAltitude:  0.0,              // Default altitude (sea level)
		},
		Collection: CollectionConfig{
			Duration:    60 * time.Second, // 60 second collection duration
			OutputDir:   "./data",         // Current directory data folder
			FilePrefix:  "argus",          // File prefix for output files
			SyncedStart: true,             // Enable synchronized start by default
		},
		Logging: LoggingConfig{
			Level: "info",      // Info level logging
			File:  "argus.log", // Log to argus.log file
		},
	}
}