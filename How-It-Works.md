# How It Works: TDoA Signal Processing with Argus

## Overview

The Argus project implements a Time Difference of Arrival (TDoA) system for radio signal localization using multiple synchronized Software Defined Radio (SDR) receivers. This document explains the TDoA process, how Argus implements it, and the complete workflow from data collection to signal localization.

## What is TDoA (Time Difference of Arrival)?

Time Difference of Arrival is a passive localization technique that determines the position of a radio transmitter by measuring the time differences at which the signal arrives at multiple geographically separated receivers. Unlike triangulation methods that require signal strength measurements, TDoA relies purely on precise timing measurements.

### Basic TDoA Principle

When a transmitter broadcasts a signal, it reaches different receivers at different times based on:
- The speed of light (radio waves travel at ~299,792,458 m/s)
- The distance from transmitter to each receiver
- The geometry of the receiver network

By measuring these time differences with nanosecond precision, we can calculate hyperbolic curves that intersect at the transmitter's location.

### Mathematical Foundation

For a signal received at receivers A, B, and C:
- Time difference A-B creates one hyperbolic curve
- Time difference A-C creates another hyperbolic curve  
- Time difference B-C provides validation/redundancy
- The intersection of these curves pinpoints the transmitter location

**Accuracy Requirements:**
- 1 nanosecond timing error = ~30 cm position error
- GPS timing precision: ~10-40 nanoseconds
- Target accuracy: <100 meter positioning

## Argus TDoA Implementation

### System Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ Station A   │    │ Station B   │    │ Station C   │
│ RTL-SDR     │    │ RTL-SDR     │    │ RTL-SDR     │
│ GPS         │    │ GPS         │    │ GPS         │
│ argus-coll. │    │ argus-coll. │    │ argus-coll. │
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       └─────── Network ───┼─────── Network ───┘
                           │
                    ┌─────────────┐
                    │ Processing  │
                    │ Server      │  
                    │ argus-proc. │
                    └─────────────┘
```

### Key Components

1. **argus-collector**: Synchronized SDR data collection
2. **argus-reader**: Data validation and analysis
3. **argus-processor**: TDoA correlation and localization
4. **GPS synchronization**: Nanosecond-precision timing
5. **Network coordination**: Multi-station orchestration

## Data Collection Process

### Step 1: Station Deployment

**Hardware Requirements per Station:**
- RTL-SDR dongle (RTL2832U + R820T2/R828D)
- GPS receiver (NMEA or gpsd compatible)
- Computing device (Raspberry Pi, laptop, etc.)
- Antenna suitable for target frequency
- Network connectivity

**Geographic Considerations:**
- Minimum 3 stations for 2D localization
- 4+ stations for 3D localization and redundancy
- Station separation: 1-50 km depending on target range
- Baseline diversity: avoid collinear arrangements

### Step 2: Time Synchronization

```bash
# GPS-based synchronization
./argus-collector --gps-mode=nmea --gps-port=/dev/ttyACM0

# Network time coordination
# All stations automatically synchronize to GPS epoch boundaries
# Collections start simultaneously across all stations
```

**Synchronization Process:**
1. Each station acquires GPS lock
2. GPS provides nanosecond-precision timestamps  
3. Collectors coordinate start times via network
4. Data collection begins simultaneously across all stations
5. Each sample tagged with GPS timestamp

### Step 3: RF Data Collection

```bash
# Basic collection (5 seconds at 162.400 MHz)
./argus-collector --frequency=162400000 --duration=5s

# Advanced collection with custom settings
./argus-collector \
  --frequency=162400000 \
  --sample-rate=2048000 \
  --duration=10s \
  --gain=20.7 \
  --gps-mode=nmea \
  --gps-port=/dev/ttyACM0
```

**Collection Parameters:**
- **Frequency**: Target signal frequency (Hz)
- **Sample Rate**: Typically 1-3 MSps for TDoA
- **Duration**: Collection time (seconds/minutes)
- **Gain**: RTL-SDR gain setting (0-50 dB)
- **GPS Mode**: Timing source (nmea/gpsd/manual)

### Step 4: Data Storage

Each station produces binary data files containing:
- **Header Metadata**: Frequency, sample rate, timestamps, GPS coordinates
- **IQ Samples**: Complex floating-point data (I/Q pairs)
- **Timing Information**: Nanosecond-precision GPS timestamps
- **Device Configuration**: Gain settings, hardware info

**File Format:**
```
ARGUS Binary Format (.dat):
┌─────────────┐
│ Magic "ARGUS"│ 5 bytes
│ Version     │ 2 bytes  
│ Frequency   │ 8 bytes
│ Sample Rate │ 4 bytes
│ Timestamps  │ 24 bytes
│ GPS Data    │ 36 bytes  
│ Device Info │ Variable
│ Sample Count│ 4 bytes
├─────────────┤
│ IQ Samples  │ 8 bytes per sample
│ (Complex64) │ (float32 I + float32 Q)
│     ...     │
└─────────────┘
```

## Data Processing Workflow

### Step 1: Data Collection Validation

Use `argus-reader` to validate each station's data quality:

```bash
# Quick metadata check
./argus-reader station1_data.dat

