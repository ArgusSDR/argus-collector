# Argus Collector

A high-precision RTL-SDR signal collection tool designed for Time Difference of Arrival (TDOA) analysis and transmitter localization. Argus Collector simultaneously captures radio frequency signals and GPS positioning data with nanosecond timing precision to support multi-station TDOA workflows.

## Features

- **RTL-SDR Integration**: Direct interface with RTL-SDR hardware for IQ signal collection
- **GPS Synchronization**: NMEA-compliant GPS receiver support for precise timing and positioning
- **Synchronized Collection**: Epoch-based timing for coordinated multi-station data collection
- **TDOA-Ready Output**: Custom binary format optimized for multi-station analysis
- **Nanosecond Precision**: High-resolution timestamps for accurate time difference calculations
- **Configurable Collection**: Flexible frequency, duration, and output settings
- **Graceful Operation**: Signal handling and error recovery mechanisms

## Hardware Requirements

- **RTL-SDR Device**: Any RTL2832U-based Software Defined Radio
- **GPS Receiver**: NMEA-compatible GPS module (USB or serial interface)
- **Linux System**: Tested on Ubuntu/Debian with librtlsdr support

## Quick Start

### 1. Install Prerequisites

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install librtlsdr-dev build-essential git

# Fedora/RHEL
sudo dnf install rtl-sdr-devel gcc git

# Arch Linux
sudo pacman -S rtl-sdr git base-devel
```

### 2. Install Go (if not already installed)

```bash
# Download and install Go 1.19 or later
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

### 3. Clone and Build

```bash
git clone <repository-url>
cd Argus_Collector
make build
```

### 4. Connect Hardware

```bash
# Connect RTL-SDR device
# Connect GPS receiver (USB/serial)

# Test RTL-SDR detection
rtl_test

# Test GPS communication
cat /dev/ttyACM0  # or your GPS device
```

### 5. Run Collection

```bash
# Basic collection (60 seconds at 433.92 MHz)
./argus-collector --gps-port /dev/ttyACM0

# Custom frequency and duration
./argus-collector --frequency 915e6 --duration 30s --gps-port /dev/ttyACM0
```

## Build Instructions

### Using Make (Recommended)

```bash
# Standard build with RTL-SDR support
make build

# Development build without RTL-SDR (for testing)
make build-stub

# Build for all platforms
make build-all

# Create release packages
make package

# Clean build artifacts
make clean

# View all available targets
make help
```

### Manual Build

```bash
# With RTL-SDR support (requires librtlsdr-dev)
go build -tags rtlsdr -ldflags "-s -w" -o argus-collector .

# Without RTL-SDR support (stub functions)
go build -ldflags "-s -w" -o argus-collector .

# Cross-compilation for different platforms
GOOS=linux GOARCH=amd64 go build -tags rtlsdr -o argus-collector-linux .
GOOS=windows GOARCH=amd64 go build -o argus-collector-windows.exe .
```

### Build Troubleshooting

```bash
# Missing RTL-SDR headers
sudo apt-get install librtlsdr-dev

# Missing Go compiler
go version  # Should be 1.19 or later

# Permission issues
sudo usermod -a -G plugdev $USER  # Add user to plugdev group
# Logout and login again

# Clean and retry
make clean
go clean -cache
make build
```

## Usage

### Command Line Reference

```
Usage: argus-collector [flags]

Flags:
  -f, --frequency float    frequency to monitor in Hz (default 4.3392e+08)
  -d, --duration string    collection duration (default "60s")
  -o, --output string      output directory (default "./data")
      --gps-port string    GPS serial port (default "/dev/ttyUSB0")
      --synced-start       enable delayed/synchronized start time (default true)
  -c, --config string      config file (default is ./config.yaml)
  -v, --verbose            verbose output
  -h, --help               help for argus-collector
```

### Basic Usage Examples

```bash
# Default collection (433.92 MHz, 60 seconds, synchronized start)
./argus-collector --gps-port /dev/ttyACM0

# Custom frequency and duration
./argus-collector --frequency 915e6 --duration 30s --gps-port /dev/ttyACM0

# Immediate start (no synchronization delay)
./argus-collector --frequency 433.92e6 --synced-start=false --gps-port /dev/ttyACM0

# Use configuration file
./argus-collector --config my-config.yaml

# Verbose output for debugging
./argus-collector --frequency 433.92e6 --gps-port /dev/ttyACM0 --verbose
```

