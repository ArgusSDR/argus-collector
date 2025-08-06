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

1. **File Loading**: Reads and validates all input files
2. **Parameter Validation**: Ensures compatible frequency, sample rate, and timing
3. **Cross-Correlation**: Calculates time delays between all receiver pairs
4. **TDOA Calculation**: Converts time delays to distance differences
5. **Location Solving**: Uses hyperbolic positioning to find transmitter location
6. **Confidence Analysis**: Calculates error bounds and confidence metrics
7. **Output Generation**: Exports results in selected format

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

- Processing time scales with file size and number of receivers
- Large files (>100MB) may take several minutes to process
- Use --dry-run to verify file selection before processing
- Consider using subset of samples for initial testing

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