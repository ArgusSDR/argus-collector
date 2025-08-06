# argus-collector

**Synchronized RTL-SDR data collection for Time Difference of Arrival (TDoA) signal processing**

`argus-collector` is the core data acquisition component of the Argus TDoA system, designed for precise, GPS-synchronized RF signal collection across multiple stations.

## Overview

The collector provides:
- **GPS-synchronized data collection** across multiple stations
- **Nanosecond-precision timing** for TDoA requirements
- **Flexible RTL-SDR configuration** with comprehensive parameter control
- **Robust data storage** in TDoA-optimized binary format
- **Multi-station coordination** with automatic epoch synchronization

## Key Features

### Synchronization Capabilities
- **GPS Integration**: NMEA serial and gpsd daemon support
- **Epoch-based Timing**: Automatic coordination across distributed stations
- **Nanosecond Precision**: GPS timestamps for each collection period
- **Time Zone Handling**: UTC coordination for global deployments

### RTL-SDR Control
- **Full Hardware Configuration**: Frequency, sample rate, gain control
- **Bias Tee Support**: Power external LNAs via antenna port
- **Automatic Gain Control**: AGC or manual gain settings
- **Device Detection**: Automatic RTL-SDR enumeration and selection

### Data Management
- **Custom Binary Format**: Optimized for TDoA processing requirements
- **Metadata Embedding**: GPS coordinates, timestamps, device configuration
- **Large File Handling**: Efficient storage of multi-GB datasets
- **Cross-platform Support**: Linux, Windows, macOS compatibility

## Quick Start

### Basic Collection
```bash
# 5-second collection at 162.400 MHz with GPS synchronization
./argus-collector --frequency=162400000 --duration=5s --gps-mode=nmea --gps-port=/dev/ttyACM0
```

### Multi-Station Deployment
```bash
# Station 1
./argus-collector --frequency=162400000 --duration=30s --collection-id=station1 \
  --gps-mode=nmea --gps-port=/dev/ttyACM0 --gain=20.7

# Station 2 (different location)
./argus-collector --frequency=162400000 --duration=30s --collection-id=station2 \
  --gps-mode=gpsd --gpsd-host=localhost --gain=20.7

# Station 3 (manual coordinates for testing)
./argus-collector --frequency=162400000 --duration=30s --collection-id=station3 \
  --gps-mode=manual --latitude=35.533 --longitude=-97.621 --gain=20.7
```

## Command Line Options

### Required Parameters
- `--frequency=Hz` - Center frequency for collection
- `--duration=Xs` - Collection duration (e.g., 30s, 5m, 1h)

### GPS Configuration
```bash
# NMEA Serial GPS (most common)
--gps-mode=nmea --gps-port=/dev/ttyACM0 --gps-baud=9600

# GPSD Daemon
--gps-mode=gpsd --gpsd-host=localhost --gpsd-port=2947

# Manual Coordinates (testing only)
--gps-mode=manual --latitude=35.533 --longitude=-97.621 --altitude=365
```

### RTL-SDR Settings
```bash
# Basic RF parameters
--sample-rate=2048000    # Sample rate in Hz (default: 2.048 MSps)
--gain=20.7              # Gain in dB (0-50, or 'auto' for AGC)
--frequency-correction=0 # PPM correction for crystal accuracy

# Hardware control  
--device-index=0         # RTL-SDR device index (if multiple devices)
--bias-tee              # Enable bias tee for LNA power
--direct-sampling       # Enable direct sampling mode
```

### Collection Control
```bash
--collection-id=mystation    # Unique identifier for this station
--output-dir=./data         # Output directory for data files
--file-prefix=capture       # Custom filename prefix
--config=config.yaml        # Load settings from configuration file
```

## Configuration File

Create a YAML configuration file to simplify deployment:

```yaml
# config.yaml
collection:
  frequency: 162400000
  sample_rate: 2048000
  duration: "30s"
  collection_id: "station1"

rtlsdr:
  gain: 20.7
  bias_tee: false
  frequency_correction: 0
  
gps:
  mode: "nmea"
  port: "/dev/ttyACM0"
  baud_rate: 9600

logging:
  level: "info"
  file: "argus-collector.log"
```

Usage:
```bash
./argus-collector --config=config.yaml
```

## GPS Integration

### NMEA Serial Connection
Most common setup for dedicated GPS receivers:
```bash
./argus-collector --gps-mode=nmea --gps-port=/dev/ttyACM0
```

**Supported Devices:**
- u-blox GPS modules (NEO-6M, NEO-8M, NEO-9M)
- Generic NMEA-compatible GPS receivers
- USB GPS dongles with serial interface

### GPSD Integration
For systems running gpsd daemon:
```bash
./argus-collector --gps-mode=gpsd --gpsd-host=localhost
```

**Advantages:**
- Shared GPS access across multiple applications
- Network-based GPS sharing
- Enhanced GPS status monitoring

### Manual Mode (Testing)
For testing without GPS hardware:
```bash
./argus-collector --gps-mode=manual --latitude=35.533 --longitude=-97.621
```

**Note:** Manual mode provides no timing synchronization - only for single-station testing.

## Multi-Station Synchronization

### Automatic Epoch Coordination
The collector automatically coordinates with other stations:

1. **GPS Lock Acquisition**: Each station waits for GPS synchronization
2. **Network Coordination**: Stations communicate via broadcast/multicast
3. **Epoch Calculation**: Next GPS second boundary selected as start time
4. **Synchronized Start**: All stations begin collection simultaneously
5. **Timestamp Embedding**: Each sample period tagged with GPS time

