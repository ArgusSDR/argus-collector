# Argus Processor - TDOA Signal Processing Tool

The `argus-processor` tool analyzes multiple synchronized argus data files to calculate transmitter locations using Time Difference of Arrival (TDOA) algorithms.

## Features

- **Multi-receiver TDOA processing**: Processes 3+ synchronized data files from different receiver locations
- **Cross-correlation analysis**: Finds time delays between signal arrivals using FFT-based correlation
- **Hyperbolic positioning**: Calculates transmitter location with error bounds and confidence intervals
- **Multiple output formats**: Exports results in GeoJSON, KML, and CSV formats for mapping and analysis
- **Configurable algorithms**: Supports basic, weighted, and Kalman filter approaches
- **Probability heatmaps**: Generates confidence area visualizations

## Requirements

- Minimum 3 synchronized argus data files from different receiver locations
- Files must have same frequency and sample rate
- GPS coordinates must be recorded for each receiver
- Collection times should be synchronized (within 1 second)

## Usage

### Basic Usage

```bash
# Process files matching pattern (include path!) - generates KML by default
./argus-processor --input "data/argus-?_*.dat"

# Process with higher confidence threshold
./argus-processor --input "/path/to/station*.dat" --confidence 0.8 --verbose

# Generate GeoJSON for web mapping
./argus-processor --input "data/argus*.dat" --output-format geojson --output ./results
```

### Command Line Options

- `--input`, `-i`: Input file pattern (e.g., "argus-?_*.dat") [REQUIRED]
- `--output-format`, `-f`: Output format (geojson, kml, csv) [default: kml]
- `--output`, `-o`: Output directory [default: ./tdoa-results]
- `--algorithm`, `-a`: TDOA algorithm (basic, weighted, kalman) [default: basic]
- `--confidence`, `-c`: Minimum confidence threshold (0.0-1.0) [default: 0.5]
- `--max-distance`, `-d`: Maximum expected transmitter distance (km) [default: 50]
- `--frequency-range`: Frequency range to analyze (e.g., '433.9-434.0')
- `--verbose`, `-v`: Enable verbose logging
- `--dry-run`: Show what would be processed without doing it
- `--version`: Show version information

### File Naming Patterns

The tool supports flexible file patterns (must include path):
- `data/argus-?_*.dat` - Matches data/argus-0_timestamp.dat, data/argus-1_timestamp.dat, etc.
- `/path/to/station*.dat` - Matches any file starting with "station" in specified path
- `./data/2025*.dat` - Matches files in data directory starting with "2025"

**Important**: Always include the directory path in your pattern. Patterns like `argus-*.dat` will only search the current working directory.

## Output Formats

### GeoJSON Format
- Compatible with web mapping libraries (Leaflet, Mapbox, OpenLayers)
- Contains transmitter location, confidence area, receiver positions, and TDOA baselines
- Includes probability heatmap points for visualization

### KML Format  
- Compatible with Google Earth and other KML viewers
- Shows transmitter location with styled markers
- Displays confidence circle and receiver stations
- Includes TDOA measurement lines between receivers

### CSV Format
- Suitable for spreadsheet analysis and custom plotting
- Contains receiver information, TDOA measurements, and heatmap data
- Header comments include processing metadata

## Output Filename Format

Files are named automatically based on processing parameters:
```
tdoa_YYYYMMDD_HHMMSS_FrequencyHz_heatmap.extension
```

Example: `tdoa_20250801_143022_433920000Hz_heatmap.geojson`

## Algorithms

### Basic Algorithm
- Simple cross-correlation with least-squares positioning
- Fast processing, suitable for strong signals
- Good for initial location estimates

### Weighted Algorithm (Future)
- Weights measurements by signal strength and confidence
- Better handling of varying signal quality
- Improved accuracy with mixed signal conditions

### Kalman Algorithm (Future)
- Incorporates multiple time samples for tracking
- Provides smooth location estimates over time
- Best for continuous monitoring applications

## Processing Steps

1. **File Loading**: Reads and validates all input files using optimized I/O
2. **Parameter Validation**: Ensures compatible frequency, sample rate, and timing
3. **Multi-Resolution Cross-Correlation**: 
   - **Coarse Search**: Fast correlation with 8x decimated samples
   - **Medium Search**: Refined correlation with 2x decimated samples  
   - **Fine Search**: Precise correlation at full resolution
4. **TDOA Calculation**: Converts time delays to distance differences
5. **Location Solving**: Uses hyperbolic positioning to find transmitter location
6. **Confidence Analysis**: Calculates error bounds and confidence metrics
7. **Output Generation**: Exports results in selected format

### Multi-Resolution Correlation Details

The processor uses a three-stage correlation approach for optimal speed:

1. **Stage 1 - Coarse (8x decimation)**: 
   - Uses every 8th sample for rapid initial search
   - Searches up to 100 delay positions with large steps
   - Provides rough delay estimate with ~95% speed improvement

2. **Stage 2 - Medium (2x decimation)**:
   - Uses every 2nd sample around coarse result
   - Searches ±32 samples around coarse estimate
   - Refines delay estimate with good speed/accuracy balance

3. **Stage 3 - Fine (full resolution)**:
   - Uses all samples around medium result  
   - Searches ±8 samples for final precision
   - Provides sample-accurate delay measurement

## Example Workflow

