package gps

import (
	"bufio"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/adrianmo/go-nmea"
	"github.com/stratoberry/go-gpsd"
	"go.bug.st/serial"
)

type Position struct {
	Latitude   float64
	Longitude  float64
	Altitude   float64
	Timestamp  time.Time
	FixQuality int
	Satellites int
}

// GPSInterface defines the common interface for GPS implementations
type GPSInterface interface {
	Start() error
	WaitForFix(timeout time.Duration) (*Position, error)
	GetCurrentPosition() (*Position, error)
	IsFixValid() bool
	GetFixQualityString() string
	Close() error
}

// GPS wraps either NMEA serial or gpsd implementation
type GPS struct {
	impl GPSInterface
}

// NMEASerial implements GPS via serial NMEA interface
type NMEASerial struct {
	port     serial.Port
	position Position
	fixChan  chan Position
	mu       sync.RWMutex
	debug    bool
}

// GPSDClient implements GPS via gpsd daemon
type GPSDClient struct {
	client   *gpsd.Session
	position Position
	fixChan  chan Position
	host     string
	port     string
}

// NewGPS creates a GPS instance with NMEA serial interface
func NewGPS(portName string, baudRate int) (*GPS, error) {
	nmeaSerial, err := NewNMEASerial(portName, baudRate)
	if err != nil {
		return nil, err
	}
	return &GPS{impl: nmeaSerial}, nil
}

// NewGPSD creates a GPS instance with gpsd interface
func NewGPSD(host, port string) (*GPS, error) {
	gpsdClient, err := NewGPSDClient(host, port)
	if err != nil {
		return nil, err
	}
	return &GPS{impl: gpsdClient}, nil
}

// NewNMEASerial creates a new NMEA serial GPS interface
func NewNMEASerial(portName string, baudRate int) (*NMEASerial, error) {
	return NewNMEASerialWithDebug(portName, baudRate, false)
}

// NewNMEASerialWithDebug creates a new NMEA serial GPS interface with debug option
func NewNMEASerialWithDebug(portName string, baudRate int, debug bool) (*NMEASerial, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to open GPS port %s: %w", portName, err)
	}

	nmea := &NMEASerial{
		port:    port,
		fixChan: make(chan Position, 10),
		debug:   debug,
	}

	// Try to configure u-blox GPS to output NMEA GGA messages if it's not already
	nmea.configureUbloxNMEA()

	return nmea, nil
}

// configureUbloxNMEA attempts to configure u-blox GPS to output NMEA GGA messages
func (n *NMEASerial) configureUbloxNMEA() {
	log.Printf("GPS: Attempting to configure u-blox GPS for NMEA output")

	// Send u-blox UBX command to enable NMEA GGA messages on UART1
	// UBX-CFG-MSG: Enable GGA messages (Class=0xF0, ID=0x00) on UART1 (port 1)
	// Message format: 0xB5 0x62 0x06 0x01 0x08 0x00 0xF0 0x00 0x00 0x01 0x00 0x00 0x00 0x00 + checksum
	ggaCmd := []byte{0xB5, 0x62, 0x06, 0x01, 0x08, 0x00, 0xF0, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x31}
	
	// Send u-blox UBX command to enable NMEA RMC messages on UART1  
	// UBX-CFG-MSG: Enable RMC messages (Class=0xF0, ID=0x04) on UART1 (port 1)
	rmcCmd := []byte{0xB5, 0x62, 0x06, 0x01, 0x08, 0x00, 0xF0, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x05, 0x3B}

	// Send the commands
	n.port.Write(ggaCmd)
	time.Sleep(100 * time.Millisecond)
	n.port.Write(rmcCmd)
	time.Sleep(100 * time.Millisecond)

	log.Printf("GPS: Sent u-blox configuration commands to enable NMEA GGA/RMC output")
}

// NewGPSDClient creates a new gpsd client interface
func NewGPSDClient(host, port string) (*GPSDClient, error) {
	return &GPSDClient{
		fixChan: make(chan Position, 10),
		host:    host,
		port:    port,
	}, nil
}

// GPS wrapper methods delegate to implementation
func (g *GPS) Start() error {
	return g.impl.Start()
}

func (g *GPS) WaitForFix(timeout time.Duration) (*Position, error) {
	return g.impl.WaitForFix(timeout)
}

func (g *GPS) GetCurrentPosition() (*Position, error) {
	return g.impl.GetCurrentPosition()
}

func (g *GPS) IsFixValid() bool {
	return g.impl.IsFixValid()
}

func (g *GPS) GetFixQualityString() string {
	return g.impl.GetFixQualityString()
}

func (g *GPS) Close() error {
	return g.impl.Close()
}

