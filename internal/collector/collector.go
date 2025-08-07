package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"argus-collector/internal/config"
	"argus-collector/internal/filewriter"
	"argus-collector/internal/gps"
	"argus-collector/internal/rtlsdr"
)

type Collector struct {
	config   *config.Config
	rtlsdr   *rtlsdr.Device
	gps      *gps.GPS
	writer   *filewriter.Writer
	stopChan chan struct{}
	wg       sync.WaitGroup
}

type CollectionData struct {
	IQSamples    rtlsdr.IQSample
	GPSPosition  gps.Position
	CollectionID string
}

func NewCollector(cfg *config.Config) *Collector {
	return &Collector{
		config:   cfg,
		stopChan: make(chan struct{}),
	}
}

func (c *Collector) Initialize() error {
	var err error

	// Choose device selection method based on configuration
	if c.config.RTLSDR.SerialNumber != "" {
		// Use serial number to select device
		c.rtlsdr, err = rtlsdr.NewDeviceBySerial(c.config.RTLSDR.SerialNumber)
		if err != nil {
			return fmt.Errorf("failed to initialize RTL-SDR by serial %s: %w", c.config.RTLSDR.SerialNumber, err)
		}
	} else {
		// Fall back to device index
		c.rtlsdr, err = rtlsdr.NewDevice(c.config.RTLSDR.DeviceIndex)
		if err != nil {
			return fmt.Errorf("failed to initialize RTL-SDR by index %d: %w", c.config.RTLSDR.DeviceIndex, err)
		}
	}

	if err := c.rtlsdr.SetFrequency(uint32(c.config.RTLSDR.Frequency)); err != nil {
		return fmt.Errorf("failed to set RTL-SDR frequency: %w", err)
	}

	if err := c.rtlsdr.SetSampleRate(c.config.RTLSDR.SampleRate); err != nil {
		return fmt.Errorf("failed to set RTL-SDR sample rate: %w", err)
	}

	// Set gain mode first
	if err := c.rtlsdr.SetGainMode(c.config.RTLSDR.GainMode); err != nil {
		return fmt.Errorf("failed to set RTL-SDR gain mode: %w", err)
	}

	// Set manual gain if in manual mode
	if c.config.RTLSDR.GainMode == "manual" {
		if err := c.rtlsdr.SetGain(c.config.RTLSDR.Gain); err != nil {
			return fmt.Errorf("failed to set RTL-SDR gain: %w", err)
		}
	}

	// Set bias tee if enabled
	if err := c.rtlsdr.SetBiasTee(c.config.RTLSDR.BiasTee); err != nil {
		return fmt.Errorf("failed to set RTL-SDR bias tee: %w", err)
	}

	// Initialize GPS based on mode
	gpsMode := c.config.GPS.Mode
	// Handle backward compatibility with deprecated Disable flag
	if c.config.GPS.Disable {
		gpsMode = "manual"
	}

	switch gpsMode {
	case "nmea":
		c.gps, err = gps.NewGPS(c.config.GPS.Port, c.config.GPS.BaudRate)
		if err != nil {
			return fmt.Errorf("failed to initialize NMEA GPS: %w", err)
		}
		// Enable debug mode if verbose or debug logging is enabled
		if c.config.Logging.Level == "debug" {
			c.gps.SetDebug(true)
		}
		if err := c.gps.Start(); err != nil {
			return fmt.Errorf("failed to start NMEA GPS: %w", err)
		}
	case "gpsd":
		c.gps, err = gps.NewGPSD(c.config.GPS.GPSDHost, c.config.GPS.GPSDPort)
		if err != nil {
			return fmt.Errorf("failed to initialize GPSD: %w", err)
		}
		if err := c.gps.Start(); err != nil {
			return fmt.Errorf("failed to start GPSD: %w", err)
		}
	case "manual":
		// GPS disabled, will use manual coordinates
		c.gps = nil
	default:
		return fmt.Errorf("invalid GPS mode: %s (must be 'nmea', 'gpsd', or 'manual')", gpsMode)
	}

	if err := os.MkdirAll(c.config.Collection.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	c.writer = filewriter.NewWriter()

	return nil
}

func (c *Collector) WaitForGPSFix() error {
	return c.WaitForGPSFixWithContext(context.Background())
}

func (c *Collector) WaitForGPSFixWithContext(ctx context.Context) error {
	gpsMode := c.config.GPS.Mode
	if c.config.GPS.Disable {
		gpsMode = "manual"
	}

	if gpsMode == "manual" {
		// GPS is disabled, use manual coordinates
		fmt.Printf("GPS disabled - using manual coordinates: %.8f°, %.8f°\n",
			c.config.GPS.ManualLatitude, c.config.GPS.ManualLongitude)
		return nil
	}

	fmt.Printf("Waiting for GPS fix via %s (timeout: %v)...\n", gpsMode, c.config.GPS.Timeout)

	// Create a channel for GPS fix result
	type gpsResult struct {
		pos *gps.Position
		err error
	}

	gpsResultChan := make(chan gpsResult, 1)
	go func() {
		pos, err := c.gps.WaitForFix(c.config.GPS.Timeout)
		gpsResultChan <- gpsResult{pos, err}
	}()

	// Wait for GPS fix or context cancellation
	var position *gps.Position
	select {
	case result := <-gpsResultChan:
		if result.err != nil {
			return fmt.Errorf("GPS fix failed: %w", result.err)
		}
		position = result.pos
	case <-ctx.Done():
		return fmt.Errorf("GPS fix cancelled: %w", ctx.Err())
	}

	fmt.Printf("GPS fix acquired: %.6f, %.6f (quality: %s, satellites: %d)\n",
		position.Latitude, position.Longitude,
		c.gps.GetFixQualityString(), position.Satellites)

	return nil
}

func (c *Collector) Collect() error {
	return c.CollectWithContext(context.Background())
}

func (c *Collector) CollectWithContext(ctx context.Context) error {
	var startTime time.Time

	if c.config.Collection.StartTime > 0 {
		// Use exact epoch timestamp from --start-time
		startTime = time.Unix(c.config.Collection.StartTime, 0)
		fmt.Printf("Exact start time specified - waiting until: %s\n", startTime.Format("15:04:05.000"))

		waitDuration := time.Until(startTime)
		if waitDuration > 0 {
			// Wait for exact start time or context cancellation
			select {
			case <-time.After(waitDuration):
				// Normal exact time wait completed
			case <-ctx.Done():
				return fmt.Errorf("exact start time cancelled: %w", ctx.Err())
			}
		} else if waitDuration < -10*time.Second {
			return fmt.Errorf("start time is too far in the past: %s", startTime.Format("15:04:05.000"))
		}
	} else if c.config.Collection.SyncedStart {
		startTime = c.calculateSyncedStartTime()
		fmt.Printf("Synchronized start enabled - waiting until: %s\n", startTime.Format("15:04:05.000"))

		waitDuration := time.Until(startTime)
		if waitDuration > 0 {

			// Wait for sync start time or context cancellation
			select {
			case <-time.After(waitDuration):
				// Normal sync wait completed
			case <-ctx.Done():
				return fmt.Errorf("synchronized start cancelled: %w", ctx.Err())
			}
		}
	} else {
		fmt.Printf("Synchronized start disabled - starting immediately\n")
		startTime = time.Now()
	}

	// Generate collection ID based on configuration
	var collectionID string
	if c.config.Collection.CollectionID != "" {
		// Use configured collection ID with timestamp suffix
		collectionID = fmt.Sprintf("%s_%d", c.config.Collection.CollectionID, startTime.Unix())
	} else {
		// Generate collection ID using file prefix and device identifier
		deviceID := c.getDeviceIdentifier()
		collectionID = fmt.Sprintf("%s-%s_%d", c.config.Collection.FilePrefix, deviceID, startTime.Unix())
	}

	fmt.Printf("Starting collection (ID: %s, Duration: %v)\n", collectionID, c.config.Collection.Duration)
	// Calculate timeout buffer: 3.2x the collection duration
	totalTimeout := time.Duration(float64(c.config.Collection.Duration) * 3.2)

	deviceInfo, err := c.rtlsdr.GetDeviceInfo()
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}
	fmt.Printf("Device: %s\n", deviceInfo)

	samplesChan := make(chan rtlsdr.IQSample, 1)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.rtlsdr.StartCollection(c.config.Collection.Duration, samplesChan); err != nil {
			fmt.Printf("RTL-SDR collection error: %v\n", err)
		}
		// Always close the samples channel when collection ends
		close(samplesChan)
	}()

	// Create a done channel to coordinate goroutine completion
	done := make(chan error, 1)

	// Handle the collection result in a separate goroutine
	go func() {
		select {
		case samples, ok := <-samplesChan:
			if !ok {
				done <- fmt.Errorf("RTL-SDR collection channel closed without data")
				return
			}
			var gpsPosition gps.Position

			gpsMode := c.config.GPS.Mode
			if c.config.GPS.Disable {
				gpsMode = "manual"
			}

			if gpsMode == "manual" {
				// Use manual coordinates when GPS is disabled
				gpsPosition = gps.Position{
					Latitude:   c.config.GPS.ManualLatitude,
					Longitude:  c.config.GPS.ManualLongitude,
					Altitude:   c.config.GPS.ManualAltitude,
					Timestamp:  time.Now(),
					FixQuality: 1, // Indicate valid fix for manual coordinates
					Satellites: 0, // No satellites for manual coordinates
				}
			} else {
				// Get position from GPS hardware (nmea or gpsd)
				gpsPos, err := c.gps.GetCurrentPosition()
				if err != nil {
					done <- fmt.Errorf("failed to get GPS position: %w", err)
					return
				}
				gpsPosition = *gpsPos
			}

			collectionData := CollectionData{
				IQSamples:    samples,
				GPSPosition:  gpsPosition,
				CollectionID: collectionID,
			}

			filename := filepath.Join(c.config.Collection.OutputDir, collectionID+".dat")
			if err := c.saveData(filename, collectionData); err != nil {
				done <- fmt.Errorf("failed to save data: %w", err)
				return
			}

			fmt.Printf("Collection saved to: %s\n", filename)
			fmt.Printf("Samples collected: %d\n", len(samples.Data))
			done <- nil

		case <-time.After(c.config.Collection.Duration + 10*time.Second):
			done <- fmt.Errorf("collection timeout - no data received from RTL-SDR")
		}
	}()

	// Wait for either successful completion, timeout, or context cancellation
	// Use the same timeout buffer calculation as above for display
	select {
	case err := <-done:
		if err != nil {
			return err
		}
	case <-time.After(totalTimeout):
		return fmt.Errorf("collection timeout - exceeded maximum wait time")
	case <-ctx.Done():
		return fmt.Errorf("collection cancelled: %w", ctx.Err())
	}

	// Wait for RTL-SDR goroutine to finish, but with a timeout
	waitDone := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		// RTL-SDR goroutine completed normally
	case <-time.After(5 * time.Second):
		// RTL-SDR goroutine is taking too long, proceed anyway
		fmt.Printf("Warning: RTL-SDR collection goroutine did not complete in time\n")
	case <-ctx.Done():
		// Context cancelled during cleanup
		fmt.Printf("Warning: Cleanup cancelled, forcing exit\n")
		return fmt.Errorf("cleanup cancelled: %w", ctx.Err())
	}

	return nil
}

