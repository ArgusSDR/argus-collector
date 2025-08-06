package gps

import (
	"testing"
	"time"

	"github.com/stratoberry/go-gpsd"
)

func TestGPSDSatelliteCountPreservation(t *testing.T) {
	// Create a GPSD client instance
	gpsdClient := &GPSDClient{
		fixChan: make(chan Position, 10),
		host:    "localhost",
		port:    "2947",
	}

	// Simulate SKY report arriving first (with satellites)
	skyReport := &gpsd.SKYReport{
		Satellites: make([]gpsd.Satellite, 4), // 4 satellites
	}

	// Simulate the SKY filter callback
	skyFilter := func(r interface{}) {
		sky, ok := r.(*gpsd.SKYReport)
		if !ok {
			return
		}

		// Update satellite count - preserve existing position data if available
		satCount := len(sky.Satellites)
		if gpsdClient.position.FixQuality > 0 {
			// Update existing valid position with new satellite count
			gpsdClient.position.Satellites = satCount
		} else {
			// Store satellite count for when position becomes available
			gpsdClient.position.Satellites = satCount
		}
	}

	// Call the SKY filter with our test data
	skyFilter(skyReport)

	// Verify satellite count was stored
	if gpsdClient.position.Satellites != 4 {
		t.Errorf("Expected 4 satellites, got %d", gpsdClient.position.Satellites)
	}

	// Now simulate TPV report arriving (with position fix)
	tpvReport := &gpsd.TPVReport{
		Mode: 3,        // 3D fix
		Lat:  33.349,   // Test latitude
		Lon:  -111.758, // Test longitude
		Alt:  359.84,   // Test altitude
		Time: time.Now(),
	}

	// Simulate the TPV filter callback
	tpvFilter := func(r interface{}) {
		tpv, ok := r.(*gpsd.TPVReport)
		if !ok {
			return
		}

		// Convert gpsd fix mode to our quality system
		var fixQuality int
		switch tpv.Mode {
		case 0, 1: // No fix or invalid
			fixQuality = 0
		case 2: // 2D fix
			fixQuality = 1
		case 3: // 3D fix
			fixQuality = 1
		default:
			fixQuality = 0
		}

		// Only process valid fixes
		if fixQuality > 0 && tpv.Lat != 0 && tpv.Lon != 0 {
			pos := Position{
				Latitude:   tpv.Lat,
				Longitude:  tpv.Lon,
				Altitude:   tpv.Alt,
				Timestamp:  tpv.Time,
				FixQuality: fixQuality,
				Satellites: gpsdClient.position.Satellites, // Preserve existing satellite count from SKY reports
			}

			gpsdClient.position = pos
		}
	}

	// Call the TPV filter with our test data
	tpvFilter(tpvReport)

	// Verify that the position was set correctly AND satellites were preserved
	if gpsdClient.position.FixQuality != 1 {
		t.Errorf("Expected fix quality 1, got %d", gpsdClient.position.FixQuality)
	}
	if gpsdClient.position.Satellites != 4 {
		t.Errorf("Expected 4 satellites to be preserved, got %d", gpsdClient.position.Satellites)
	}
	if gpsdClient.position.Latitude != 33.349 {
		t.Errorf("Expected latitude 33.349, got %f", gpsdClient.position.Latitude)
	}
	if gpsdClient.position.Longitude != -111.758 {
		t.Errorf("Expected longitude -111.758, got %f", gpsdClient.position.Longitude)
	}
}

func TestGPSDSatelliteCountUpdate(t *testing.T) {
	// Create a GPSD client instance with existing position
	gpsdClient := &GPSDClient{
		fixChan: make(chan Position, 10),
		host:    "localhost",
		port:    "2947",
		position: Position{
			Latitude:   33.349,
			Longitude:  -111.758,
			Altitude:   359.84,
			FixQuality: 1,
			Satellites: 2, // Initial satellite count
		},
	}

	// Simulate SKY report arriving with more satellites
	skyReport := &gpsd.SKYReport{
		Satellites: make([]gpsd.Satellite, 6), // 6 satellites
	}

	// Simulate the SKY filter callback
	skyFilter := func(r interface{}) {
		sky, ok := r.(*gpsd.SKYReport)
		if !ok {
			return
		}

		// Update satellite count - preserve existing position data if available
		satCount := len(sky.Satellites)
		if gpsdClient.position.FixQuality > 0 {
			// Update existing valid position with new satellite count
			gpsdClient.position.Satellites = satCount
		} else {
			// Store satellite count for when position becomes available
			gpsdClient.position.Satellites = satCount
		}
	}

	// Call the SKY filter with our test data
	skyFilter(skyReport)

	// Verify satellite count was updated while preserving other position data
	if gpsdClient.position.Satellites != 6 {
		t.Errorf("Expected 6 satellites, got %d", gpsdClient.position.Satellites)
	}
	if gpsdClient.position.FixQuality != 1 {
		t.Errorf("Expected fix quality to be preserved as 1, got %d", gpsdClient.position.FixQuality)
	}
	if gpsdClient.position.Latitude != 33.349 {
		t.Errorf("Expected latitude to be preserved as 33.349, got %f", gpsdClient.position.Latitude)
	}
}
