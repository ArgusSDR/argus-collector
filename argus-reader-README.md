# Argus Reader - Data File Analysis Tool

A utility for analyzing and displaying the contents of Argus Collector data files (`.dat` format). Argus Reader provides fast metadata inspection and detailed signal analysis capabilities.

## Overview

Argus Reader allows you to:
- **Instantly view file metadata** (frequency, GPS location, timestamps)
- **Analyze IQ sample data** (magnitude, phase, statistics)
- **Verify collection parameters** (sample rate, duration, device info)
- **Debug data collection issues** (GPS fix quality, timing accuracy)

## Installation

### Build from Source

```bash
# Build the reader utility
make build-reader

# Or build manually
go build -o argus-reader ./cmd/argus-reader
```

### Prerequisites

- Go 1.19 or later
- Access to Argus Collector data files (`.dat` format)

## Usage

### Basic Syntax

```bash
./argus-reader [options] <data-file.dat>
```

### Command Line Options

| Option | Short | Default | Description |
|--------|-------|---------|-------------|
| `--samples` | `-s` | `false` | Display IQ sample data |
| `--limit` | `-l` | `10` | Number of samples to display |
| `--stats` | | `false` | Show statistical analysis |
| `--format` | `-f` | `table` | Output format (table, json, csv) |
| `--help` | `-h` | | Show help information |

## Examples

### Quick File Inspection

```bash
# Fast metadata display (< 1ms)
./argus-reader data/argus_1234567890.dat
```

**Output:**
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
│ File Format Version     │ 1                                       │
│ Collection ID           │ argus_1234567890                        │
│ Device Info             │ RTL-SDR Device 0                        │
├─────────────────────────┼─────────────────────────────────────────┤
│ Frequency               │ 433.920 MHz                            │
│ Sample Rate             │ 2.048 MSps                             │
├─────────────────────────┼─────────────────────────────────────────┤
│ Collection Time         │ 2025-07-13 10:48:26.116               │
│ GPS Timestamp           │ 2025-07-13 10:49:26.063               │
├─────────────────────────┼─────────────────────────────────────────┤
│ GPS Latitude            │ 35.53319850°                          │
│ GPS Longitude           │ -97.62123717°                         │
│ GPS Altitude            │ 399.60 m                              │
└─────────────────────────┴─────────────────────────────────────────┘

📡 Sample Information:
┌─────────────────────────┬─────────────────────────────────────────┐
│ Parameter               │ Value                                   │
├─────────────────────────┼─────────────────────────────────────────┤
│ Total Samples           │ 122,421,248                            │
│ Sample Type             │ Complex64 (32-bit I + 32-bit Q)        │
│ Data Size               │ 934.00 MB                              │
│ Collection Duration     │ 59.776 seconds                         │
└─────────────────────────┴─────────────────────────────────────────┘
```

### Sample Data Analysis

```bash
# Display first 10 IQ samples
./argus-reader --samples data/argus_1234567890.dat

# Show more samples
./argus-reader --samples --limit 20 data/argus_1234567890.dat
```

**Sample Output:**
```
📈 IQ Sample Data (first 10 samples):
┌──────┬──────────────┬──────────────┬──────────────┬────────────┐
│ #    │ I (Real)     │ Q (Imag)     │ Magnitude    │ Phase (°)  │
├──────┼──────────────┼──────────────┼──────────────┼────────────┤
│ 0    │     0.125490 │    -0.133333 │     0.183012 │     -46.75 │
│ 1    │     0.086275 │    -0.094118 │     0.127714 │     -47.48 │
│ 2    │     0.109804 │    -0.125490 │     0.166667 │     -48.81 │
│ 3    │     0.094118 │    -0.117647 │     0.150980 │     -51.34 │
│ 4    │     0.078431 │    -0.101961 │     0.128676 │     -52.42 │
└──────┴──────────────┴──────────────┴──────────────┴────────────┘
```

### Statistical Analysis

```bash
# Comprehensive statistics
./argus-reader --stats data/argus_1234567890.dat

