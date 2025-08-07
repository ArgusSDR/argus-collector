# Argus SDR - Time Difference of Arrival Signal Processing System

**Multi-station synchronized RF collection and analysis for transmitter localization using RTL-SDR hardware**

Argus is a complete TDoA (Time Difference of Arrival) signal processing system that enables precise localization of radio transmitters using multiple synchronized RTL-SDR receivers. The system provides GPS-synchronized data collection, comprehensive signal analysis, and automated TDoA processing to determine transmitter positions with high accuracy.

## System Components

### Core Applications

- **[argus-collector](argus-collector-README.md)** - GPS-synchronized RTL-SDR data collection
- **[argus-reader](argus-reader-README.md)** - Data validation and signal analysis  
- **[argus-processor](argus-processor-README.md)** - TDoA processing and localization

### Documentation

- **[How-It-Works.md](How-It-Works.md)** - Complete TDoA theory and system workflow
- **[CLAUDE.md](CLAUDE.md)** - Development commands and architecture guide

## Quick Start

### Prerequisites

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install librtlsdr-dev build-essential git go

# Fedora/RHEL
sudo dnf install rtl-sdr-devel gcc git go

# Arch Linux
sudo pacman -S rtl-sdr git base-devel go
```

### Build All Tools

```bash
git clone <repository-url>
cd argus-collector
make build-all-tools
```

### Basic TDoA Workflow

```bash
# 1. Collect synchronized data from multiple stations (with AGC optimization)
./argus-collector --frequency=162400000 --duration=30s \
  --collection-id=station1 --gps-mode=nmea --gps-port=/dev/ttyACM0 --gain-mode=auto

./argus-collector --frequency=162400000 --duration=30s \
  --collection-id=station2 --gps-mode=gpsd --gpsd-host=localhost --gain-mode=auto

./argus-collector --frequency=162400000 --duration=30s \
  --collection-id=station3 --gps-mode=manual --latitude=35.533 --longitude=-97.621 --gain-mode=manual --gain=28.1

# 2. Validate data quality with enhanced signal analysis  
./argus-reader --stats --graph station1_data.dat
./argus-reader --stats --graph station2_data.dat  
./argus-reader --stats --graph station3_data.dat

# 3. Process for transmitter location
./argus-processor --input "argus-*_timestamp.dat" --verbose
```

## Key Features

### Enhanced v0.2.0-beta Features
- **Software-based AGC** - Intelligent automatic gain control for optimal signal capture
- **Improved GPSD integration** - Accurate satellite count reporting (satellites used in fix)
- **Advanced signal analysis** - SNR calculation, noise floor analysis, and signal strength metrics
- **Verbose logging control** - Detailed AGC and GPS debugging with --verbose flag

### Synchronized Collection
- **Nanosecond GPS timing** for multi-station coordination
- **Automatic epoch synchronization** across distributed receivers
- **Sub-sample timing accuracy** essential for TDoA processing

### Signal Analysis
- **Real-time streaming display** of IQ samples and hex dumps  
- **Comprehensive statistics** including SNR, signal strength, and noise floor analysis
- **Advanced signal visualization** with ASCII graphs and quality metrics
- **Quality validation** with automated analysis recommendations

### TDoA Processing  
- **Cross-correlation analysis** between receiver pairs
- **Multiple output formats** (KML, GeoJSON, CSV)
- **Confidence assessment** with probability heatmaps
- **Google Earth visualization** of results

## Hardware Requirements

### Per Station
- **RTL-SDR Device**: RTL2832U-based SDR (RTL-SDR Blog v3/v4 recommended)
- **GPS Receiver**: NMEA-compatible with external antenna
- **Computing Platform**: Raspberry Pi 4+ or equivalent x86 system
- **Storage**: SSD recommended for high sample rate collections
- **Network**: Ethernet for multi-station coordination

### System Deployment
- **Minimum 3 stations** for 2D localization  
- **4+ stations** for 3D localization and redundancy
- **1-50 km baseline** depending on target range
- **Non-collinear geometry** for optimal accuracy

## Build Options

### Standard Build
```bash
# Build all tools with RTL-SDR support
make build-all-tools

# Individual components
make build          # argus-collector
make build-reader   # argus-reader  
make build-processor # argus-processor
```

### Development Build
```bash
# Build without RTL-SDR hardware (stub mode)
make build-stub

# Testing and quality checks
make test
make check          # fmt, vet, lint, test
```

### Cross-Platform Build
```bash
# All platforms
make build-all

# Specific platforms
make build-linux
make build-windows
make build-darwin

# Release packages
make package
```

### Manual Build
```bash
# With RTL-SDR support (requires librtlsdr-dev)
go build -tags rtlsdr -o argus-collector .
go build -o argus-reader ./cmd/argus-reader
go build -o argus-processor ./cmd/argus-processor

