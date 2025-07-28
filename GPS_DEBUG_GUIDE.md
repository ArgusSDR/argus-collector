# GPS NMEA Troubleshooting Guide

## Issues Fixed

### 1. Silent NMEA Parsing Failures
**Problem**: NMEA parsing errors were silently ignored, making it hard to debug GPS issues.

**Solution**: Added comprehensive logging and error handling:
- Parse errors are now logged in debug mode
- Scanner errors are detected and reported
- Empty lines are properly skipped

### 2. Missing Debug Output
**Problem**: No way to see what NMEA data was being received.

**Solution**: Added debug logging that shows:
- Raw NMEA sentences received
- Parsed GGA/RMC data
- Position updates with coordinates and quality

### 3. Race Conditions
**Problem**: GPS position could be read while being written, causing data races.

**Solution**: Added proper mutex synchronization:
- Read/write locks protect position updates
- Thread-safe access to GPS state

### 4. Overly Restrictive Coordinate Filtering
**Problem**: GPS coordinates of (0,0) were being filtered out, but this is a valid location.

**Solution**: Changed logic to trust GPS fix quality indicators instead of coordinate values.

## Usage

### Enable Debug Mode
```bash
# Via command line flag
./argus-collector --verbose --gps-mode=nmea --gps-port=/dev/ttyACM0

# Via config file
gps:
  mode: "nmea"
  port: "/dev/ttyACM0"
logging:
  level: "debug"
```

### Debug Output Examples
```
GPS: Starting NMEA read loop
GPS: Received NMEA: $GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47
GPS: Processing GGA - Quality: GPS, Lat: 48.117300, Lon: 11.516667, Sats: 8
GPS: Updated position - Lat: 48.117300, Lon: 11.516667, Alt: 545.4, Quality: 1, Sats: 8
```

### Common Issues and Solutions

#### No NMEA Data Received
- Check serial port permissions: `sudo usermod -a -G dialout $USER`
- Verify GPS device path: `ls -la /dev/tty*`
- Test with: `cat /dev/ttyACM0` to see raw NMEA output

#### GPS Fix Timeout
- GPS may need clear sky view for initial fix
- Cold start can take 30+ seconds
- Check satellite count in debug output

#### Permission Denied on Serial Port
```bash
sudo chmod 666 /dev/ttyACM0  # Temporary fix
# Or add user to dialout group (permanent)
sudo usermod -a -G dialout $USER
# Logout and login again
```

## Testing GPS Modes

### NMEA Mode (Direct Serial)
```bash
./argus-collector --verbose --gps-mode=nmea --gps-port=/dev/ttyACM0 --duration=10s
```

### GPSD Mode
```bash
# Start gpsd daemon first
sudo gpsd /dev/ttyACM0
./argus-collector --verbose --gps-mode=gpsd --gpsd-host=localhost --duration=10s
```

### Manual Mode (Testing)
```bash
./argus-collector --gps-mode=manual --latitude=35.533133 --longitude=-97.621302 --duration=10s
```