# Combined sample and statistical analysis
./argus-reader --samples --stats data/argus_1234567890.dat
```

**Statistics Output:**
```
📊 Statistical Analysis:
┌─────────────────────────┬─────────────────────────────────────────┐
│ Statistic               │ Value                                   │
├─────────────────────────┼─────────────────────────────────────────┤
│ Mean I (Real)           │     0.001234                           │
│ Mean Q (Imaginary)      │    -0.000987                           │
│ I Variance              │     0.125678                           │
│ Q Variance              │     0.134567                           │
├─────────────────────────┼─────────────────────────────────────────┤
│ Mean Magnitude          │     0.156789                           │
│ Min Magnitude           │     0.000123                           │
│ Max Magnitude           │     1.987654                           │
│ Mean Power              │     0.024567                           │
│ Power (dB)              │       -16.10 dB                        │
└─────────────────────────┴─────────────────────────────────────────┘
```

## Performance

### Speed Optimization

Argus Reader is optimized for different use cases:

| Operation | Speed | Memory Usage | Use Case |
|-----------|-------|--------------|----------|
| Metadata only | < 1ms | ~1KB | Quick file verification |
| Sample display | ~1s | ~100KB | Signal inspection |
| Full statistics | ~5-10s | ~1GB | Comprehensive analysis |

### File Size Handling

- **Small files** (< 100MB): All samples loaded for analysis
- **Large files** (> 100MB): Statistical sampling used for performance
- **Very large files** (> 1GB): Metadata-only mode recommended

## Use Cases

### 1. File Verification

Quickly verify collection parameters and GPS data:

```bash
./argus-reader data/*.dat | grep "GPS\|Frequency\|Duration"
```

### 2. Collection Quality Check

Verify GPS fix quality and timing accuracy:

```bash
# Check multiple files for GPS accuracy
for file in data/*.dat; do
    echo "=== $file ==="
    ./argus-reader "$file" | grep -E "(GPS|Collection Time)"
done
```

### 3. Signal Analysis

Analyze signal characteristics and noise levels:

```bash
# Get power measurements from multiple collections
./argus-reader --stats data/argus_*.dat | grep "Power (dB)"
```

### 4. TDOA Preparation

Verify synchronization across multiple stations:

```bash
# Check collection timing across stations
./argus-reader station1/argus_1234567890.dat | grep "Collection Time"
./argus-reader station2/argus_1234567890.dat | grep "Collection Time"
./argus-reader station3/argus_1234567890.dat | grep "Collection Time"
```

## File Format Information

### Supported Files

- **Extension**: `.dat` (Argus Collector binary format)
- **Magic Header**: `ARGUS` (5 bytes)
- **Endianness**: Little-endian
- **Sample Format**: Complex64 (32-bit float I + 32-bit float Q)

### Metadata Fields

| Field | Type | Description |
|-------|------|-------------|
| File Format Version | uint16 | Binary format version |
| Frequency | uint64 | RF frequency in Hz |
| Sample Rate | uint32 | Samples per second |
| Collection Time | timestamp | RTL-SDR start time (nanosecond precision) |
| GPS Timestamp | timestamp | GPS-synchronized time |
| GPS Location | lat/lon/alt | Collector position (float64) |
| Device Info | string | RTL-SDR device description |
| Collection ID | string | Unique collection identifier |
| Sample Count | uint32 | Number of IQ samples |

## Error Handling

### Common Issues

```bash
# File not found
./argus-reader nonexistent.dat
# Error: file does not exist: nonexistent.dat

# Invalid file format
./argus-reader textfile.txt
# Error: failed to read metadata: invalid file format

# Corrupted file
./argus-reader corrupted.dat
# Error: failed to read metadata: unexpected EOF
```

### Troubleshooting

1. **File Permission Issues**:
   ```bash
   chmod 644 data/*.dat
   ```

2. **Large File Performance**:
   ```bash
   # Use metadata-only for large files
   ./argus-reader large_file.dat
   
   # Avoid --stats on very large files
   ./argus-reader --samples --limit 5 large_file.dat
   ```

3. **Memory Issues**:
   ```bash
   # Monitor memory usage for large files
   time ./argus-reader --stats huge_file.dat
   ```

## Integration Examples

### Bash Scripts

```bash
#!/bin/bash
# Analyze all data files in a directory

for file in data/*.dat; do
    echo "=== Analyzing $file ==="
    
    # Quick metadata check
    ./argus-reader "$file" | grep -E "(Frequency|Duration|GPS)"
    
    # Signal quality check
    power=$(./argus-reader --stats "$file" | grep "Power (dB)" | awk '{print $5}')
    echo "Signal Power: $power"
    
    echo ""
done
```

### Python Integration

```python
#!/usr/bin/env python3
import subprocess
import json
import glob

def analyze_argus_files(directory):
    """Analyze all Argus files in directory"""
    
    for filepath in glob.glob(f"{directory}/*.dat"):
        print(f"Analyzing {filepath}")
        
        # Run argus-reader and capture output
        result = subprocess.run(
            ["./argus-reader", filepath],
            capture_output=True,
            text=True
        )
        
        if result.returncode == 0:
            # Parse output for specific fields
            lines = result.stdout.split('\n')
            for line in lines:
                if 'Frequency' in line or 'GPS' in line:
                    print(f"  {line.strip()}")
        else:
            print(f"  Error: {result.stderr}")

if __name__ == "__main__":
    analyze_argus_files("data")
```

## Advanced Usage

### Batch Processing

```bash
# Generate summary report for all collections
echo "Collection Summary Report" > report.txt
echo "=========================" >> report.txt

for file in data/*.dat; do
    echo "" >> report.txt
    echo "File: $(basename $file)" >> report.txt
    ./argus-reader "$file" | grep -E "(Frequency|Collection Duration|GPS)" >> report.txt
done
```

### Data Validation

```bash
# Validate GPS coordinates are within expected region
./argus-reader data/*.dat | grep "GPS" | while read line; do
    if echo "$line" | grep -q "35\.[0-9]*°.*-97\.[0-9]*°"; then
        echo "✓ GPS coordinates valid: $line"
    else
        echo "✗ GPS coordinates out of range: $line"
    fi
done
```

## Contributing

To contribute to Argus Reader:

1. Fork the repository
2. Add new features to `cmd/argus-reader/main.go`
3. Update this README with new functionality
4. Submit a pull request

## See Also

- **Argus Collector**: Main collection program
- **PLAN.md**: Project architecture and goals
- **README.md**: Complete system documentation

## License

Same license as Argus Collector project.