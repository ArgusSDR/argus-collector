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
- **Software-based AGC**: Intelligent automatic gain control with configurable target levels
- **Manual Gain Control**: Precise gain settings for consistent multi-station operation
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
--gain=20.7              # Manual gain in dB (0-50)
--gain-mode=auto         # Automatic gain control (auto|manual)
--frequency-correction=0 # PPM correction for crystal accuracy

# Hardware control  
--device-index=0         # RTL-SDR device index (if multiple devices)
--bias-tee              # Enable bias tee for LNA power
--direct-sampling       # Enable direct sampling mode

# Advanced options
--verbose               # Enable detailed logging (AGC, GPS debug)
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

## Automatic Gain Control (AGC)

The argus-collector includes sophisticated software-based AGC for optimal signal capture across varying conditions.

### AGC Operation

```bash
# Enable automatic gain control
./argus-collector --gain-mode=auto --frequency=162400000 --duration=30s

# Manual gain control (recommended for multi-station TDoA)
./argus-collector --gain-mode=manual --gain=20.7 --frequency=162400000 --duration=30s

# AGC with verbose monitoring
./argus-collector --gain-mode=auto --verbose --frequency=162400000 --duration=10s
```

### AGC Configuration

The AGC system can be fine-tuned via configuration file:

```yaml
# config.yaml
rtlsdr:
  gain_mode: "auto"        # Enable AGC
  # gain: 20.7             # Not used in auto mode
  
# AGC operates with these built-in parameters:
# - Target Power: 70% of full scale
# - Gain Range: 0.0 to 49.6 dB
# - Adjustment Step: 3.0 dB
# - Update Rate: Per collection period
```

### AGC Output Example

```bash
$ ./argus-collector --gain-mode=auto --duration=10s --frequency=162400000 --verbose

GPS fix acquired: 35.533210, -97.621322 (quality: GPS fix (via gpsd), satellites: 7)
Starting collection (ID: argus-0_1754539847, Duration: 10s)
Device: RTL-SDR Blog V3 (freq: 162400000 Hz, rate: 2048000 Hz, gain: 20.7 dB (auto), bias-tee: off)
AGC: Power=0.086 (target=0.700), Gain: 20.7→23.7 dB
AGC: Power=0.345 (target=0.700), Gain: 23.7→26.2 dB
AGC: Power=0.521 (target=0.700), Gain: 26.2→27.8 dB
AGC: Power=0.683 (target=0.700), Gain: 27.8→28.1 dB
AGC: Power=0.697 (target=0.700), Gain: 28.1→28.1 dB
Collection saved to: data/argus-0_1754539847.dat
Samples collected: 20480000
AGC converged to 28.1 dB gain
```

### AGC vs Manual Gain

| Mode | Best For | Advantages | Disadvantages |
|------|----------|------------|---------------|
| **AGC (auto)** | Single station, varying conditions | Adapts to signal levels, maximizes dynamic range | Gain varies between collections |
| **Manual** | Multi-station TDoA | Consistent gain across stations, repeatable results | Requires manual optimization |

### TDoA Deployment Recommendations

For **multi-station TDoA** deployments:

1. **Use Manual Gain**: Ensures consistent gain across all stations
2. **Test with AGC First**: Determine optimal gain for each location
3. **Apply Consistent Settings**: Use the AGC-determined gain in manual mode

```bash
# Step 1: Determine optimal gain at each station using AGC
./argus-collector --gain-mode=auto --duration=30s --frequency=162400000 --verbose

# Step 2: Note the final AGC gain (e.g., 28.1 dB)
# AGC converged to 28.1 dB gain

# Step 3: Use manual gain for actual TDoA collections
./argus-collector --gain-mode=manual --gain=28.1 --duration=30s --frequency=162400000
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

**Example Output:**
```bash
GPS fix acquired: 35.533210, -97.621322 (quality: GPS fix (via gpsd), satellites: 7)
```

**Advantages:**
- Shared GPS access across multiple applications
- Network-based GPS sharing
- Enhanced GPS status monitoring
- Accurate satellite count reporting (shows satellites used in position fix)

### Manual Mode (Testing)
For testing without GPS hardware:
```bash
./argus-collector --gps-mode=manual --latitude=35.533 --longitude=-97.621
```

**Note:** Manual mode provides no timing synchronization - only for single-station testing.

## Multi-Station Synchronization

### Synchronized Start Algorithm

The collector uses a sophisticated epoch-based timing algorithm for autonomous multi-station coordination without requiring network communication between stations.

### Algorithm Overview

```bash
# Enable synchronized start (default)
./argus-collector --synced-start --frequency=162400000 --duration=30s

# Disable for immediate start
./argus-collector --synced-start=false --frequency=162400000 --duration=30s
```

### Mathematical Algorithm

The synchronized start calculation uses fixed 100-second epochs with a predetermined sync point:

```
Algorithm Steps:
1. syncEpoch = ((currentTime + 30) ÷ 100 + 1) × 100
2. syncPoint = 30 seconds (fixed)
3. targetTime = syncEpoch + syncPoint
4. If (targetTime - currentTime) < 10: targetTime += 100
```

### Example Calculation

```bash
Current time: 13:00:45 (epoch: 1754589645)

Step 1: Calculate next epoch boundary
syncEpoch = ((1754589645 + 30) ÷ 100 + 1) × 100 = 1754589700

Step 2: Add fixed sync point  
targetTime = 1754589700 + 30 = 1754589730

