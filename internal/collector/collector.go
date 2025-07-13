package collector

import (
	"fmt"
	"os"
	"path/filepath"
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

	c.rtlsdr, err = rtlsdr.NewDevice(c.config.RTLSDR.DeviceIndex)
	if err != nil {
		return fmt.Errorf("failed to initialize RTL-SDR: %w", err)
	}

	if err := c.rtlsdr.SetFrequency(uint32(c.config.RTLSDR.Frequency)); err != nil {
		return fmt.Errorf("failed to set RTL-SDR frequency: %w", err)
	}

	if err := c.rtlsdr.SetSampleRate(c.config.RTLSDR.SampleRate); err != nil {
		return fmt.Errorf("failed to set RTL-SDR sample rate: %w", err)
	}

	if err := c.rtlsdr.SetGain(c.config.RTLSDR.Gain); err != nil {
		return fmt.Errorf("failed to set RTL-SDR gain: %w", err)
	}

	c.gps, err = gps.NewGPS(c.config.GPS.Port, c.config.GPS.BaudRate)
	if err != nil {
		return fmt.Errorf("failed to initialize GPS: %w", err)
	}

	if err := c.gps.Start(); err != nil {
		return fmt.Errorf("failed to start GPS: %w", err)
	}

	if err := os.MkdirAll(c.config.Collection.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	c.writer = filewriter.NewWriter()

	return nil
}

func (c *Collector) WaitForGPSFix() error {
	fmt.Printf("Waiting for GPS fix (timeout: %v)...\n", c.config.GPS.Timeout)
	
	position, err := c.gps.WaitForFix(c.config.GPS.Timeout)
	if err != nil {
		return fmt.Errorf("GPS fix failed: %w", err)
	}

	fmt.Printf("GPS fix acquired: %.6f, %.6f (quality: %s, satellites: %d)\n",
		position.Latitude, position.Longitude,
		c.gps.GetFixQualityString(), position.Satellites)

	return nil
}

func (c *Collector) Collect() error {
	var startTime time.Time
	
	if c.config.Collection.SyncedStart {
		startTime = c.calculateSyncedStartTime()
		fmt.Printf("Synchronized start enabled - waiting until: %s\n", startTime.Format("15:04:05.000"))
		
		waitDuration := time.Until(startTime)
		if waitDuration > 0 {
			fmt.Printf("Waiting %.3f seconds for synchronized start...\n", waitDuration.Seconds())
			time.Sleep(waitDuration)
		}
	} else {
		startTime = time.Now()
	}
	
	collectionID := fmt.Sprintf("%s_%d", c.config.Collection.FilePrefix, startTime.Unix())
	
	fmt.Printf("Starting collection (ID: %s)\n", collectionID)
	fmt.Printf("Duration: %v\n", c.config.Collection.Duration)
	
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
	}()

	select {
	case samples := <-samplesChan:
		gpsPos, err := c.gps.GetCurrentPosition()
		if err != nil {
			return fmt.Errorf("failed to get GPS position: %w", err)
		}

		collectionData := CollectionData{
			IQSamples:    samples,
			GPSPosition:  *gpsPos,
			CollectionID: collectionID,
		}

		filename := filepath.Join(c.config.Collection.OutputDir, collectionID+".dat")
		if err := c.saveData(filename, collectionData); err != nil {
			return fmt.Errorf("failed to save data: %w", err)
		}

		fmt.Printf("Collection saved to: %s\n", filename)
		fmt.Printf("Samples collected: %d\n", len(samples.Data))

	case <-time.After(c.config.Collection.Duration + 5*time.Second):
		return fmt.Errorf("collection timeout")
	}

	c.wg.Wait()
	return nil
}

func (c *Collector) saveData(filename string, data CollectionData) error {
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
		DeviceInfo:        fmt.Sprintf("RTL-SDR Device %d", c.config.RTLSDR.DeviceIndex),
		FileFormatVersion: 1,
		CollectionID:      data.CollectionID,
	}

	return c.writer.WriteFile(filename, metadata, data.IQSamples.Data)
}

func (c *Collector) calculateSyncedStartTime() time.Time {
	now := time.Now()
	currentEpoch := now.Unix()
	
	// Add 5 seconds to current epoch time, then mod 100 for sync point
	futureEpoch := currentEpoch + 5
	syncPoint := futureEpoch % 100
	
	// Find next time when seconds field equals syncPoint
	nextMinute := (currentEpoch/60 + 1) * 60  // Start of next minute
	targetTime := nextMinute + int64(syncPoint)
	
	// If target is in the past, add another minute
	if targetTime <= currentEpoch {
		targetTime += 60
	}
	
	return time.Unix(targetTime, 0)
}

func (c *Collector) Stop() {
	close(c.stopChan)
	c.wg.Wait()
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