### Network Requirements
- **UDP Multicast**: Default coordination method
- **Broadcast**: Alternative for simple networks  
- **Manual Sync**: Specify exact start time for air-gapped deployments

### Timing Accuracy
- **GPS Precision**: ±10-40 nanoseconds typical
- **Network Jitter**: <1 millisecond impact
- **Sample Alignment**: Sub-sample timing accuracy
- **Clock Drift**: Periodic GPS re-synchronization

## Data Output Format

### File Structure
```
Output files: argus-{collection-id}_{timestamp}.dat

Example: argus-station1_1698765432.dat
```

### Binary Format
```
Header (variable length):
- Magic: "ARGUS" (5 bytes)
- Format Version: uint16 (2 bytes)
- Frequency: uint64 (8 bytes)
- Sample Rate: uint32 (4 bytes)
- Collection Timestamp: GPS time (12 bytes)
- GPS Location: lat/lon/alt (24 bytes)
- GPS Timestamp: GPS time (12 bytes)
- Device Info: string (variable)
- Collection ID: string (variable)
- Sample Count: uint32 (4 bytes)

Data (fixed length per sample):
- IQ Samples: complex64 pairs (8 bytes each)
  - I component: float32 (4 bytes)
  - Q component: float32 (4 bytes)
```

### Metadata Preservation
Each file contains complete collection context:
- **Precise GPS timestamps** for correlation
- **Station coordinates** for TDoA geometry
- **Hardware configuration** for signal analysis
- **Collection parameters** for processing validation

## Hardware Requirements

### Minimum System
- **RTL-SDR Dongle**: RTL2832U + R820T2/R828D tuner
- **GPS Receiver**: NMEA or gpsd compatible
- **Computing Platform**: Raspberry Pi 3B+ or equivalent
- **Storage**: 1GB+ available space per minute of collection
- **Network**: Ethernet or WiFi for multi-station coordination

### Recommended System  
- **High-Performance RTL-SDR**: RTL-SDR Blog V3 or V4
- **Precision GPS**: u-blox NEO-8M/9M with external antenna
- **Computing Platform**: Raspberry Pi 4 or x86 system
- **Storage**: SSD recommended for high sample rate collections
- **Network**: Wired Ethernet for lowest latency

### Antenna Considerations
- **Frequency Appropriate**: Matched to target signals
- **Omnidirectional**: Broad coverage for unknown signal directions
- **Low Noise**: Minimize system noise figure
- **Stable Mounting**: Consistent pattern and gain

## Performance Optimization

### Sample Rate Selection
```bash
# Conservative (recommended for most deployments)
--sample-rate=1024000    # 1.024 MSps - good balance

# High Performance (requires fast storage)
--sample-rate=2048000    # 2.048 MSps - maximum precision

# Low Resource (constrained systems)
--sample-rate=512000     # 512 kSps - minimum viable
```

### Storage Requirements
- **1 MSps**: ~460 MB per minute
- **2 MSps**: ~920 MB per minute  
- **Factor in Duration**: 30-minute collection = ~14 GB at 2 MSps

### CPU and Memory
- **CPU Usage**: ~5-15% on Raspberry Pi 4
- **Memory**: ~50-100 MB base + sample buffers
- **I/O Performance**: Limited by storage write speed

## Troubleshooting

### GPS Issues
```bash
# Check GPS device
ls -la /dev/tty*        # Look for /dev/ttyACM0 or /dev/ttyUSB0

# Test GPS communication
sudo apt-get install gpsd-clients
cgps -s                 # Monitor GPS status

# GPS permissions
sudo usermod -a -G dialout $USER
```

### RTL-SDR Problems
```bash
# Verify RTL-SDR detection
rtl_test -t            # Hardware test
rtl_eeprom            # Device information

# Fix device permissions
sudo usermod -a -G plugdev $USER

# Kill conflicting processes
sudo killall rtl_fm rtl_power rtl_sdr
```

### Network Coordination
```bash
# Test multicast connectivity
ping 224.0.0.251      # Multicast ping test

# Firewall configuration
sudo ufw allow 23456/udp    # Default coordination port
```

### Performance Issues
```bash
# Monitor system resources
htop                   # CPU and memory usage
iotop                 # Disk I/O monitoring
df -h                 # Storage space

# Check for dropped samples
dmesg | grep -i rtl   # Kernel messages
```

## Integration

### With argus-reader
```bash
# Validate collected data
./argus-reader collected_data.dat

# Quality analysis
./argus-reader --stats collected_data.dat
```

### With argus-processor
```bash
# Multi-station TDoA processing
./argus-processor --station1=station1.dat --station2=station2.dat --station3=station3.dat
```

### Automated Deployment
```bash
#!/bin/bash
# Automated collection script
./argus-collector \
  --config=station.yaml \
  --duration=1h \
  --output-dir=/data/$(date +%Y%m%d) \
  >> collection.log 2>&1
```

## Advanced Features

### Long Duration Collections
```bash
# Multi-hour collections with automatic file rotation
./argus-collector --duration=8h --max-file-size=2GB
```

### Signal Detection Mode
```bash
# Trigger-based collection (future enhancement)
./argus-collector --trigger-threshold=-50dBm --pre-trigger=1s
```

### Remote Operation
```bash
# Headless deployment with remote monitoring
./argus-collector --daemon --status-port=8080
```

---

For complete system deployment information, see the main project [README.md](README.md) and [How-It-Works.md](How-It-Works.md).

For data analysis and validation, see [argus-reader-README.md](argus-reader-README.md).

For TDoA processing, see [argus-processor-README.md](argus-processor-README.md).