func (c *Collector) saveData(filename string, data CollectionData) error {
	// Get actual device information including gain settings
	deviceInfo, err := c.rtlsdr.GetDeviceInfo()
	if err != nil {
		// Fallback to basic device info if GetDeviceInfo fails
		deviceInfo = fmt.Sprintf("RTL-SDR Device %d", c.config.RTLSDR.DeviceIndex)
	}

	metadata := filewriter.Metadata{
		Frequency:      uint64(c.config.RTLSDR.Frequency),
		SampleRate:     c.config.RTLSDR.SampleRate,
		CollectionTime: data.IQSamples.Timestamp,
		GPSLocation: filewriter.GPSLocation{
			Latitude:  data.GPSPosition.Latitude,
			Longitude: data.GPSPosition.Longitude,
			Altitude:  data.GPSPosition.Altitude,
		},
		GPSTimestamp:      data.GPSPosition.Timestamp,
		DeviceInfo:        deviceInfo,
		FileFormatVersion: 1,
		CollectionID:      data.CollectionID,
	}

	return c.writer.WriteFile(filename, metadata, data.IQSamples.Data)
}

// getDeviceIdentifier returns a device identifier for use in filenames
// Prefers serial number if available, otherwise uses device index
func (c *Collector) getDeviceIdentifier() string {
	if c.config.RTLSDR.SerialNumber != "" {
		// Use serial number if available (clean it for filename safety)
		serialClean := strings.ReplaceAll(c.config.RTLSDR.SerialNumber, " ", "")
		return serialClean
	}
	// Use device index (but handle -1 case when serial number should be used)
	if c.config.RTLSDR.DeviceIndex >= 0 {
		return fmt.Sprintf("%d", c.config.RTLSDR.DeviceIndex)
	}
	// Fallback for invalid configurations
	return "unknown"
}

