# Argus DTOA - Time Difference of Arrival Analysis Tool

Argus DTOA is a sophisticated signal analysis tool that processes multiple Argus Collector data files to calculate the GPS location of a signal transmitter using Time Difference of Arrival (TDOA) analysis.

## Overview

TDOA analysis is a passive geolocation technique that determines the position of a signal source by measuring the time differences of signal arrival at multiple receiver stations. By analyzing these time differences, the system can triangulate the transmitter's location using hyperbolic positioning algorithms.

## Features

### 🎯 **Advanced TDOA Analysis**
- **Multiple Algorithms**: Hyperbolic, Least-Squares, and Newton-Raphson positioning
- **High Precision**: Sub-meter accuracy with optimal station geometry
- **Quality Assessment**: Automated confidence scoring and geometry analysis
- **Signal Validation**: Comprehensive signal strength and quality metrics

### 📊 **Signal Processing**
- **Cross-Correlation Analysis**: Advanced signal processing for precise TOA calculation
- **Signal Quality Metrics**: Power analysis, dynamic range, and SNR estimation
- **Configurable Analysis**: Adjustable sample limits up to 1M samples per station
- **Multi-Station Synchronization**: Collection time validation and drift detection

### 📈 **Results & Visualization**
- **Professional Output**: Formatted tables with comprehensive analysis data
- **Multiple Coordinate Systems**: Decimal degrees, DMS, and map coordinates
- **Accuracy Estimation**: GDOP calculation and confidence intervals
- **Export Capabilities**: Detailed results with intermediate analysis data

## Installation

The argus-dtoa tool is built as part of the Argus Collector suite:

```bash
# Build argus-dtoa specifically
make build-dtoa

# Build all tools
make build-all-tools
```

## Requirements

### Minimum Station Requirements
- **3 Stations**: Required for 2D positioning
- **4+ Stations**: Recommended for 3D positioning and improved accuracy
- **Station Separation**: Minimum 100m separation recommended
- **Time Synchronization**: All stations should collect data within ±60 seconds

### Data File Requirements
- Valid Argus Collector `.dat` files
- Consistent frequency across all stations
- GPS location data for each station
- Signal data with sufficient quality (SNR > 10dB recommended)

## Usage

### Basic Usage

```bash
# Analyze 3 stations with default settings
./argus-dtoa station1.dat station2.dat station3.dat

# Verbose analysis with detailed output
./argus-dtoa station1.dat station2.dat station3.dat --verbose

# Use specific algorithm
./argus-dtoa *.dat --algorithm least-squares
```

### Advanced Usage

```bash
# High-confidence analysis with results export
./argus-dtoa data/*.dat --confidence 0.8 --export --output results.txt

# Analysis with map coordinates and detailed export
./argus-dtoa *.dat --map --verbose --export

# Custom analysis parameters
./argus-dtoa *.dat --algorithm newton-raphson --max-samples 500000 --speed 299792458
```

## Command Line Options

### Required Arguments
- `[file1.dat] [file2.dat] [file3.dat] ...`: Input data files (minimum 3 required)

### Analysis Options
- `-a, --algorithm string`: TDOA algorithm selection
  - `hyperbolic` (default): Fast geometric positioning
  - `least-squares`: Enhanced accuracy through optimization
  - `newton-raphson`: Iterative high-precision positioning

### Quality Control
- `--confidence float`: Minimum confidence threshold (0.0-1.0, default: 0.5)
- `--max-samples int`: Maximum samples to analyze per station (default: 1,000,000)
- `--speed float`: RF signal speed in m/s (default: 299,792,458)

### Output Options
- `-v, --verbose`: Detailed analysis output with step-by-step processing
- `-o, --output string`: Save results to specified file
- `--export`: Export detailed intermediate data and analysis
- `--map`: Include Google Maps and OpenStreetMap coordinate URLs

## Output Format

### Standard Output
```
╔══════════════════════════════════════════════════════════════╗
║                    ARGUS DTOA ANALYZER                      ║
║             Time Difference of Arrival Analysis             ║
╚══════════════════════════════════════════════════════════════╝

📡 Collection Stations (3 total):
┌──────┬─────────────────────┬──────────────┬───────────────┬─────────────────────┬─────────────┐
│ #    │ Station             │ Latitude     │ Longitude     │ Collection Time     │ File Size   │
├──────┼─────────────────────┼──────────────┼───────────────┼─────────────────────┼─────────────┤
│ 1    │ Station-001         │  40.12345678 │  -74.12345678 │ 15:30:45.123        │    934.5 MB │
│ 2    │ Station-002         │  40.13456789 │  -74.13456789 │ 15:30:45.124        │    945.2 MB │
│ 3    │ Station-003         │  40.14567890 │  -74.14567890 │ 15:30:45.125        │    928.8 MB │
└──────┴─────────────────────┴──────────────┴───────────────┴─────────────────────┴─────────────┘

🎯 Calculated Transmitter Location:
┌─────────────────────────┬─────────────────────────────────────────┐
│ Parameter               │ Value                                   │
├─────────────────────────┼─────────────────────────────────────────┤
│ Latitude                │      40.12789012°                       │
│ Longitude               │     -74.12789012°                       │
│ Altitude                │           125.50 m                      │
│ Estimated Accuracy      │            15.2 m                       │
│ Confidence Level        │           85.40%                        │
└─────────────────────────┴─────────────────────────────────────────┘
```

### Quality Assessment
- ✅ **HIGH CONFIDENCE**: Excellent signal conditions and geometry
- ⚠️ **MEDIUM CONFIDENCE**: Good conditions with some limitations  
- ❌ **LOW CONFIDENCE**: Poor signal or geometry conditions