Result: All stations start at 13:02:10 (30 seconds past epoch)
```

### Algorithm Properties

| Property | Value | Explanation |
|----------|-------|-------------|
| **Epoch Cycle** | 100 seconds | Fixed synchronization intervals |
| **Sync Point** | 30 seconds | Fixed offset from epoch boundary |
| **Minimum Wait** | 10 seconds | Guaranteed preparation time |
| **Maximum Wait** | ~110 seconds | If sync point just passed |
| **Race Condition** | **Eliminated** | All stations calculate identical target time |

### Timing Characteristics

**✅ Advantages:**
- **No Race Conditions**: Stations starting within 100s window synchronize
- **Deterministic**: Same calculation result regardless of start time
- **Autonomous**: No network coordination required
- **Predictable**: Fixed 100-second intervals with :10, :40 start times

**Example Sync Times:**
```
13:01:10, 13:02:40, 13:04:10, 13:05:40, 13:07:10...
```

### Multi-Station Coordination

**Deployment Process:**

1. **GPS Synchronization**: Ensure all stations have GPS time sync
2. **Launch Window**: Start all collectors within same 100-second window  
3. **Automatic Coordination**: Algorithm calculates identical target time
4. **Synchronized Start**: All stations begin simultaneously
5. **Timestamp Embedding**: Each sample period tagged with GPS time

**Example Multi-Station Launch:**
```bash
# All stations can start anytime between 13:00:00 - 13:01:39
# They will all synchronize to start at 13:02:10

# Station 1 (started at 13:00:15)
./argus-collector --collection-id=north --synced-start

# Station 2 (started at 13:00:22)  
./argus-collector --collection-id=south --synced-start

# Station 3 (started at 13:01:05)
./argus-collector --collection-id=east --synced-start

# All stations output: "Synchronized start enabled - waiting until: 13:02:10.000"
```

### Network Requirements

**No Network Coordination Required:**
- **Independent Calculation**: Each station computes target time autonomously
- **GPS-Only Dependency**: Only requires GPS time synchronization
- **Air-Gap Compatible**: Works without network connectivity between stations
- **Scalable**: Supports unlimited number of stations

### Timing Accuracy

- **GPS Precision**: ±10-40 nanoseconds typical
- **Algorithm Precision**: 1-second synchronization accuracy
- **Sample Alignment**: Sub-sample timing accuracy with GPS timestamps
- **Clock Drift Immunity**: GPS provides continuous time reference

### Exact Start Time Option

For ultimate precision, you can specify an exact epoch timestamp:

```bash
# Calculate future time (current + 30 seconds)
future_time=$(date -d "+30 seconds" +%s)

# All stations use identical timestamp  
./argus-collector --start-time $future_time --collection-id=station1
./argus-collector --start-time $future_time --collection-id=station2  
./argus-collector --start-time $future_time --collection-id=station3
```

**Example with Specific Time:**
```bash
# Start all stations at exactly 2025-08-07 13:30:00 UTC
./argus-collector --start-time 1754591400 --collection-id=north
./argus-collector --start-time 1754591400 --collection-id=south  
./argus-collector --start-time 1754591400 --collection-id=east

# Output: "Exact start time specified - waiting until: 13:30:00.000"
```

**Advantages of --start-time:**
- **Perfect Synchronization**: All stations start at identical epoch timestamp
- **No Race Conditions**: Eliminates any timing calculation differences
- **External Coordination**: Allows coordination via external scheduling systems
- **Precision Control**: Nanosecond-level start time accuracy

**Time Validation:**
- **Future Time Required**: Start time must be in the future
- **Maximum Past Time**: Rejects times more than 10 seconds in the past
- **Format**: Unix epoch timestamp (seconds since 1970-01-01 00:00:00 UTC)

### Configuration Options

**Command Line:**
```bash
--synced-start=true     # Enable synchronized start (default)
--synced-start=false    # Start immediately
--start-time=1754591260 # Exact epoch timestamp for collection start (overrides synced-start)
```

**Configuration File:**
```yaml
collection:
  synced_start: true    # Enable epoch-based synchronization
  start_time: 1754591260 # Exact epoch timestamp (overrides synced_start)
```

### Timing Options Hierarchy

The collector uses the following priority order for start timing:

1. **--start-time** (highest priority) - Exact epoch timestamp
2. **--synced-start=true** - Automatic epoch-based synchronization  
3. **--synced-start=false** - Immediate start

**Example Priority Demonstration:**
```bash
# Uses exact start time (ignores synced-start)
./argus-collector --start-time 1754591400 --synced-start=true

# Uses synchronized start
./argus-collector --synced-start=true

# Starts immediately  
./argus-collector --synced-start=false
```

### Troubleshooting Synchronization

**Verify Synchronization:**
```bash
# Synchronized start - all stations should show identical target time
./argus-collector --synced-start --frequency=162400000 --duration=10s
# Output: "Synchronized start enabled - waiting until: 13:04:10.000"

# Exact start time - all stations should show identical target time
./argus-collector --start-time 1754591400 --frequency=162400000 --duration=10s
# Output: "Exact start time specified - waiting until: 13:30:00.000"
```

**Common Issues:**
- **Different Target Times**: Check GPS synchronization on all stations
- **Long Wait Times**: Normal behavior with --synced-start, maximum ~110 seconds
- **"Start time too far in past"**: Use future timestamp with --start-time
- **Immediate Start**: Verify correct timing option is set

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