// SetDebug enables or disables debug logging for GPS implementations that support it
func (g *GPS) SetDebug(debug bool) {
	if nmea, ok := g.impl.(*NMEASerial); ok {
		nmea.SetDebug(debug)
	}
}

// NMEASerial implementation methods
func (n *NMEASerial) Start() error {
	go n.readLoop()
	return nil
}

func (n *NMEASerial) readLoop() {
	scanner := bufio.NewScanner(n.port)
	log.Printf("GPS: Starting NMEA read loop")

	for scanner.Scan() {
		line := scanner.Text()

		// Only process lines that look like NMEA sentences (start with $ and contain only printable ASCII)
		if len(line) == 0 || line[0] != '$' {
			continue
		}
		
		// Validate that line contains only printable ASCII to filter out binary data
		isPrintable := true
		for _, r := range line {
			if r < 32 || r > 126 {
				isPrintable = false
				break
			}
		}
		if !isPrintable {
			continue
		}

		if n.debug {
			log.Printf("GPS: Received NMEA: %s", line)
		}

		sentence, err := nmea.Parse(line)
		if err != nil {
			if n.debug {
				log.Printf("GPS: NMEA parse error: %v (line: %s)", err, line)
			}
			continue
		}

		switch s := sentence.(type) {
		case nmea.GGA:
			if n.debug {
				log.Printf("GPS: Processing GGA message")
			}
			n.processGGA(s)
		case nmea.RMC:
			if n.debug {
				log.Printf("GPS: Processing RMC message")
			}
			n.processRMC(s)
		case nmea.GLL, nmea.VTG, nmea.GSA, nmea.GSV:
			// These are valid NMEA sentences but don't contain position fixes we need
			if n.debug {
				log.Printf("GPS: Received %T message (not needed for position)", s)
			}
		default:
			if n.debug {
				log.Printf("GPS: Received %T message (ignoring)", s)
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		log.Printf("GPS: Scanner error: %v", err)
	}
	log.Printf("GPS: NMEA read loop ended")
}

func (n *NMEASerial) processGGA(s nmea.GGA) {
	if n.debug {
		log.Printf("GPS: Processing GGA - Quality: %v, Lat: %f, Lon: %f, Sats: %d",
			s.FixQuality, s.Latitude, s.Longitude, s.NumSatellites)
	}

	if s.FixQuality != nmea.Invalid {
		var fixQuality int
		switch s.FixQuality {
		case nmea.GPS:
			fixQuality = 1
		case nmea.DGPS:
			fixQuality = 2
		case nmea.PPS:
			fixQuality = 3
		case nmea.RTK:
			fixQuality = 4
		case nmea.FRTK:
			fixQuality = 5
		case nmea.Manual:
			fixQuality = 7
		default:
			fixQuality = 0
		}

		// Only process coordinates when we have a valid fix quality
		// Some GPS receivers output (0,0) when they don't have a fix yet, but
		// if fix quality is valid, we should trust the coordinates
		if fixQuality > 0 {
			pos := Position{
				Latitude:   s.Latitude,
				Longitude:  s.Longitude,
				Altitude:   s.Altitude,
				Timestamp:  time.Now(),
				FixQuality: fixQuality,
				Satellites: int(s.NumSatellites),
			}

			n.mu.Lock()
			n.position = pos
			n.mu.Unlock()

			if n.debug {
				log.Printf("GPS: Updated position - Lat: %.6f, Lon: %.6f, Alt: %.1f, Quality: %d, Sats: %d",
					pos.Latitude, pos.Longitude, pos.Altitude, pos.FixQuality, pos.Satellites)
			}

			select {
			case n.fixChan <- pos:
			default:
			}
		}
	}
}

func (n *NMEASerial) processRMC(s nmea.RMC) {
	// RMC provides additional validation and time info
	if n.debug {
		log.Printf("GPS: Processing RMC - Valid: %t, Lat: %f, Lon: %f",
			s.Validity == "A", s.Latitude, s.Longitude)
	}

	// Use RMC to supplement/validate position if we have one
	if s.Validity == "A" {
		n.mu.RLock()
		currentPos := n.position
		n.mu.RUnlock()

		// If we have a current position, update with RMC timestamp if more recent
		if currentPos.FixQuality > 0 {
			// Convert NMEA time to Go time.Time
			// RMC provides time but not full timestamp, so we use current date
			rncTime := time.Now()
			if s.Time.Valid {
				// Use today's date with the GPS time
				rncTime = time.Date(
					rncTime.Year(), rncTime.Month(), rncTime.Day(),
					s.Time.Hour, s.Time.Minute, s.Time.Second,
					int(s.Time.Millisecond)*1000000, // Convert ms to ns
					time.UTC,
				)
			}

			pos := Position{
				Latitude:   s.Latitude,
				Longitude:  s.Longitude,
				Altitude:   currentPos.Altitude, // RMC doesn't have altitude
				Timestamp:  rncTime,
				FixQuality: currentPos.FixQuality,
				Satellites: currentPos.Satellites,
			}

			n.mu.Lock()
			n.position = pos
			n.mu.Unlock()
		}
	}
}

func (n *NMEASerial) WaitForFix(timeout time.Duration) (*Position, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case pos := <-n.fixChan:
			if pos.FixQuality > 0 {
				return &pos, nil
			}
		case <-timer.C:
			return nil, fmt.Errorf("GPS fix timeout after %v. GPS may be configured to output UBX binary protocol instead of NMEA position messages. Consider using --gps-mode=gpsd or configure GPS to output NMEA GGA/RMC messages", timeout)
		}
	}
}