### GDOP (Geometric Dilution of Precision)
- ✅ **EXCELLENT GEOMETRY**: GDOP ≤ 3.0
- ⚠️ **GOOD GEOMETRY**: GDOP ≤ 6.0
- ❌ **POOR GEOMETRY**: GDOP > 6.0 (consider repositioning stations)

## Algorithms

### Hyperbolic Positioning (Default)
- **Speed**: Fastest processing
- **Accuracy**: Good for most applications
- **Use Case**: Real-time analysis, initial positioning

### Least-Squares
- **Speed**: Moderate processing time
- **Accuracy**: Enhanced precision through optimization
- **Use Case**: Improved accuracy when geometry is adequate

### Newton-Raphson
- **Speed**: Slowest (iterative)
- **Accuracy**: Highest precision available
- **Use Case**: High-precision applications, research analysis

## Best Practices

### Station Deployment
1. **Geometry**: Deploy stations to form a triangle/polygon around expected transmitter area
2. **Separation**: Maintain minimum 100m separation between stations
3. **Baseline**: Longer baselines generally improve accuracy
4. **Elevation**: Consider terrain and line-of-sight factors

### Data Collection
1. **Synchronization**: Ensure all stations start collection within ±60 seconds
2. **Duration**: Collect sufficient data for cross-correlation analysis
3. **Frequency**: Use consistent frequency across all stations
4. **Signal Quality**: Maintain SNR > 10dB for reliable results

### Analysis Tips
1. **Algorithm Selection**: Start with hyperbolic, upgrade to least-squares or Newton-Raphson for higher precision
2. **Confidence Threshold**: Use 0.7+ for high-reliability applications
3. **Validation**: Cross-check results with known transmitter locations when possible
4. **Geometry Check**: Monitor GDOP values; reposition stations if GDOP > 6.0

## Troubleshooting

### Common Issues

**Error: "need at least 3 stations for TDOA analysis"**
- Solution: Provide minimum 3 data files as arguments

**Error: "station validation failed"**
- Check: All stations have valid GPS coordinates
- Check: All stations use the same frequency
- Check: Collection times are within reasonable range

**Low Confidence Results**
- Improve: Station geometry (wider separation)
- Improve: Signal quality (reduce noise, increase power)
- Try: Different algorithm (least-squares or newton-raphson)

**Poor GDOP (> 6.0)**
- Reposition: Stations for better geometric coverage
- Add: Additional stations if possible
- Consider: Terrain and obstruction effects

### Performance Optimization

```bash
# Reduce sample analysis for faster processing
./argus-dtoa *.dat --max-samples 100000

# Use fastest algorithm for real-time analysis
./argus-dtoa *.dat --algorithm hyperbolic

# High-precision analysis (slower)
./argus-dtoa *.dat --algorithm newton-raphson --max-samples 1000000
```

## File Formats

### Input Files
- **Format**: Argus Collector `.dat` files
- **Contents**: IQ sample data with GPS metadata
- **Requirements**: Valid GPS coordinates, consistent frequency

### Output Files
- **Default**: Human-readable text format
- **Export Mode**: Detailed analysis with intermediate data
- **Naming**: `tdoa_results_YYYYMMDD_HHMMSS.txt` (auto-generated)

## Technical Details

### TDOA Theory
Time Difference of Arrival (TDOA) positioning works by:
1. Measuring signal arrival times at multiple receivers
2. Calculating time differences relative to a reference station
3. Converting time differences to distance differences
4. Solving hyperbolic equations to find transmitter position

### Accuracy Factors
- **Timing Precision**: 1ns timing error ≈ 0.3m position error
- **Station Geometry**: Better geometry = lower GDOP = higher accuracy
- **Signal Quality**: Higher SNR = more precise time measurements
- **Multipath**: Reflections can introduce timing errors

### Coordinate Systems
- **WGS84**: World Geodetic System 1984 (GPS standard)
- **Decimal Degrees**: High-precision format for calculations
- **DMS**: Degrees, Minutes, Seconds for traditional navigation

## Integration

### With Argus Collector
```bash
# Collect data from multiple stations
./argus-collector --station-id Station-001 --output station1.dat
./argus-collector --station-id Station-002 --output station2.dat  
./argus-collector --station-id Station-003 --output station3.dat

# Analyze collected data
./argus-dtoa station1.dat station2.dat station3.dat
```

### With Argus Reader
```bash
# Verify data quality before TDOA analysis
./argus-reader station1.dat --analyze
./argus-reader station2.dat --analyze
./argus-reader station3.dat --analyze

# Perform TDOA analysis
./argus-dtoa station1.dat station2.dat station3.dat
```

## Examples

### Basic Transmitter Location
```bash
./argus-dtoa data/site1.dat data/site2.dat data/site3.dat
```

### High-Precision Analysis
```bash
./argus-dtoa data/*.dat \
    --algorithm newton-raphson \
    --confidence 0.8 \
    --verbose \
    --map \
    --output precise_location.txt
```

### Research Analysis with Full Export
```bash
./argus-dtoa experiment/*.dat \
    --algorithm least-squares \
    --export \
    --max-samples 1000000 \
    --verbose \
    --output research_results.txt
```

## Support

For technical support, bug reports, or feature requests:
- GitHub Issues: [Argus Collector Repository](https://github.com/ArgusSDR/argus-collector)
- Documentation: [Argus Collector Wiki](https://github.com/ArgusSDR/argus-collector/wiki)

## License

Part of the Argus Collector project. See main project LICENSE file for details.