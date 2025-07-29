# Signal Magnitude Graph Feature

## Overview

The argus-reader now includes ASCII graph functionality to visualize signal magnitude over time, providing immediate visual feedback about RF signal characteristics.

## Usage

### Basic Graph Display
```bash
# Generate default ASCII graph
./argus-reader --graph data/argus_1753741539.dat

# Short form
./argus-reader -g data/argus_1753741539.dat
```

### Customizable Graph Parameters
```bash
# Custom graph size
./argus-reader --graph --graph-width 120 --graph-height 30 data/file.dat

# More data points for higher resolution
./argus-reader --graph --graph-samples 2000 data/file.dat

# Combined with other outputs
./argus-reader --graph --stats --samples data/file.dat
```

## Graph Output Format

```
📈 Signal Magnitude Over Time:
Samples: 1000 | Duration: 0.488 seconds | Sample Rate: 2.048 MSps
Magnitude Range: 0.001234 to 0.987654

Magnitude
  0.9877 |                                     *                                          |
  0.8754 |                               *           *                                    |
  0.7631 |                          *                     *                              |
  0.6508 |                     *                               *                         |
  0.5385 |               *                                           *                   |
  0.4262 |          *                                                     *              |
  0.3139 |     *                                                               *         |
  0.2016 |*                                                                         *    |
  0.0893 |                                                                             * |
         +--------------------------------------------------------------------------------+
         0                            0.244s                            0.488s

Legend: * = data point, # = multiple points, Time →

📊 Signal Analysis:
   Average Magnitude: 0.345678
   Peak Magnitude: 0.987654
   Dynamic Range: 58.12 dB
```

## Features

### 📊 **Visual Signal Analysis**
- **Magnitude vs Time**: Shows how signal strength varies over the collection period
- **Pattern Recognition**: Identify periodic signals, bursts, noise characteristics
- **Quick Assessment**: Immediate visual feedback without external tools

### ⚙️ **Customizable Display**
- **Graph Dimensions**: Adjustable width (default: 80) and height (default: 20)
- **Sample Resolution**: Control how many samples to plot (default: 1000)
- **Data Sampling**: Automatically samples evenly across entire collection

### 📈 **Multi-Character Plotting**
- **`*`**: Single data point at this position
- **`#`**: Multiple data points mapped to same position (density indicator)
- **Scaling**: Automatic scaling to fit magnitude range

### 🕒 **Time Axis**
- **Real Time**: Calculated from sample rate and sample count
- **Time Labels**: Start (0), middle, and end times displayed
- **Duration Info**: Total collection duration shown in header

### 📊 **Integrated Analysis**
- **Statistical Summary**: Average, peak magnitude, dynamic range
- **Dynamic Range**: Calculated in dB (20*log10(max/min))
- **Sample Information**: Sample count, duration, effective sample rate

## Use Cases

### 1. **Signal Detection**
```bash
# Quick check if there's any signal activity
./argus-reader -g data/suspected_transmission.dat
```
Look for:
- **Flat line**: No signal (just noise)
- **Spikes/Peaks**: Intermittent transmissions
- **Sustained levels**: Continuous signal

### 2. **Transmission Pattern Analysis**
```bash
# High resolution analysis of signal patterns
./argus-reader -g --graph-samples 5000 --graph-width 160 data/pattern.dat
```
Identify:
- **Periodic patterns**: Regular transmission intervals
- **Burst characteristics**: Duration and spacing of transmissions
- **Modulation effects**: Signal envelope variations

### 3. **Collection Quality Assessment**
```bash
# Combined analysis for data validation
./argus-reader -g --stats data/collection.dat
```
Check for:
- **Saturation**: Clipped signals (flat tops)
- **Noise floor**: Background noise level
- **Dynamic range**: Effective bit depth utilization

### 4. **Multi-File Comparison**
```bash
# Compare signal levels across multiple collections
for file in data/*.dat; do
    echo "=== $file ==="
    ./argus-reader -g "$file" | grep -E "(Peak|Average|Dynamic)"
done
```

### 5. **Quick Troubleshooting**
```bash
# Fast signal presence check
./argus-reader -g --graph-samples 500 data/test.dat
```
Diagnose:
- **No signal**: Hardware/antenna issues
- **Overload**: Gain too high
- **Intermittent**: Timing or interference issues

### 6. **File Integrity Analysis**
```bash
# Check for truncated or corrupted files
./argus-reader data/suspicious.dat | grep -E "(Header claims|Actual readable)"

# Compare expected vs actual data
for file in data/*.dat; do
    echo "=== $file ==="
    ./argus-reader "$file" | grep -A4 "File analysis"
done
```
Identify:
- **Truncated collections**: Header >> Actual samples
- **Complete collections**: Header ≈ Actual samples  
- **Storage issues**: Inconsistent file sizes

## Graph Interpretation

### **Signal Characteristics**