func (c *Collector) calculateSyncedStartTime() time.Time {
	now := time.Now()
	currentEpoch := now.Unix()

	// Improved algorithm: Use fixed 100-second epochs with predetermined sync point
	// This eliminates race conditions when stations start at different times
	
	// Calculate next 100-second boundary with 30-second buffer
	syncEpoch := ((currentEpoch + 30) / 100 + 1) * 100
	
	// Use fixed sync point at 30 seconds past the epoch boundary
	// This provides predictable timing and eliminates race conditions
	syncPoint := int64(30)
	targetTime := syncEpoch + syncPoint

	// Ensure we have at least 10 seconds preparation time
	if targetTime - currentEpoch < 10 {
		targetTime += 100  // Add another 100-second epoch
	}

	return time.Unix(targetTime, 0)
}

func (c *Collector) Stop() {
	close(c.stopChan)
	c.wg.Wait()
}

// SetGPSDebug enables or disables GPS debug logging
func (c *Collector) SetGPSDebug(debug bool) {
	if c.gps != nil {
		c.gps.SetDebug(debug)
	}
}

// SetRTLSDRVerbose enables or disables RTL-SDR verbose logging
func (c *Collector) SetRTLSDRVerbose(verbose bool) {
	if c.rtlsdr != nil {
		c.rtlsdr.SetVerbose(verbose)
	}
}

// ReportAGCResult reports the final AGC gain if AGC was used
func (c *Collector) ReportAGCResult() {
	if c.rtlsdr != nil {
		c.rtlsdr.ReportAGCResult()
	}
}

func (c *Collector) Close() error {
	c.Stop()

	var errors []error

	if c.rtlsdr != nil {
		if err := c.rtlsdr.Close(); err != nil {
			errors = append(errors, fmt.Errorf("RTL-SDR close error: %w", err))
		}
	}

	if c.gps != nil {
		if err := c.gps.Close(); err != nil {
			errors = append(errors, fmt.Errorf("GPS close error: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}