```bash
# 1. Collect synchronized data from multiple stations
./argus-collector --duration 10s --output station1.dat  # Station 1
./argus-collector --duration 10s --output station2.dat  # Station 2  
./argus-collector --duration 10s --output station3.dat  # Station 3

# 2. Process for transmitter location
./argus-processor --input "station*.dat" --verbose

# 3. View results in Google Earth
# Results in ./tdoa-results/tdoa_YYYYMMDD_HHMMSS_433920000Hz_heatmap.kml
```

## Troubleshooting

### "Need at least 3 files" Error
- Ensure file pattern matches at least 3 data files
- Check that files have .dat extension
- Verify files exist in specified location

### "Frequency mismatch" Error  
- All files must be collected at the same frequency
- Check frequency settings in argus-collector configuration

### "Time sync issue" Error
- Collection times must be within 1 second of each other
- Use synchronized start mode in argus-collector
- Ensure GPS time synchronization

### Low Confidence Measurements
- Processor now generates output files even with low confidence measurements
- Low confidence measurements are included in output files but marked
- For better accuracy: increase receiver spacing, improve signal quality, or ensure proper time synchronization
- Adjust threshold with --confidence flag (default: 0.5)

### Poor Location Accuracy
- Increase receiver spacing for better geometry
- Use higher sample rates for better time resolution
- Ensure strong signal-to-noise ratio at all receivers
- Consider using weighted or Kalman algorithms

## Performance Considerations

### File I/O Optimization

The processor automatically selects the best I/O strategy based on file size:

- **Small files (<5MB)**: Standard file reading for compatibility
- **Medium files (5-50MB)**: Optimized buffered I/O with 64KB chunks
- **Large files (>50MB)**: Memory mapping for maximum performance

### Performance Characteristics

- **Memory mapping**: 5-10x faster than standard I/O for large files
- **Multi-resolution search**: 2-5x faster correlation with minimal accuracy loss
- **Unsafe operations**: Direct byte-to-sample conversion for speed
- **Comprehensive progress reporting**: Real-time feedback with step-by-step progress and time estimates
- **Automatic cleanup**: Proper resource management with deferred cleanup

### Processing Time Estimates

- Processing time scales with file size and number of receivers
- Memory-mapped files load much faster than buffered reading
- **Multi-resolution correlation**: 2-5x faster than single-resolution search
- Cross-correlation optimized but still the main computational bottleneck
- Use --dry-run to verify file selection before processing
- Larger sample windows (50k vs 10k samples) improve accuracy with optimized search

### Memory Usage

- **Memory mapping**: Virtual memory usage equals file size but actual RAM usage is minimal
- **Sample storage**: Full sample arrays loaded into RAM for correlation
- **Large datasets**: Monitor available RAM when processing many large files

### Performance Comparison

**Traditional Single-Resolution Search:**
- For 50,000 samples with ±5,000 sample search range
- Performs 10,001 full correlations
- Processing time: ~30-60 seconds per receiver pair

**Multi-Resolution Search:**
- **Stage 1**: ~100 correlations on 6,250 decimated samples (8x)
- **Stage 2**: ~65 correlations on 25,000 decimated samples (2x) 
- **Stage 3**: ~17 correlations on full 50,000 samples
- **Total**: ~182 correlations vs 10,001
- Processing time: ~6-12 seconds per receiver pair (**2-5x speedup**)

The multi-resolution approach maintains correlation accuracy while dramatically reducing computation time by focusing expensive full-resolution correlation only around the most promising delay candidates.

### Progress Reporting System

The processor provides comprehensive progress reporting for long-running operations:

**Step-by-Step Processing:**
1. **Step 1**: Loading and validating data files
2. **Step 2**: Performing cross-correlation analysis  
3. **Step 3**: Calculating transmitter location
4. **Step 4**: Generating probability heatmap (if enabled)

**Progress Information:**
- **Current step**: Shows which processing stage is active
- **Sub-progress**: Detailed progress within each step
- **Overall progress**: Total completion percentage across all steps
- **Elapsed time**: Time since processing started
- **Detailed status**: Specific information about current operation

**Example Progress Output:**
```
⏳ Step 1/4: Loading and validating data files (elapsed: 2s)
   📊 66.7% complete (16.7% overall, file 2/3) - 5s elapsed
✅ Step 1/4 complete: Loading and validating data files (25.0% overall, 7s elapsed)

⏳ Step 2/4: Performing cross-correlation analysis (elapsed: 7s)
   📊 33.3% complete (33.3% overall, pair 1/3 (R1↔R2)) - 15s elapsed
✅ Step 2/4 complete: Performing cross-correlation analysis (50.0% overall, 45s elapsed)
```

**Progress Reporting Modes:**
- **Normal mode**: Progress updates every 2 seconds
- **Verbose mode**: Progress updates every 500ms with additional details
- **Dry run**: No progress reporting, just shows what would be processed

## Integration with Mapping Software

### Web Mapping (GeoJSON)
```javascript
// Load in Leaflet.js
fetch('tdoa_results.geojson')
  .then(response => response.json())
  .then(data => {
    L.geoJSON(data).addTo(map);
  });
```

### Google Earth (KML)
1. Open Google Earth
2. File → Open → Select KML file
3. Transmitter location and confidence area will be displayed

### QGIS/ArcGIS (GeoJSON/CSV)
1. Add Vector Layer → Select GeoJSON file
2. Or import CSV with lat/lon columns
3. Style points and polygons as needed

## Related Tools

- `argus-collector`: Collects synchronized RF data files
- `argus-reader`: Analyzes individual data files
- Build with `make build-processor` or `make build-all-tools`