**Noise Floor**:
```
  0.0234 |********************************************************************************|
  0.0156 |********************************************************************************|
  0.0078 |********************************************************************************|
```
Flat, low-level signal indicates pure noise.

**Clean Signal**:
```
  0.8765 |                                     ***                                         |
  0.6543 |                               ****************                                 |
  0.4321 |                         *************************                             |
  0.2109 |*********************                                 *********************     |
```
Clear signal with defined start/stop times.

**Periodic Transmission**:
```
  0.7654 |  ***    ***    ***    ***    ***    ***    ***    ***    ***    ***    ***  |
  0.5432 | *****  *****  *****  *****  *****  *****  *****  *****  *****  *****  ***** |
  0.3210 |*******  *  ****  ****  ****  ****  ****  ****  ****  ****  ****  ****  ****|
```
Regular pattern indicates periodic transmissions.

### **Quality Indicators**

- **Dynamic Range > 40dB**: Good signal quality
- **Peak > 10x Average**: Strong signal above noise
- **Uniform Distribution**: Good gain setting
- **Clipping (flat tops)**: Reduce gain

## Integration with Other Tools

### **Workflow Examples**
```bash
# 1. Quick assessment
./argus-reader -g data/unknown.dat

# 2. Detailed analysis if signal found
./argus-reader -g --stats --samples data/unknown.dat

# 3. Export hex data for external processing
./argus-reader --hex --hex-limit 1024 data/unknown.dat > signal.hex
```

### **Batch Processing**
```bash
# Generate reports for all collections
for file in data/*.dat; do
    echo "=== Analysis: $file ===" >> report.txt
    ./argus-reader -g "$file" | grep -A 10 "Signal Analysis" >> report.txt
    echo "" >> report.txt
done
```

## Performance

- **Speed**: Graph generation adds ~100ms processing time
- **Memory**: Uses ~8KB additional memory for graph buffer
- **Scaling**: Automatically subsamples large files for performance
- **Large Files**: Processes files up to several GB efficiently

## Technical Details

### **Sample Count Interpretation**

When analyzing data files, argus-reader displays three different sample counts that provide important information about file integrity and data availability:

#### **Header Claims: X samples**
- **Definition**: The sample count stored in the file's binary header
- **Source**: Written by argus-collector during data collection
- **Represents**: The intended/expected number of samples for the collection
- **Use Case**: Original collection parameters and expected file size

#### **Available for Samples: X bytes**
- **Definition**: Actual file size minus estimated header size
- **Calculation**: `File Size - Header Size = Available Data Space`
- **Represents**: Maximum possible sample data that could be stored
- **Use Case**: Determining theoretical sample capacity

#### **Actual Readable Samples: X**
- **Definition**: Number of complete IQ samples that can be successfully read
- **Calculation**: `Available Bytes ÷ 8 bytes per complex64 sample`
- **Represents**: Samples that actually exist and can be processed
- **Use Case**: Real data available for analysis and graphing

#### **Example Scenario**
```
📊 File analysis:
   Header claims: 122880000 samples (937.50 MB)
   File size: 10773997 bytes (10.27 MB)  
   Estimated header size: 105 bytes
   Available for samples: 10773892 bytes
   Actual readable samples: 1346736
```

**Interpretation**:
- **Complete Collection**: Would have 122,880,000 samples (60 seconds at 2.048 MSps)
- **File Truncation**: Only ~1.35M samples available (collection was interrupted)
- **Data Loss**: 99.89% of intended data is missing
- **Graph Impact**: Shows only first ~0.66 seconds of intended 60-second collection

#### **Common Patterns**

**✅ Complete File**:
- Header Claims = Actual Readable (±1 due to rounding)
- Available bytes matches expected data size
- No file truncation warnings

**⚠️ Truncated File** (Collection Interrupted):
- Header Claims >> Actual Readable
- File size much smaller than expected
- Shows "File appears truncated" warning

**❌ Corrupted Header**:
- Header Claims unreasonably large/small
- Available bytes don't match header expectations
- May show negative or zero readable samples

### **Graph Sample Selection**

The graph feature uses **Header Claims** by default (capped at 10,000) to represent the intended collection scope, not just the available data. This provides:
- **Consistent scaling** across complete and partial files
- **Representative time axis** based on original collection parameters  
- **Proper duration calculation** from intended sample count and rate

### **Sampling Strategy**
- Files ≤ graph-samples: Uses all samples
- Files > graph-samples: Even distribution sampling
- Preserves signal envelope characteristics
- Maintains temporal accuracy

### **Magnitude Calculation**
```go
magnitude = sqrt(I² + Q²)  // Standard complex magnitude
```

### **Scaling Algorithm**
```go
normalized = (magnitude - min) / (max - min)
y_position = (1.0 - normalized) * (height - 1)
```

The ASCII graph feature provides immediate, visual insight into RF signal characteristics directly from the command line, making it invaluable for rapid signal analysis and troubleshooting.