# Detailed quality analysis  
./argus-reader --stats station1_data.dat

# Signal visualization
./argus-reader --graph station1_data.dat
```

### Step 2: Cross-Correlation Processing

```bash
# TDoA processing (future argus-processor)
./argus-processor \
  --station1=station1_data.dat \
  --station2=station2_data.dat \
  --station3=station3_data.dat \
  --output=tdoa_results.json
```

**Cross-Correlation Steps:**
1. **Time Alignment**: Align data using GPS timestamps
2. **Signal Detection**: Identify target signals in each dataset
3. **Cross-Correlation**: Calculate correlation between station pairs
4. **Peak Detection**: Find correlation peaks indicating signal arrival
5. **Time Difference Extraction**: Measure precise timing differences
6. **Hyperbolic Positioning**: Calculate intersection of hyperbolic curves

### Step 3: Localization Calculation

**Mathematical Process:**
1. **Time Differences**: Extract TDoA measurements (τ₁₂, τ₁₃, τ₂₃)
2. **Distance Differences**: Convert to distance (c × τ, where c = speed of light)
3. **Hyperbolic Equations**: Generate hyperbolic constraint equations
4. **Least Squares Solution**: Solve non-linear system for (x, y) coordinates
5. **Error Analysis**: Calculate position uncertainty ellipse

## Data Quality Validation with argus-reader

### Basic Validation

```bash
# Verify file integrity and metadata
./argus-reader data_file.dat
```

**Checks Performed:**
- File format validation
- GPS timing accuracy
- Sample count verification
- Collection duration accuracy
- Device configuration review

### Signal Quality Analysis

```bash
# Statistical analysis
./argus-reader --stats data_file.dat
```

**Quality Metrics:**
- **Signal Strength**: Received power levels (-30 to -100 dBm typical)
- **Signal-to-Noise Ratio**: SNR measurements (>10 dB preferred)
- **Dynamic Range**: Signal variation analysis
- **I/Q Balance**: Phase and amplitude balance
- **Noise Floor**: Background noise characterization

### Sample Data Inspection

```bash
# View raw sample data
./argus-reader --samples data_file.dat | head -100

# Hexadecimal dump for debugging
./argus-reader --hex data_file.dat | head -50
```

**Use Cases:**
- Verify IQ sample integrity
- Check for clipping/saturation
- Analyze signal characteristics
- Debug collection issues

### Visual Analysis

```bash
# Signal magnitude over time
./argus-reader --graph data_file.dat
```

**Graph Analysis:**
- Signal presence verification
- Amplitude variations over time
- Noise floor consistency
- Signal burst detection

### Quality Assessment Criteria

**Good Quality Indicators:**
- GPS timestamp accuracy: <40 ns
- Signal-to-Noise Ratio: >15 dB  
- No sample clipping (magnitude < 1.0)
- Consistent noise floor
- Complete sample count
- Stable device configuration

**Quality Issues to Flag:**
- Poor GPS synchronization (>100 ns error)
- Low SNR (<5 dB)
- Sample clipping/saturation
- Excessive noise variations  
- Incomplete or corrupted data
- Inconsistent gain settings

## TDoA Processing Considerations

### Accuracy Factors

**Timing Precision:**
- GPS accuracy: ±10-40 nanoseconds
- Sample rate impact: Higher rates improve resolution
- Clock stability: Crystal oscillator drift effects

**Signal Characteristics:**
- Bandwidth: Wider signals provide better correlation
- Duration: Longer signals improve SNR
- Modulation: Some modulations correlate better than others

**Geometric Factors:**
- Baseline length: Longer baselines improve accuracy
- Station geometry: Avoid collinear arrangements
- Target position: Accuracy degrades outside station network

### Limitations and Challenges

**Technical Limitations:**
- Requires synchronized timing across stations
- Limited by GPS timing accuracy
- Sensitive to multipath propagation
- Requires adequate signal strength at all stations

**Environmental Factors:**
- Atmospheric propagation effects
- Terrain blocking/reflecting signals  
- RF interference and noise
- Weather impact on propagation

**Operational Constraints:**
- Real-time processing requirements
- Network connectivity between stations
- Data storage and transfer capabilities
- Computational processing power

## Future Enhancements

### Advanced Processing
- Real-time TDoA processing
- Machine learning signal classification
- Automated quality scoring
- Multi-frequency correlation

### System Improvements  
- Enhanced GPS timing (PPS integration)
- Network time protocol backup
- Distributed processing architecture
- Cloud-based correlation services

### Analysis Tools
- Interactive visualization
- Statistical quality metrics
- Performance benchmarking
- Calibration validation

---

This TDoA implementation provides a foundation for passive radio signal localization using commodity SDR hardware and GPS timing. The Argus system's modular design allows for flexible deployment scenarios while maintaining the precision required for accurate geolocation.