func (n *NMEASerial) GetCurrentPosition() (*Position, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.position.FixQuality == 0 {
		return nil, fmt.Errorf("no GPS fix available")
	}

	pos := n.position
	return &pos, nil
}

func (n *NMEASerial) IsFixValid() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.position.FixQuality > 0
}

func (n *NMEASerial) GetFixQualityString() string {
	n.mu.RLock()
	quality := n.position.FixQuality
	n.mu.RUnlock()

	switch quality {
	case 0:
		return "Invalid"
	case 1:
		return "GPS fix (SPS)"
	case 2:
		return "DGPS fix"
	case 3:
		return "PPS fix"
	case 4:
		return "Real Time Kinematic"
	case 5:
		return "Float RTK"
	case 6:
		return "estimated (dead reckoning)"
	case 7:
		return "Manual input mode"
	case 8:
		return "Simulation mode"
	default:
		return "Unknown"
	}
}

func (n *NMEASerial) Close() error {
	if n.port != nil {
		return n.port.Close()
	}
	return nil
}

// SetDebug enables or disables debug logging for NMEA GPS
func (n *NMEASerial) SetDebug(debug bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.debug = debug
	if debug {
		log.Printf("GPS: Debug mode enabled for NMEA GPS")
	}
}

// GPSDClient implementation methods
func (g *GPSDClient) Start() error {
	client, err := gpsd.Dial(gpsd.DefaultAddress)
	if err != nil {
		// Try custom host:port if default fails
		if g.host != "" && g.port != "" {
			address := fmt.Sprintf("%s:%s", g.host, g.port)
			client, err = gpsd.Dial(address)
			if err != nil {
				return fmt.Errorf("failed to connect to gpsd at %s: %w", address, err)
			}
		} else {
			return fmt.Errorf("failed to connect to gpsd: %w", err)
		}
	}

	g.client = client

	// Start watching for GPS data
	g.client.AddFilter("TPV", func(r interface{}) {
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
				Satellites: 0, // TPV doesn't include satellite count
			}

			g.position = pos

			select {
			case g.fixChan <- pos:
			default:
			}
		}
	})

	// Also watch for satellite info
	g.client.AddFilter("SKY", func(r interface{}) {
		sky, ok := r.(*gpsd.SKYReport)
		if !ok {
			return
		}

		// Update satellite count in current position
		if g.position.FixQuality > 0 {
			g.position.Satellites = len(sky.Satellites)
		}
	})

	// Start watching
	g.client.Watch()

	return nil
}

func (g *GPSDClient) WaitForFix(timeout time.Duration) (*Position, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case pos := <-g.fixChan:
			if pos.FixQuality > 0 {
				return &pos, nil
			}
		case <-timer.C:
			return nil, fmt.Errorf("GPS fix timeout after %v", timeout)
		}
	}
}

func (g *GPSDClient) GetCurrentPosition() (*Position, error) {
	if g.position.FixQuality == 0 {
		return nil, fmt.Errorf("no GPS fix available")
	}

	pos := g.position
	return &pos, nil
}

func (g *GPSDClient) IsFixValid() bool {
	return g.position.FixQuality > 0
}

func (g *GPSDClient) GetFixQualityString() string {
	switch g.position.FixQuality {
	case 0:
		return "Invalid"
	case 1:
		return "GPS fix (via gpsd)"
	case 2:
		return "DGPS fix (via gpsd)"
	case 3:
		return "PPS fix (via gpsd)"
	case 4:
		return "Real Time Kinematic (via gpsd)"
	case 5:
		return "Float RTK (via gpsd)"
	case 6:
		return "estimated (dead reckoning) (via gpsd)"
	case 7:
		return "Manual input mode (via gpsd)"
	case 8:
		return "Simulation mode (via gpsd)"
	default:
		return "Unknown (via gpsd)"
	}
}

func (g *GPSDClient) Close() error {
	if g.client != nil {
		g.client.Close()
	}
	return nil
}