# Without RTL-SDR (stub functions)
go build -o argus-collector .
```

## System Workflow

### 1. Station Deployment
Deploy RTL-SDR stations at known GPS coordinates with optimal geometric diversity for target coverage area.

### 2. Synchronized Collection  
All stations automatically coordinate start times using GPS epoch boundaries to ensure nanosecond-level synchronization.

### 3. Data Validation
Use argus-reader to verify signal quality, GPS timing accuracy, and hardware configuration across all stations.

### 4. TDoA Processing
Process synchronized data files to calculate time differences and triangulate transmitter position with confidence bounds.

### 5. Result Visualization
View results in Google Earth (KML), web maps (GeoJSON), or spreadsheet analysis (CSV).

## Data Flow

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ Station A   │    │ Station B   │    │ Station C   │
│ RTL-SDR     │    │ RTL-SDR     │    │ RTL-SDR     │
│ GPS Sync    │    │ GPS Sync    │    │ GPS Sync    │
└──────┬──────┘    └──────┬──────┘    └──────┬──────┘
       │                  │                  │
       ▼                  ▼                  ▼
   station_a.dat      station_b.dat      station_c.dat
       │                  │                  │
       └──────────┬───────────────┬──────────┘
                  ▼               ▼
            argus-reader    argus-processor
          (validation)      (TDoA analysis)
                              │
                              ▼
                        Transmitter
                        Location
                    (KML/GeoJSON/CSV)
```

## Performance Specifications

### Timing Accuracy
- **GPS Precision**: ±10-40 nanoseconds
- **Synchronization**: <1 nanosecond between stations
- **Position Accuracy**: <100 meters typical

### Data Throughput  
- **Sample Rates**: 512 kSps - 3.2 MSps
- **File Sizes**: ~460 MB per minute at 1 MSps
- **Processing Speed**: Real-time correlation on modern hardware

### System Scalability
- **Stations**: 3+ (no upper limit)
- **Coverage Area**: 10 km - 200+ km baseline
- **Concurrent Collections**: Limited by hardware resources

## Configuration

### Basic Configuration
```yaml
# config.yaml
collection:
  frequency: 162400000
  sample_rate: 2048000
  duration: "30s"

rtlsdr:
  gain_mode: "auto"    # Enable AGC (or "manual" for fixed gain)
  gain: 20.7           # Used only in manual mode
  bias_tee: false
  
gps:
  mode: "nmea"         # nmea, gpsd, or manual
  port: "/dev/ttyACM0"

logging:
  level: "info"        # Use "debug" or --verbose for AGC details
```

### Multi-Station Setup
```yaml
# station1.yaml
collection:
  collection_id: "north_site"
  
gps:
  mode: "nmea"
  port: "/dev/ttyACM0"

rtlsdr:
  serial_number: "NORTH001"
```

## Component Documentation

| Component | Purpose | Documentation |
|-----------|---------|---------------|
| **argus-collector** | GPS-synchronized data collection | [argus-collector-README.md](argus-collector-README.md) |
| **argus-reader** | Data validation and analysis | [argus-reader-README.md](argus-reader-README.md) |
| **argus-processor** | TDoA processing and localization | [argus-processor-README.md](argus-processor-README.md) |
| **System Theory** | TDoA principles and workflow | [How-It-Works.md](How-It-Works.md) |
| **Development** | Build commands and architecture | [CLAUDE.md](CLAUDE.md) |

## Use Cases

### Emergency Services
- **Search and Rescue**: Locate emergency beacons and PLBs
- **Public Safety**: Track radio communications for incident response
- **Aircraft Location**: Complement ADS-B with TDoA positioning

### Research and Development
- **RF Propagation**: Study signal characteristics across geographic areas
- **Algorithm Development**: Test and validate TDoA processing techniques
- **Educational**: Demonstrate principles of radio direction finding

### Regulatory Compliance
- **Spectrum Monitoring**: Locate unauthorized transmissions
- **Interference Hunting**: Identify sources of harmful interference
- **License Enforcement**: Verify compliance with geographical restrictions

## Troubleshooting

### Build Issues
```bash
# Check Go version (1.22+ required)
go version

# Install missing dependencies
make deps

# Clean and rebuild
make clean && make build-all-tools
```

### Hardware Issues
```bash
# Test RTL-SDR
rtl_test

# Test GPS
cat /dev/ttyACM0

# Check permissions
sudo usermod -a -G plugdev,dialout $USER
```

### GPS Synchronization
```bash
# Verify GPS fix
cgps -s

# Check timing accuracy
./argus-reader --stats data_file.dat | grep "GPS"
```

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/new-feature`)
3. Follow development guidelines in [CLAUDE.md](CLAUDE.md)
4. Test with `make check`
5. Submit pull request

## License

This project is licensed under the MIT License - see LICENSE file for details.

## Security and Legal Notice

This system is designed for legitimate signal intelligence, research, and emergency response applications. Users are responsible for compliance with:

- Local radio frequency regulations
- Privacy and surveillance laws  
- Proper authorization for signal monitoring
- Ethical use of location information

**The authors assume no responsibility for misuse of this technology.**

---

**For detailed component documentation, see the individual README files linked above.**

**For complete system theory and workflow, see [How-It-Works.md](How-It-Works.md).**