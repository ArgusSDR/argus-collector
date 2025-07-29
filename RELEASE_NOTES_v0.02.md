# Argus Collector v0.02 Release Notes

**Release Date**: July 29, 2025  
**Branch**: controls → main  
**Previous Version**: 0.1.0  

## 🚀 Major Features

### Advanced RTL-SDR Gain Control
- **Manual Gain Control**: Precise gain setting in dB for consistent receiver behavior
- **Automatic Gain Control (AGC)**: Dynamic gain adjustment for varying signal conditions
- **Gain Mode Selection**: Command-line and configuration file support for gain control modes
- **Optimal Gain Guidelines**: Built-in recommendations for different signal environments

### Device Identification in Filenames
- **Smart Filename Generation**: Files now include device ID for multi-station deployments
- **Format**: `{prefix}-{device_id}_{timestamp}.dat` (e.g., `argus-NORTH001_1753824525.dat`)
- **Device Selection**: Support for both device index and serial number identification
- **TDOA-Ready**: Perfect for multi-station synchronized collections

### Bias Tee Support
- **External LNA Power**: Built-in bias tee control for powering Low Noise Amplifiers
- **Hardware Compatibility**: Automatic detection and control where supported
- **Safety Features**: Proper current limiting and device compatibility checking

### Enhanced Analysis Tools
- **Device Configuration Analysis**: Comprehensive RTL-SDR settings analysis with recommendations
- **Gain Reference Tables**: Built-in RTL-SDR gain guidelines for optimal performance
- **Multi-Station Validation**: Tools for verifying TDOA setup consistency
- **Enhanced argus-reader**: Extended capabilities for device configuration inspection

## 🛠️ New Command Line Options

### Argus Collector
```bash
# Gain Control
-g, --gain float         Manual gain setting in dB (default 20.7)
    --gain-mode string   Gain control mode: auto (AGC) or manual (default "manual")
    --bias-tee          Enable bias tee for powering external LNAs

# Enhanced Device Selection
-D, --device string     RTL-SDR device selection (serial number or index)
```

### Argus Reader
```bash
# Device Analysis
    --device-analysis   Show detailed device configuration analysis
    --hex              Display raw hexadecimal dump of sample data
    --hex-limit int    Limit bytes in hex dump (default 256)
```

## 📋 Usage Examples

### Basic Gain Control
```bash
# Manual gain for consistent TDOA collection
./argus-collector --gain-mode manual --gain 25.0 --gps-mode manual --latitude 35.533 --longitude -97.621

# Automatic gain for variable conditions  
./argus-collector --gain-mode auto --frequency 868e6 --gps-mode manual --latitude 35.533 --longitude -97.621

# Enable bias tee for external LNA
./argus-collector --bias-tee --gain-mode manual --gain 15.0 --frequency 433.92e6
```

### Multi-Station TDOA Setup
```bash
# Station 1 with consistent gain
./argus-collector -D NORTH001 --gain-mode manual --gain 25.0 --frequency 433.92e6 --output station1/
# Creates: station1/argus-NORTH001_timestamp.dat

# Station 2 with same settings  
./argus-collector -D SOUTH001 --gain-mode manual --gain 25.0 --frequency 433.92e6 --output station2/
# Creates: station2/argus-SOUTH001_timestamp.dat

# Station 3 with same settings
./argus-collector -D EAST0001 --gain-mode manual --gain 25.0 --frequency 433.92e6 --output station3/
# Creates: station3/argus-EAST0001_timestamp.dat
```

### Device Configuration Analysis
```bash
# Analyze device settings and get recommendations
./argus-reader --device-analysis data/argus-NORTH001_*.dat

# Verify gain consistency across stations
for station in NORTH001 SOUTH001 EAST0001; do
    ./argus-reader --device-analysis data/argus-${station}_*.dat | grep "Gain"
done
```

## 🔧 Configuration File Updates

### New RTLSDR Settings
```yaml
rtlsdr:
  frequency: 433.92e6      # Target frequency in Hz
  sample_rate: 2048000     # Sample rate in Hz
  gain: 20.7               # RF gain in dB (used when gain_mode is "manual")
  gain_mode: "manual"      # Gain mode: "auto" (AGC) or "manual"
  device_index:            # RTL-SDR device index (used if serial_number is empty)
  serial_number: ""        # RTL-SDR device serial number (preferred over device_index)  
  bias_tee: false          # Enable bias tee for powering external LNAs
```

## 📊 Enhanced Output Features

