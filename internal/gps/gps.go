package gps

import (
	"bufio"
	"fmt"
	"time"

	"github.com/adrianmo/go-nmea"
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

type GPS struct {
	port     serial.Port
	position Position
	fixChan  chan Position
}

func NewGPS(portName string, baudRate int) (*GPS, error) {
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

	return &GPS{
		port:    port,
		fixChan: make(chan Position, 10),
	}, nil
}

func (g *GPS) Start() error {
	go g.readLoop()
	return nil
}

func (g *GPS) readLoop() {
	scanner := bufio.NewScanner(g.port)
	
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
				
				g.position = pos
				
				select {
				case g.fixChan <- pos:
				default:
				}
			}
		}
	}
}

func (g *GPS) WaitForFix(timeout time.Duration) (*Position, error) {
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

func (g *GPS) GetCurrentPosition() (*Position, error) {
	if g.position.FixQuality == 0 {
		return nil, fmt.Errorf("no GPS fix available")
	}
	
	pos := g.position
	return &pos, nil
}

func (g *GPS) IsFixValid() bool {
	return g.position.FixQuality > 0
}

func (g *GPS) GetFixQualityString() string {
	switch g.position.FixQuality {
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

func (g *GPS) Close() error {
	if g.port != nil {
		return g.port.Close()
	}
	return nil
}