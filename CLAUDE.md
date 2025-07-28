# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building
- `make build` - Build with RTL-SDR support (requires librtlsdr-dev)
- `make build-stub` - Build without RTL-SDR for testing/development
- `make build-reader` - Build the argus-reader utility for analyzing data files
- `make build-all-tools` - Build both collector and reader
- `make deps` - Download and tidy Go dependencies

### Testing & Quality
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report
- `make fmt` - Format code
- `make vet` - Vet code for issues
- `make lint` - Lint code (requires golangci-lint)
- `make check` - Run all quality checks (fmt, vet, lint, test)

### Platform Builds
- `make build-all` - Build for all platforms (Linux, Windows, macOS)
- `make build-linux` - Linux-specific build
- `make package` - Create release packages

### Quick Testing
- `make test-run` - Quick 2-second test run
- `./argus-collector --duration 2s --gps-mode=nmea --gps-port /dev/ttyACM0` - Manual NMEA test
- `./argus-collector --duration 2s --gps-mode=gpsd --gpsd-host=localhost` - Manual gpsd test
- `./argus-collector --duration 2s --gps-mode=manual --latitude=35.533 --longitude=-97.621` - Manual coordinates test

## Project Architecture

### Core Components
- **main.go**: CLI entry point using Cobra for command-line interface
- **internal/collector/**: Main collection orchestration and timing synchronization
- **internal/rtlsdr/**: RTL-SDR hardware interface with build tags for stub/real implementations
- **internal/gps/**: GPS receiver interface supporting NMEA serial and gpsd daemon modes
- **internal/filewriter/**: Custom binary format for TDOA-compatible data storage
- **internal/config/**: Configuration management with YAML support
- **cmd/argus-reader/**: Data file analysis utility

### Key Design Patterns
- **Build Tags**: Uses `rtlsdr` build tag to switch between real hardware and stub implementations
- **Synchronized Collection**: Implements epoch-based timing for multi-station coordination
- **Graceful Shutdown**: Signal handling for clean hardware cleanup
- **GPS Modes**: Support for NMEA serial, gpsd daemon, or manual coordinates with backward compatibility

### Configuration
- **config.yaml**: Main configuration file with RTL-SDR, GPS, collection, and logging settings
- **Viper**: Configuration library supporting file, environment, and CLI flag overrides
- **GPS Modes**: Set `gps.mode` to "nmea", "gpsd", or "manual" for different GPS interfaces
- **GPS Backward Compatibility**: Legacy `gps.disable: true` still supported

### Binary Data Format
Custom format optimized for TDOA analysis:
- Magic header "ARGUS" with version
- Metadata: frequency, sample rate, collection/GPS timestamps, location
- IQ samples as float32 pairs (I/Q components)
- Little-endian encoding throughout

### Multi-Station Support
- **Synchronized Start**: Automatic epoch-based timing coordination across stations
- **TDOA Ready**: Nanosecond-precision timestamps for time difference calculations
- **Location Metadata**: GPS coordinates stored with each collection

## Development Workflow

### Prerequisites
- Go 1.22.2+
- For RTL-SDR builds: `sudo apt-get install librtlsdr-dev build-essential`
- For analysis: Build argus-reader with `make build-reader`

### Common Tasks
1. **Stub Development**: Use `make build-stub` when developing without RTL-SDR hardware
2. **Testing Changes**: Use `make test-run` for quick validation
3. **Data Analysis**: Use `./argus-reader data/filename.dat` to examine collected data
4. **Quality Checks**: Run `make check` before committing changes

### Hardware Integration Notes
- RTL-SDR interface uses CGO - requires proper C library linking
- GPS communication supports multiple modes:
  - **NMEA**: Direct serial port communication (typically /dev/ttyACM0 or /dev/ttyUSB0)
  - **GPSD**: Connection to gpsd daemon (typically localhost:2947)
  - **Manual**: Fixed coordinates for testing or known locations
- Build system handles cross-compilation (Windows/macOS builds disable CGO automatically)
- Hardware permissions may require adding user to plugdev/dialout groups

## File Structure Insights
- **Build tags separation**: rtlsdr.go vs rtlsdr_stub.go for hardware abstraction
- **Modular design**: Each internal package handles a specific concern
- **Reader utility**: Separate command for data file analysis and validation
- **Documentation**: Comprehensive README.md with usage examples and troubleshooting

## GPS Integration Notes

### GPS Mode Selection
Use `--gps-mode` flag or `gps.mode` config to select GPS interface:
- `--gps-mode=nmea` - Direct serial communication (default)
- `--gps-mode=gpsd` - Connect to gpsd daemon
- `--gps-mode=manual` - Use fixed coordinates

### NMEA Mode Flags
- `--gps-port=/dev/ttyACM0` - Serial port device
- Configuration: `port`, `baud_rate` in config file

### GPSD Mode Flags  
- `--gpsd-host=localhost` - GPSD host address
- `--gpsd-port=2947` - GPSD port number
- Configuration: `gpsd_host`, `gpsd_port` in config file

### Manual Mode Flags
- `--latitude=35.533` - Fixed latitude in decimal degrees
- `--longitude=-97.621` - Fixed longitude in decimal degrees  
- `--altitude=365.0` - Fixed altitude in meters
- Configuration: `manual_latitude`, `manual_longitude`, `manual_altitude` in config file

### Backward Compatibility
- `--disable-gps` flag still supported (equivalent to `--gps-mode=manual`)
- `gps.disable: true` in config file still supported