### Improved Device Information Display
- **Detailed Device Info**: Comprehensive RTL-SDR configuration display
- **Gain Settings**: Clear indication of gain mode and value
- **Bias Tee Status**: Hardware power supply status indication
- **Device Identification**: Clear device naming and identification

### Example Device Info Output
```
Device: RTL-SDR Stub Device (freq: 433920000 Hz, rate: 2048000 Hz, gain: 25.0 dB (manual), bias-tee: off)
```

## 🚨 Breaking Changes

### Filename Format Change
- **Old Format**: `argus_timestamp.dat`
- **New Format**: `argus-deviceID_timestamp.dat`
- **Impact**: Scripts parsing filenames may need updates
- **Benefit**: Clear device identification for multi-station deployments

### Device Selection Logic Enhancement
- **Improved Serial Number Detection**: Better handling of numeric serial numbers with leading zeros
- **Smart Device ID Resolution**: Automatic differentiation between device indices and serial numbers
- **Command Line Priority**: `-D` flag now properly overrides configuration file settings

## 🐛 Bug Fixes

### Device Selection
- **Fixed**: Device selection logic now properly handles serial numbers like "00000001"
- **Fixed**: Command line device selection properly overrides configuration file settings
- **Fixed**: Device information now includes actual RTL-SDR settings instead of hardcoded values

### Configuration Handling
- **Fixed**: Gain mode configuration now properly persists in data files
- **Fixed**: Viper configuration binding for RTL-SDR settings improved
- **Fixed**: Device identifier logic handles edge cases correctly

## 📈 Performance Improvements

### Argus Reader Enhancements
- **Smart Metadata Parsing**: Enhanced device information extraction from stored data
- **Improved Error Handling**: Better handling of truncated or corrupted data files
- **Memory Optimization**: Efficient processing of large data files
- **Faster Analysis**: Optimized device configuration parsing

## 🎯 TDOA-Specific Improvements

### Multi-Station Consistency
- **Gain Verification**: Tools for ensuring consistent gain settings across all stations
- **Device Identification**: Clear station identification through filename device IDs
- **Configuration Validation**: Built-in recommendations for optimal TDOA performance
- **Synchronized Settings**: Easy verification of identical configurations across stations

### Quality Assurance
- **Real-time Validation**: Device configuration displayed during collection
- **Post-Collection Analysis**: Comprehensive device settings verification
- **Troubleshooting Tools**: Built-in gain reference and recommendations
- **Batch Processing**: Multi-station configuration comparison tools

## 📚 Documentation Updates

### Comprehensive README Updates
- **New Gain Control Section**: Complete guide to manual and automatic gain control
- **Device ID Documentation**: Full explanation of new filename format and benefits
- **Enhanced Examples**: Updated all usage examples with gain control
- **TDOA Guidelines**: Specific recommendations for multi-station deployments

### Enhanced Argus Reader Documentation
- **Device Analysis Guide**: Complete documentation of configuration analysis features
- **Updated Examples**: All examples reflect new device ID filename format
- **New Use Cases**: Multi-station gain verification and quality assurance workflows

## 🔗 Migration Guide

### For Existing Users
1. **Filename Scripts**: Update any scripts that parse data filenames to handle the new `prefix-deviceID_timestamp.dat` format
2. **Configuration Files**: Add `gain_mode` and `bias_tee` settings to existing config.yaml files
3. **Multi-Device Setups**: Consider setting unique serial numbers for RTL-SDR devices using `rtl_eeprom`

### Recommended Actions
1. **Set Unique Serial Numbers**: Use `rtl_eeprom` to assign descriptive serial numbers to RTL-SDR devices
2. **Standardize Gain Settings**: Establish consistent gain settings across all TDOA stations
3. **Update Analysis Scripts**: Incorporate new device analysis features into data processing workflows

## 🔮 Future Compatibility

This release maintains backward compatibility for:
- **Configuration Files**: Existing config.yaml files continue to work with new defaults
- **Data File Format**: No changes to binary data format - all existing files remain readable
- **Command Line Interface**: All existing command line options continue to work as expected

## 🙏 Acknowledgments

- Enhanced RTL-SDR integration for professional TDOA applications
- Improved multi-station deployment support
- Community feedback on gain control requirements
- Professional signal intelligence workflow optimization

---

**Full Changelog**: https://github.com/username/argus-collector/compare/v0.1.0...v0.02  
**Download**: https://github.com/username/argus-collector/releases/tag/v0.02