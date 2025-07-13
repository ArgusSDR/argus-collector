# Argus Collector - Project Plan

## Project Description

Argus Collector is a Go-based data collection program designed to interface with RTL-SDR (Software Defined Radio) hardware for signal intelligence gathering. The program simultaneously collects radio frequency data and GPS positioning information to support Time Difference of Arrival (TDOA) analysis for transmitter localization.

### Primary Objectives

- **Signal Collection**: Capture IQ (In-phase/Quadrature) samples from RTL-SDR at specified frequencies
- **GPS Integration**: Record precise timing and location data synchronized with signal collection
- **Data Storage**: Write collected data in a standardized format compatible with TDOA processing
- **Multi-Station Support**: Enable coordinated data collection across multiple geographic locations
- **Transmitter Localization**: Support downstream TDOA analysis to triangulate transmitter positions

## Technical Architecture

### System Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   RTL-SDR       │    │   GPS Module    │    │  File Writer    │
│   Interface     │    │   Interface     │    │   Module        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Main Collector │
                    │    Controller   │
                    └─────────────────┘
```

### Core Modules

#### 1. RTL-SDR Interface Module
- **Library**: `github.com/jpoirier/gortlsdr`
- **Functions**:
  - Device initialization and configuration
  - Frequency tuning and gain control
  - IQ sample collection with precise timing
  - Error handling and device cleanup

#### 2. GPS Data Module
- **Libraries**: 
  - `github.com/adrianmo/go-nmea` (NMEA parsing)
  - `go.bug.st/serial` (Serial communication)
- **Functions**:
  - GPS receiver connection management
  - NMEA sentence parsing for position/time
  - GPS fix quality monitoring
  - Time synchronization with signal collection

#### 3. Data Collection Controller
- **Functions**:
  - Coordinate simultaneous RTL-SDR and GPS data collection
  - Manage collection timing and duration
  - Handle system interrupts and graceful shutdown
  - Implement precise timestamping for TDOA compatibility

#### 4. File Output Module
- **Format**: Custom binary format with metadata header
- **Structure**:
  ```
  Header: {
    frequency: float64
    sample_rate: uint32
    collection_time: timestamp
    gps_location: {lat, lon, alt}
    gps_timestamp: timestamp
    device_info: string
    file_format_version: uint16
  }
  Data: []complex64 (IQ samples)
  ```

## Implementation Plan

### Phase 1: Core Infrastructure (Weeks 1-2)
1. **Project Setup**
   - Initialize Go module and dependency management
   - Set up project directory structure
   - Configure logging and error handling frameworks

2. **RTL-SDR Integration**
   - Implement RTL-SDR device discovery and initialization
   - Create frequency tuning and sample rate configuration
   - Develop IQ data collection pipeline
   - Add device health monitoring and error recovery

3. **GPS Module Development**
   - Implement serial port communication for GPS
   - Create NMEA sentence parser for position/time data
   - Add GPS fix quality assessment
   - Implement GPS timeout and retry logic

### Phase 2: Data Management (Weeks 3-4)
1. **File Format Design**
   - Define binary file structure for efficient storage
   - Implement metadata header with collection parameters
   - Create data validation and integrity checking
   - Add file compression options for storage efficiency

2. **Configuration System**
   - Implement YAML-based configuration file parsing
   - Create command-line argument processing
   - Add configuration validation and default values
   - Implement environment variable overrides

3. **Timing Synchronization**
   - Develop GPS-synchronized timing system
   - Implement precise timestamp generation
   - Create time drift compensation mechanisms
   - Add timing accuracy validation

### Phase 3: User Interface & Testing (Weeks 5-6)
1. **Command Line Interface**
   - Implement CLI using `github.com/spf13/cobra`
   - Create help documentation and usage examples
   - Add progress indicators and status reporting
   - Implement verbose logging options

2. **Error Handling & Resilience**
   - Comprehensive error handling throughout system
   - Graceful degradation for missing GPS data
   - Disk space monitoring and management
   - Signal interruption handling (SIGINT/SIGTERM)

3. **Testing & Validation**
   - Unit tests for all core modules
   - Integration tests with mock hardware
   - Field testing with actual RTL-SDR and GPS hardware
   - Performance benchmarking and optimization

### Phase 4: Advanced Features (Weeks 7-8)
1. **Multi-Station Coordination**
   - Network synchronization protocols for coordinated collection
   - Time server integration for precise timing
   - Distributed configuration management
   - Data collection scheduling and automation

2. **Data Analysis Integration**
   - Export utilities for TDOA analysis tools
   - Data format converters for legacy systems
   - Statistical analysis of collection quality
   - Signal strength and noise floor monitoring

## Technology Stack

### Core Dependencies
- **RTL-SDR Interface**: `github.com/jpoirier/gortlsdr`
- **GPS/NMEA Processing**: `github.com/adrianmo/go-nmea`
- **Serial Communication**: `go.bug.st/serial`
- **Configuration Management**: `github.com/spf13/viper`
- **Command Line Interface**: `github.com/spf13/cobra`
- **Structured Logging**: `github.com/sirupsen/logrus`
- **Testing Framework**: Go standard library `testing`

### Optional Enhancements
- **Data Format**: `github.com/gonum/hdf5` for scientific data storage
- **Compression**: `github.com/klauspost/compress` for efficient storage
- **Monitoring**: `github.com/prometheus/client_golang` for metrics collection

## Configuration Example

```yaml
# config.yaml
rtlsdr:
  frequency: 433.92e6      # Target frequency in Hz
  sample_rate: 2.048e6     # Sample rate in Hz
  gain: 20.7               # RF gain in dB
  device_index: 0          # RTL-SDR device index

