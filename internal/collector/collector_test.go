package collector

import (
	"context"
	"os"
	"testing"
	"time"

	"argus-collector/internal/config"
)

func TestCollectionNormalOperation(t *testing.T) {
	// This test verifies that collection works correctly with normal durations
	// after fixing the filewriter performance issue

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "collector_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration with 2 second duration
	cfg := &config.Config{
		Collection: config.CollectionConfig{
			Duration:   2 * time.Second,
			FilePrefix: "test",
			OutputDir:  tempDir,
		},
		RTLSDR: config.RTLSDRConfig{
			Frequency:  433000000,
			SampleRate: 2048000,
			Gain:       0,
			GainMode:   "manual",
		},
		GPS: config.GPSConfig{
			Mode:            "manual",
			ManualLatitude:  35.533,
			ManualLongitude: -97.621,
			ManualAltitude:  365.0,
		},
	}

	// Create collector using the NewCollector function
	collector := NewCollector(cfg)

	// Initialize the collector
	err = collector.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize collector: %v", err)
	}
	defer collector.Close()

	// Test that collection now succeeds
	ctx := context.Background()

	// Record start time
	startTime := time.Now()

	// Run collection - this should now succeed
	err = collector.CollectWithContext(ctx)

	// Record end time
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)

	// Verify the collection succeeded
	if err != nil {
		t.Fatalf("Expected collection to succeed but got error: %v", err)
	}

	// Verify timing - should complete in reasonable time (duration + some overhead)
	maxExpectedTime := cfg.Collection.Duration + 5*time.Second // Allow overhead for file I/O
	if elapsedTime > maxExpectedTime {
		t.Fatalf("Collection took too long. Expected max %v, got %v", maxExpectedTime, elapsedTime)
	}

	// Should take at least the collection duration
	if elapsedTime < cfg.Collection.Duration {
		t.Fatalf("Collection completed too quickly. Expected at least %v, got %v", cfg.Collection.Duration, elapsedTime)
	}

	// Verify that a data file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No data file was created")
	}

	// Check that the file has the expected prefix
	foundTestFile := false
	for _, file := range files {
		if len(file.Name()) >= 4 && file.Name()[:4] == "test" {
			foundTestFile = true
			break
		}
	}

	if !foundTestFile {
		t.Fatal("Data file with expected prefix 'test' not found")
	}

	t.Logf("Collection succeeded in %v with %d file(s) created", elapsedTime, len(files))
}

func TestCollectionSuccess(t *testing.T) {
	// This test verifies that collection now works correctly after fixing the timeout issue

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "collector_test_success")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration with short duration for quick test
	cfg := &config.Config{
		Collection: config.CollectionConfig{
			Duration:   100 * time.Millisecond, // Short duration for quick test
			FilePrefix: "test",
			OutputDir:  tempDir,
		},
		RTLSDR: config.RTLSDRConfig{
			Frequency:  433000000,
			SampleRate: 2048000,
			Gain:       0,
			GainMode:   "manual",
		},
		GPS: config.GPSConfig{
			Mode:            "manual",
			ManualLatitude:  35.533,
			ManualLongitude: -97.621,
			ManualAltitude:  365.0,
		},
	}

	// Create collector using the NewCollector function
	collector := NewCollector(cfg)

	// Initialize the collector
	err = collector.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize collector: %v", err)
	}
	defer collector.Close()

	// Test that collection now succeeds
	ctx := context.Background()

	// Record start time
	startTime := time.Now()

	// Run collection - this should now succeed
	err = collector.CollectWithContext(ctx)

	// Record end time
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)

	// Verify the collection succeeded
	if err != nil {
		t.Fatalf("Expected collection to succeed but got error: %v", err)
	}

	// Verify timing - should complete quickly (not wait for duration + 20 seconds)
	maxExpectedTime := cfg.Collection.Duration + 2*time.Second // Allow some overhead
	if elapsedTime > maxExpectedTime {
		t.Fatalf("Collection took too long. Expected max %v, got %v", maxExpectedTime, elapsedTime)
	}

	// Verify that a data file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No data file was created")
	}

	// Check that the file has the expected prefix
	foundTestFile := false
	for _, file := range files {
		if file.Name()[:4] == "test" {
			foundTestFile = true
			break
		}
	}

	if !foundTestFile {
		t.Fatal("Data file with expected prefix 'test' not found")
	}

	t.Logf("Collection succeeded in %v with %d file(s) created", elapsedTime, len(files))
}