### Multi-Station Deployment

```bash
# Station 1 (synchronized start enabled by default)
./argus-collector --frequency 433.92e6 --output ./station1/ --gps-port /dev/ttyACM0

# Station 2 (different location, same frequency)
./argus-collector --frequency 433.92e6 --output ./station2/ --gps-port /dev/ttyACM0

# Station 3 (different location, same frequency)
./argus-collector --frequency 433.92e6 --output ./station3/ --gps-port /dev/ttyACM0

# All stations will automatically start collection at the same epoch second
```

### Frequency Examples

```bash
# Common ISM bands
./argus-collector --frequency 433.92e6   # 433.92 MHz (ISM)
./argus-collector --frequency 868e6      # 868 MHz (EU ISM)
./argus-collector --frequency 915e6      # 915 MHz (US ISM)

# Aviation bands
./argus-collector --frequency 121.5e6    # Emergency frequency
./argus-collector --frequency 1090e6     # ADS-B

# Amateur radio
./argus-collector --frequency 144.39e6   # 2m APRS
./argus-collector --frequency 446e6      # 70cm
```

### Configuration File

Create a `config.yaml` file for persistent settings:

```yaml
rtlsdr:
  frequency: 433.92e6      # Target frequency in Hz
  sample_rate: 2048000     # Sample rate in Hz
  gain: 20.7               # RF gain in dB
  device_index: 0          # RTL-SDR device index

gps:
  port: "/dev/ttyACM0"     # GPS serial port
  baud_rate: 9600          # Serial communication speed
  timeout: 30s             # GPS fix timeout

collection:
  duration: 60s            # Collection duration
  output_dir: "./data"     # Output directory
  file_prefix: "argus"     # File naming prefix
  synced_start: true       # Enable synchronized start based on epoch time

logging:
  level: "info"            # Log level
  file: "argus.log"        # Log file path
```

## Output Data Format

Argus Collector creates binary files with the `.dat` extension containing signal data and metadata optimized for TDOA analysis.

### File Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                           HEADER                                │
├─────────────────────────────────────────────────────────────────┤
│                         IQ SAMPLES                              │
└─────────────────────────────────────────────────────────────────┘
```

### Header Format (Binary, Little Endian)

| Field | Type | Size | Description |
|-------|------|------|-------------|
| Magic | string | 5 bytes | "ARGUS" file identifier |
| Format Version | uint16 | 2 bytes | File format version number |
| Frequency | uint64 | 8 bytes | Collection frequency (Hz) |
| Sample Rate | uint32 | 4 bytes | Sample rate (Hz) |
| Collection Time (Unix) | int64 | 8 bytes | RTL-SDR collection start time (seconds) |
| Collection Time (Nano) | int32 | 4 bytes | RTL-SDR collection nanoseconds |
| GPS Latitude | float64 | 8 bytes | GPS latitude (decimal degrees) |
| GPS Longitude | float64 | 8 bytes | GPS longitude (decimal degrees) |
| GPS Altitude | float64 | 8 bytes | GPS altitude (meters) |
| GPS Time (Unix) | int64 | 8 bytes | GPS timestamp (seconds) |
| GPS Time (Nano) | int32 | 4 bytes | GPS timestamp nanoseconds |
| Device Info Length | uint8 | 1 byte | Length of device info string |
| Device Info | string | variable | Device description |
| Collection ID Length | uint8 | 1 byte | Length of collection ID |
| Collection ID | string | variable | Unique collection identifier |
| Sample Count | uint32 | 4 bytes | Number of IQ samples |

### IQ Sample Data

Following the header, IQ samples are stored as consecutive float32 pairs:

```
Sample 1: [I₁ (float32)][Q₁ (float32)]
Sample 2: [I₂ (float32)][Q₂ (float32)]
...
Sample N: [Iₙ (float32)][Qₙ (float32)]
```

**Sample Format:**
- **I (In-phase)**: Real component, normalized to [-1.0, 1.0]
- **Q (Quadrature)**: Imaginary component, normalized to [-1.0, 1.0]
- **Encoding**: IEEE 754 single-precision, little-endian

### Timing Precision

The format stores two high-precision timestamps:

1. **Collection Time**: When RTL-SDR started sampling (for signal analysis)
2. **GPS Time**: GPS-synchronized timestamp (for TDOA calculations)

Both timestamps provide nanosecond resolution essential for accurate TDOA processing.

## Synchronized Collection

Argus Collector supports synchronized data collection across multiple stations using epoch-based timing:

### Synchronized Start Algorithm

When `synced_start` is enabled (default), the collection start time is calculated as:
1. Take current epoch time + 5 seconds
2. Calculate `sync_point = (epoch + 5) % 100`
3. Start collection at the next minute when seconds = `sync_point`

This ensures all stations start collection at the same second, critical for TDOA accuracy.

### Usage Examples

```bash
# Enable synchronized start (default)
./argus-collector --frequency 433.92e6 --synced-start