gps:
  port: "/dev/ttyUSB0"     # GPS serial port
  baud_rate: 9600          # Serial communication speed
  timeout: 30s             # GPS fix timeout

collection:
  duration: 10s            # Collection duration
  output_dir: "./data"     # Output directory
  file_prefix: "argus"     # File naming prefix

logging:
  level: "info"            # Log level (debug, info, warn, error)
  file: "argus.log"        # Log file path
```

## Usage Examples

```bash
# Basic signal collection
./argus-collector --frequency 433.92e6 --duration 10s

# Advanced configuration
./argus-collector \
  --config config.yaml \
  --frequency 433.92e6 \
  --duration 30s \
  --output ./collection_001 \
  --gps-port /dev/ttyUSB0 \
  --sample-rate 2.048e6 \
  --gain 20.7 \
  --verbose

# Continuous monitoring mode
./argus-collector --config monitor.yaml --continuous --interval 60s
```

## TDOA Analysis Integration

The collected data files are designed for seamless integration with TDOA analysis workflows:

1. **Precise Timing**: GPS-synchronized timestamps enable accurate time difference calculations
2. **Location Metadata**: Collector positions support multilateration algorithms
3. **Signal Quality**: Raw IQ data preserves all signal characteristics for analysis
4. **Standardized Format**: Consistent file structure across multiple collection stations
5. **Batch Processing**: File naming and metadata support automated analysis pipelines

## Success Criteria

- **Accuracy**: Sub-microsecond timing precision for TDOA compatibility
- **Reliability**: 99%+ successful data collection rate under normal conditions
- **Performance**: Real-time processing of 2+ MHz sample rates
- **Usability**: Single-command operation with sensible defaults
- **Portability**: Cross-platform compatibility (Linux, Windows, macOS)
- **Documentation**: Comprehensive user and developer documentation

## Future Enhancements

- **Real-time Analysis**: Live signal processing and transmitter detection
- **Web Interface**: Browser-based monitoring and control dashboard
- **Cloud Integration**: Automated data upload and distributed analysis
- **Machine Learning**: Automatic signal classification and interference detection
- **Hardware Integration**: Support for additional SDR platforms and GPS receivers