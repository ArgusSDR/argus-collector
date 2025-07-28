package gps

import (
	"bufio"
	"fmt"
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

	return &NMEASerial{
		port:    port,
		fixChan: make(chan Position, 10),
	}, nil
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

// NMEASerial implementation methods
func (n *NMEASerial) Start() error {
	go n.readLoop()
	return nil
}

func (n *NMEASerial) readLoop() {
	scanner := bufio.NewScanner(n.port)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		sentence, err := nmea.Parse(line)
		if err != nil {
			continue
		}

		switch s := sentence.(type) {
		case nmea.GGA:
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
				
				pos := Position{
					Latitude:   s.Latitude,
					Longitude:  s.Longitude,
					Altitude:   s.Altitude,
					Timestamp:  time.Now(),
					FixQuality: fixQuality,
					Satellites: int(s.NumSatellites),
				}
				
				n.position = pos
				
				select {
				case n.fixChan <- pos:
				default:
				}
			}
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
			return nil, fmt.Errorf("GPS fix timeout after %v", timeout)
		}
	}
}

func (n *NMEASerial) GetCurrentPosition() (*Position, error) {
	if n.position.FixQuality == 0 {
		return nil, fmt.Errorf("no GPS fix available")
	}
	
	pos := n.position
	return &pos, nil
}

func (n *NMEASerial) IsFixValid() bool {
	return n.position.FixQuality > 0
}

func (n *NMEASerial) GetFixQualityString() string {
	switch n.position.FixQuality {
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