# Disable synchronized start for immediate collection
./argus-collector --frequency 433.92e6 --synced-start=false

# Multiple stations will automatically synchronize
# Station 1: Starts at HH:MM:23 (if sync_point = 23)
# Station 2: Starts at HH:MM:23 (same time)
# Station 3: Starts at HH:MM:23 (same time)
```

## TDOA Analysis Workflow

1. **Deploy Multiple Stations**: Place Argus Collectors at known GPS coordinates
2. **Synchronized Collection**: All stations automatically start at the same epoch second
3. **Time Difference Analysis**: Calculate arrival time differences using GPS timestamps
4. **Multilateration**: Triangulate transmitter position using TDOA algorithms

### Example Multi-Station Setup

```bash
# Station 1
./argus-collector --frequency 433.92e6 --output station1/ --config station1.yaml

# Station 2 (different location)
./argus-collector --frequency 433.92e6 --output station2/ --config station2.yaml

# Station 3 (different location)
./argus-collector --frequency 433.92e6 --output station3/ --config station3.yaml
```

## Data Analysis Tools

### Argus Reader Utility

The `argus-reader` is a specialized analysis tool for examining Argus Collector data files. It provides instant metadata inspection and comprehensive signal analysis capabilities without requiring external software.

**Key Features:**
- **⚡ Ultra-fast metadata display** (< 1ms) - instantly verify collection parameters
- **📊 IQ sample analysis** - examine raw signal data with magnitude and phase
- **📈 Statistical analysis** - calculate power, variance, and signal characteristics  
- **🗂️ File format validation** - verify data integrity and format compliance
- **💾 Memory efficient** - handles large files (1GB+) through smart sampling
- **🔍 GPS data inspection** - validate positioning and timing accuracy

The utility displays the contents of collected data files:

```bash
# Build the reader tool
make build-reader

# Display file metadata and sample information
./argus-reader data/argus_1234567890.dat

# Show IQ sample data (first 10 samples)
./argus-reader --samples data/argus_1234567890.dat

# Show detailed statistics
./argus-reader --stats data/argus_1234567890.dat

# Show more samples with statistics
./argus-reader --samples --limit 20 --stats data/argus_1234567890.dat
```

**Performance Characteristics:**
- **Metadata inspection**: Sub-millisecond display of file parameters
- **Large file support**: Handles 1GB+ files efficiently  
- **Smart sampling**: Statistical analysis uses representative samples for speed
- **Memory optimized**: Only loads data when specifically requested

**Example Output:**
```
╔══════════════════════════════════════════════════════════════╗
║                    ARGUS DATA FILE READER                   ║
╚══════════════════════════════════════════════════════════════╝

📁 File Information:
   Name: argus_1234567890.dat
   Size: 934.00 MB (979,370,089 bytes)
   Modified: 2025-07-13 10:53:15

📊 Collection Metadata:
┌─────────────────────────┬─────────────────────────────────────────┐
│ Parameter               │ Value                                   │
├─────────────────────────┼─────────────────────────────────────────┤
│ Frequency               │ 433.920 MHz                            │
│ Sample Rate             │ 2.048 MSps                             │
│ Collection Duration     │ 59.776 seconds                         │
│ GPS Location            │ 35.533198°, -97.621237°               │
│ Total Samples           │ 122,421,248                            │
└─────────────────────────┴─────────────────────────────────────────┘

real    0m0.002s  ← Ultra-fast metadata display
```

**Use Cases:**
- **File verification**: Quickly check collection parameters across multiple files
- **GPS validation**: Verify positioning accuracy and timing synchronization  
- **Signal analysis**: Examine IQ data quality and calculate power measurements
- **TDOA preparation**: Validate timing accuracy across multiple collection stations
- **Debugging**: Identify collection issues and verify file integrity

For complete usage documentation, see: [argus-reader-README.md](argus-reader-README.md)

### Programmatic File Reading

For custom analysis, use the `ReadFile` function:

```go
import "argus-collector/internal/filewriter"

metadata, samples, err := filewriter.ReadFile("data/argus_1234567890.dat")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Frequency: %.2f MHz\n", float64(metadata.Frequency)/1e6)
fmt.Printf("Samples: %d\n", len(samples))
fmt.Printf("Collection Time: %v\n", metadata.CollectionTime)
fmt.Printf("GPS Position: %.6f, %.6f\n", 
    metadata.GPSLocation.Latitude, metadata.GPSLocation.Longitude)
```

## Troubleshooting

### Build Issues

```bash
# Check Go version (1.19+ required)
go version

# Missing RTL-SDR development libraries
sudo apt-get install librtlsdr-dev build-essential  # Ubuntu/Debian
sudo dnf install rtl-sdr-devel gcc                  # Fedora/RHEL

# Missing dependencies
go mod download
go mod tidy

# Clean and rebuild
make clean
go clean -cache
make build

# Build without RTL-SDR for testing
make build-stub
```

### RTL-SDR Hardware Issues

```bash
# Test RTL-SDR device detection
rtl_test

# Check connected devices
lsusb | grep RTL

# Verify permissions (add user to plugdev group)
sudo usermod -a -G plugdev $USER
# Logout and login again

# Check udev rules for RTL-SDR
ls /etc/udev/rules.d/*rtl*

# Install RTL-SDR tools
sudo apt-get install rtl-sdr

# Test with different sample rates
rtl_test -s 2048000
```

### GPS Hardware Issues

```bash
# List GPS devices
ls /dev/tty* | grep -E "(ACM|USB)"

# Test GPS communication directly
cat /dev/ttyACM0  # Should show NMEA sentences

# Check GPS fix status with gpsd tools
sudo apt-get install gpsd gpsd-clients
sudo gpsd /dev/ttyACM0
cgps -s

# Test different baud rates
stty -F /dev/ttyACM0 9600
stty -F /dev/ttyACM0 4800

# Check device permissions
ls -la /dev/ttyACM0
sudo chmod 666 /dev/ttyACM0  # Temporary fix
```

### Runtime Issues

```bash
# Collection timeout error
# - Check GPS fix (may need clear sky view)
# - Verify GPS device connection
# - Try longer GPS timeout in config

# RTL-SDR "insufficient memory" error
# - Reduce sample rate or duration
# - Free system memory
# - Check available disk space

# "No RTL-SDR devices found"
# - Reconnect USB device
# - Check dmesg for USB errors
# - Try different USB port

# GPS fix timeout
# - Move to location with clear sky view
# - Check GPS antenna connection
# - Increase timeout in config.yaml
```

### Performance Optimization

```bash
# Monitor resource usage
top -p $(pgrep argus-collector)

# Check disk space (collections can be large)
df -h ./data/

# Optimize for long collections
# - Use faster storage (SSD)
# - Increase system memory
# - Close unnecessary applications

# Monitor GPS signal quality
sudo apt-get install gpsd gpsd-clients
gpsmon /dev/ttyACM0
```

### Common Error Messages

```bash
# "RTL-SDR support not compiled in"
# Solution: Rebuild with RTL-SDR tags
make clean && make build

# "failed to open GPS port"
# Solution: Check device path and permissions
ls -la /dev/ttyACM*
sudo usermod -a -G dialout $USER

# "GPS fix timeout"
# Solution: Move to clear sky location or increase timeout
./argus-collector --config config.yaml  # Edit GPS timeout

# "collection timeout"
# Solution: Check RTL-SDR connection and reduce duration
./argus-collector --duration 10s --gps-port /dev/ttyACM0
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- RTL-SDR community for hardware support
- Go NMEA library contributors
- TDOA research community

## Security Notice

This tool is designed for legitimate signal intelligence and research purposes. Users are responsible for compliance with local radio frequency regulations and privacy laws.