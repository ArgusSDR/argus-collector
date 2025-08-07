// Argus Collector - RTL-SDR signal collection tool for TDOA analysis
// This program captures radio frequency signals using RTL-SDR hardware
// and GPS positioning data for Time Difference of Arrival analysis.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"argus-collector/internal/collector"
	"argus-collector/internal/config"
	"argus-collector/internal/rtlsdr"
	"argus-collector/internal/version"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Command line flag variables
var (
	cfgFile         string  // Configuration file path
	frequency       float64 // RF frequency to monitor in Hz
	duration        string  // Collection duration (e.g., "60s")
	output          string  // Output directory for data files
	gpsMode         string  // GPS mode: nmea, gpsd, or manual
	gpsPort         string  // GPS device serial port (for NMEA mode)
	gpsdHost        string  // GPSD host address (for gpsd mode)
	gpsdPort        string  // GPSD port (for gpsd mode)
	verbose         bool    // Enable verbose logging
	syncedStart     bool    // Enable synchronized start timing
	startTime       int64   // Exact epoch timestamp for collection start
	latitude        float64 // Manual latitude in decimal degrees
	longitude       float64 // Manual longitude in decimal degrees
	altitude        float64 // Manual altitude in meters
	device          string  // RTL-SDR device selection (serial number or index)
	gain            float64 // Manual gain setting in dB
	gainMode        string  // Gain mode: auto or manual
	biasTeeFlag     bool    // Enable bias tee for external LNA power
	showVersion     bool    // Show version information
	sampleRate      uint32  // Sample rate in Hz
	freqCorrection  int     // Frequency correction in PPM
	collectionID    string  // Collection identifier for filename
	filePrefix      string  // Prefix for output filenames
	gpsBaudRate     int     // GPS serial port baud rate
	gpsTimeout      string  // GPS fix timeout duration
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "argus-collector",
	Short: "RTL-SDR signal collection tool for TDOA analysis",
	Long: `Argus Collector captures radio frequency signals using RTL-SDR hardware
and GPS positioning data for Time Difference of Arrival (TDOA) analysis.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle version flag
		if showVersion {
			fmt.Println(version.GetVersionInfo("Argus Collector"))
			return
		}

		if err := runCollector(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// devicesCmd represents the devices command to list available RTL-SDR devices
var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List available RTL-SDR devices",
	Long: `List all available RTL-SDR devices with their index, name, manufacturer,
product, and serial number information. Use this to identify devices for
configuration with serial numbers.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listDevices(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// init initializes the CLI flags and configuration
func init() {
	// Initialize configuration when cobra starts
	cobra.OnInitialize(initConfig)

	// Persistent flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./config.yaml", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&showVersion, "version", false, "show version information")

	// Command-specific flags
	rootCmd.Flags().Float64VarP(&frequency, "frequency", "f", 433.92e6, "frequency to monitor (Hz)")
	rootCmd.Flags().StringVarP(&duration, "duration", "d", "60s", "collection duration")
	rootCmd.Flags().StringVarP(&output, "output", "o", "./data", "output directory")
	rootCmd.Flags().BoolVar(&syncedStart, "synced-start", true, "enable delayed/synchronized start time (true|false)")
	rootCmd.Flags().Int64Var(&startTime, "start-time", 0, "exact epoch timestamp for collection start (overrides synced-start)")

	// GPS configuration options
	rootCmd.Flags().StringVar(&gpsMode, "gps-mode", "nmea", "GPS mode: nmea, gpsd, or manual")
	rootCmd.Flags().StringVarP(&gpsPort, "gps-port", "p", "/dev/ttyUSB0", "GPS serial port (for NMEA mode)")
	rootCmd.Flags().StringVar(&gpsdHost, "gpsd-host", "localhost", "GPSD host address (for gpsd mode)")
	rootCmd.Flags().StringVar(&gpsdPort, "gpsd-port", "2947", "GPSD port (for gpsd mode)")

	// Manual GPS coordinates (for manual mode)
	rootCmd.Flags().Float64Var(&latitude, "latitude", 0.0, "manual latitude in decimal degrees (for manual mode)")
	rootCmd.Flags().Float64Var(&longitude, "longitude", 0.0, "manual longitude in decimal degrees (for manual mode)")
	rootCmd.Flags().Float64Var(&altitude, "altitude", 0.0, "manual altitude in meters (for manual mode)")

	// RTL-SDR device selection and gain control
	rootCmd.Flags().StringVarP(&device, "device", "D", "", "RTL-SDR device selection (serial number or index)")
	rootCmd.Flags().Float64VarP(&gain, "gain", "g", 10.0, "manual gain setting in dB (used when gain-mode is manual)")
	rootCmd.Flags().StringVar(&gainMode, "gain-mode", "manual", "gain control mode: auto (AGC) or manual")
	rootCmd.Flags().BoolVar(&biasTeeFlag, "bias-tee", false, "enable bias tee for powering external LNAs")
	
	// Add missing flags for complete configuration coverage
	rootCmd.Flags().Uint32Var(&sampleRate, "sample-rate", 0, "sample rate in Hz")
	rootCmd.Flags().IntVar(&freqCorrection, "frequency-correction", 0, "frequency correction in PPM")
	rootCmd.Flags().StringVar(&collectionID, "collection-id", "", "collection identifier for filename")
	rootCmd.Flags().StringVar(&filePrefix, "file-prefix", "", "prefix for output filenames")
	rootCmd.Flags().IntVar(&gpsBaudRate, "gps-baud", 0, "GPS serial port baud rate (for NMEA mode)")
	rootCmd.Flags().StringVar(&gpsTimeout, "gps-timeout", "", "GPS fix timeout duration")

	// Add subcommands
	rootCmd.AddCommand(devicesCmd)

	// Bind command line flags to viper configuration keys
	viper.BindPFlag("rtlsdr.frequency", rootCmd.Flags().Lookup("frequency"))
	viper.BindPFlag("collection.duration", rootCmd.Flags().Lookup("duration"))
	viper.BindPFlag("collection.output_dir", rootCmd.Flags().Lookup("output"))
	viper.BindPFlag("collection.synced_start", rootCmd.Flags().Lookup("synced-start"))
	viper.BindPFlag("collection.start_time", rootCmd.Flags().Lookup("start-time"))
	viper.BindPFlag("gps.mode", rootCmd.Flags().Lookup("gps-mode"))
	viper.BindPFlag("gps.port", rootCmd.Flags().Lookup("gps-port"))
	viper.BindPFlag("gps.gpsd_host", rootCmd.Flags().Lookup("gpsd-host"))
	viper.BindPFlag("gps.gpsd_port", rootCmd.Flags().Lookup("gpsd-port"))
	viper.BindPFlag("gps.manual_latitude", rootCmd.Flags().Lookup("latitude"))
	viper.BindPFlag("gps.manual_longitude", rootCmd.Flags().Lookup("longitude"))
	viper.BindPFlag("gps.manual_altitude", rootCmd.Flags().Lookup("altitude"))
	viper.BindPFlag("gps.disable", rootCmd.Flags().Lookup("disable-gps"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("rtlsdr.device", rootCmd.Flags().Lookup("device"))
	viper.BindPFlag("rtlsdr.gain", rootCmd.Flags().Lookup("gain"))
	viper.BindPFlag("rtlsdr.gain_mode", rootCmd.Flags().Lookup("gain-mode"))
	viper.BindPFlag("rtlsdr.bias_tee", rootCmd.Flags().Lookup("bias-tee"))
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config.yaml in current directory
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		configPath, _ := filepath.Abs(viper.ConfigFileUsed())
		fmt.Printf("Reading configuration file: %s\n", configPath)
	}
}

// runCollector is the main application logic
func runCollector(cmd *cobra.Command) error {
	// Load default configuration
	cfg := config.DefaultConfig()

	// Apply configuration with proper precedence: defaults < config file < command line
	applyConfiguration(cfg, cmd)

	// Handle device selection with proper precedence
	handleDeviceSelection(cfg, cmd)

	// Handle backward compatibility for GPS disable flag
	if cfg.GPS.Disable {
		// Backward compatibility: config file has disable: true
		cfg.GPS.Mode = "manual"
		if cfg.GPS.ManualLatitude == 0.0 && cfg.GPS.ManualLongitude == 0.0 {
			cfg.GPS.ManualLatitude = viper.GetFloat64("gps.manual_latitude")
			cfg.GPS.ManualLongitude = viper.GetFloat64("gps.manual_longitude")
			cfg.GPS.ManualAltitude = viper.GetFloat64("gps.manual_altitude")
		}
	}

	// Validate duration format
	if cfg.Collection.Duration == 0 {
		return fmt.Errorf("invalid duration: must be greater than 0")
	}

	// Validate GPS configuration
	switch cfg.GPS.Mode {
	case "manual":
		// Validate manual coordinates
		if cfg.GPS.ManualLatitude < -90 || cfg.GPS.ManualLatitude > 90 {
			return fmt.Errorf("invalid latitude: %.8f (must be between -90 and 90 degrees)", cfg.GPS.ManualLatitude)
		}
		if cfg.GPS.ManualLongitude < -180 || cfg.GPS.ManualLongitude > 180 {
			return fmt.Errorf("invalid longitude: %.8f (must be between -180 and 180 degrees)", cfg.GPS.ManualLongitude)
		}
		// Check if coordinates are set to default values (0,0) which likely means they weren't configured
		if cfg.GPS.ManualLatitude == 0.0 && cfg.GPS.ManualLongitude == 0.0 {
			return fmt.Errorf("manual coordinates not specified: set manual_latitude and manual_longitude in config file or use --latitude and --longitude flags")
		}
	case "nmea":
		// Validate NMEA serial port configuration
		if cfg.GPS.Port == "" {
			return fmt.Errorf("GPS port not specified for NMEA mode")
		}
	case "gpsd":
		// Validate gpsd configuration
		if cfg.GPS.GPSDHost == "" {
			return fmt.Errorf("GPSD host not specified for gpsd mode")
		}
		if cfg.GPS.GPSDPort == "" {
			return fmt.Errorf("GPSD port not specified for gpsd mode")
		}
	default:
		return fmt.Errorf("invalid GPS mode: %s (must be 'nmea', 'gpsd', or 'manual')", cfg.GPS.Mode)
	}

	// Display startup information
	fmt.Printf("Argus Collector %s starting...\n", version.GetFullVersion())

	switch cfg.GPS.Mode {
	case "manual":
		fmt.Printf("GPS: MANUAL MODE (using fixed coordinates)\n")
		fmt.Printf("Location: %.8f°, %.8f° (%.1f m)\n",
			cfg.GPS.ManualLatitude, cfg.GPS.ManualLongitude, cfg.GPS.ManualAltitude)
	case "nmea":
		fmt.Printf("GPS: NMEA MODE (serial port %s)\n", cfg.GPS.Port)
	case "gpsd":
		fmt.Printf("GPS: GPSD MODE (%s:%s)\n", cfg.GPS.GPSDHost, cfg.GPS.GPSDPort)
	}

	// Set up signal handling for graceful shutdown EARLY
	// This ensures Ctrl-C works even during initialization
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a context that can be cancelled by signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals in a separate goroutine
	go func() {
		<-sigChan
		fmt.Printf("\nReceived interrupt signal, shutting down...\n")
		cancel()   // Cancel the context to stop all operations
		os.Exit(1) // Force exit if graceful shutdown takes too long
	}()

	// Create and initialize collector
	c := collector.NewCollector(cfg)

	// Check for cancellation before initialization
	select {
	case <-ctx.Done():
		return fmt.Errorf("cancelled during setup")
	default:
	}

	if err := c.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize collector: %w", err)
	}
	defer c.Close()

	// Enable debug modes if verbose logging is enabled
	if viper.GetBool("verbose") {
		c.SetGPSDebug(true)
		c.SetRTLSDRVerbose(true)
	}

	// Check for cancellation before GPS fix
	select {
	case <-ctx.Done():
		return fmt.Errorf("cancelled before GPS fix")
	default:
	}

	// Wait for GPS fix before starting collection
	if err := c.WaitForGPSFixWithContext(ctx); err != nil {
		return fmt.Errorf("GPS initialization failed: %w", err)
	}

	// Check for cancellation before collection
	select {
	case <-ctx.Done():
		return fmt.Errorf("cancelled before collection")
	default:
	}

	// Perform signal collection
	if err := c.CollectWithContext(ctx); err != nil {
		return fmt.Errorf("collection failed: %w", err)
	}

	// Report final AGC result if AGC was used
	c.ReportAGCResult()
	
	fmt.Printf("Collection completed successfully.\n")
	return nil
}

// applyConfiguration applies configuration with proper precedence: defaults < config file < command line
func applyConfiguration(cfg *config.Config, cmd *cobra.Command) {
	// Apply config file values (overrides defaults)
	applyConfigFileValues(cfg)
	
	// Apply command line flags (overrides config file and defaults)
	applyCommandLineFlags(cfg, cmd)
}

// applyConfigFileValues applies configuration file values to override defaults
func applyConfigFileValues(cfg *config.Config) {
	// RTL-SDR configuration
	if viper.IsSet("rtlsdr.frequency") {
		cfg.RTLSDR.Frequency = viper.GetFloat64("rtlsdr.frequency")
	}
	if viper.IsSet("rtlsdr.sample_rate") {
		cfg.RTLSDR.SampleRate = uint32(viper.GetInt("rtlsdr.sample_rate"))
	}
	if viper.IsSet("rtlsdr.gain") {
		cfg.RTLSDR.Gain = viper.GetFloat64("rtlsdr.gain")
	}
	if viper.IsSet("rtlsdr.gain_mode") {
		cfg.RTLSDR.GainMode = viper.GetString("rtlsdr.gain_mode")
	}
	if viper.IsSet("rtlsdr.device_index") {
		cfg.RTLSDR.DeviceIndex = viper.GetInt("rtlsdr.device_index")
	}
	if viper.IsSet("rtlsdr.serial_number") {
		cfg.RTLSDR.SerialNumber = viper.GetString("rtlsdr.serial_number")
	}
	if viper.IsSet("rtlsdr.bias_tee") {
		cfg.RTLSDR.BiasTee = viper.GetBool("rtlsdr.bias_tee")
	}
	if viper.IsSet("rtlsdr.frequency_correction") {
		cfg.RTLSDR.FrequencyCorrection = viper.GetInt("rtlsdr.frequency_correction")
	}

	// GPS configuration
	if viper.IsSet("gps.mode") {
		cfg.GPS.Mode = viper.GetString("gps.mode")
	}
	if viper.IsSet("gps.port") {
		cfg.GPS.Port = viper.GetString("gps.port")
	}
	if viper.IsSet("gps.baud_rate") {
		cfg.GPS.BaudRate = viper.GetInt("gps.baud_rate")
	}
	if viper.IsSet("gps.gpsd_host") {
		cfg.GPS.GPSDHost = viper.GetString("gps.gpsd_host")
	}
	if viper.IsSet("gps.gpsd_port") {
		cfg.GPS.GPSDPort = viper.GetString("gps.gpsd_port")
	}
	if viper.IsSet("gps.timeout") {
		cfg.GPS.Timeout = viper.GetDuration("gps.timeout")
	}
	if viper.IsSet("gps.disable") {
		cfg.GPS.Disable = viper.GetBool("gps.disable")
	}
	if viper.IsSet("gps.manual_latitude") {
		cfg.GPS.ManualLatitude = viper.GetFloat64("gps.manual_latitude")
	}
	if viper.IsSet("gps.manual_longitude") {
		cfg.GPS.ManualLongitude = viper.GetFloat64("gps.manual_longitude")
	}
	if viper.IsSet("gps.manual_altitude") {
		cfg.GPS.ManualAltitude = viper.GetFloat64("gps.manual_altitude")
	}

	// Collection configuration
	if viper.IsSet("collection.duration") {
		if duration, err := time.ParseDuration(viper.GetString("collection.duration")); err == nil {
			cfg.Collection.Duration = duration
		}
	}
	if viper.IsSet("collection.output_dir") {
		cfg.Collection.OutputDir = viper.GetString("collection.output_dir")
	}
	if viper.IsSet("collection.file_prefix") {
		cfg.Collection.FilePrefix = viper.GetString("collection.file_prefix")
	}
	if viper.IsSet("collection.collection_id") {
		cfg.Collection.CollectionID = viper.GetString("collection.collection_id")
	}
	if viper.IsSet("collection.synced_start") {
		cfg.Collection.SyncedStart = viper.GetBool("collection.synced_start")
	}
	if viper.IsSet("collection.start_time") {
		cfg.Collection.StartTime = viper.GetInt64("collection.start_time")
	}

	// Logging configuration
	if viper.IsSet("logging.level") {
		cfg.Logging.Level = viper.GetString("logging.level")
	}
	if viper.IsSet("logging.file") {
		cfg.Logging.File = viper.GetString("logging.file")
	}
}

// applyCommandLineFlags applies command line flags to override config file and defaults
func applyCommandLineFlags(cfg *config.Config, cmd *cobra.Command) {
	// RTL-SDR flags
	if cmd.Flags().Changed("frequency") {
		cfg.RTLSDR.Frequency = frequency
	}
	if cmd.Flags().Changed("sample-rate") {
		cfg.RTLSDR.SampleRate = sampleRate
	}
	if cmd.Flags().Changed("gain") {
		cfg.RTLSDR.Gain = gain
	}
	if cmd.Flags().Changed("gain-mode") {
		cfg.RTLSDR.GainMode = gainMode
	}
	if cmd.Flags().Changed("bias-tee") {
		cfg.RTLSDR.BiasTee = biasTeeFlag
	}
	if cmd.Flags().Changed("frequency-correction") {
		cfg.RTLSDR.FrequencyCorrection = freqCorrection
	}

	// GPS flags
	if cmd.Flags().Changed("gps-mode") {
		cfg.GPS.Mode = gpsMode
	}
	if cmd.Flags().Changed("gps-port") {
		cfg.GPS.Port = gpsPort
	}
	if cmd.Flags().Changed("gps-baud") {
		cfg.GPS.BaudRate = gpsBaudRate
	}
	if cmd.Flags().Changed("gps-timeout") {
		if timeout, err := time.ParseDuration(gpsTimeout); err == nil {
			cfg.GPS.Timeout = timeout
		}
	}
	if cmd.Flags().Changed("gpsd-host") {
		cfg.GPS.GPSDHost = gpsdHost
	}
	if cmd.Flags().Changed("gpsd-port") {
		cfg.GPS.GPSDPort = gpsdPort
	}
	if cmd.Flags().Changed("latitude") {
		cfg.GPS.ManualLatitude = latitude
	}
	if cmd.Flags().Changed("longitude") {
		cfg.GPS.ManualLongitude = longitude
	}
	if cmd.Flags().Changed("altitude") {
		cfg.GPS.ManualAltitude = altitude
	}

	// Collection flags
	if cmd.Flags().Changed("duration") {
		if duration, err := time.ParseDuration(duration); err == nil {
			cfg.Collection.Duration = duration
		}
	}
	if cmd.Flags().Changed("output") {
		cfg.Collection.OutputDir = output
	}
	if cmd.Flags().Changed("file-prefix") {
		cfg.Collection.FilePrefix = filePrefix
	}
	if cmd.Flags().Changed("collection-id") {
		cfg.Collection.CollectionID = collectionID
	}
	if cmd.Flags().Changed("synced-start") {
		cfg.Collection.SyncedStart = syncedStart
	}
	if cmd.Flags().Changed("start-time") {
		cfg.Collection.StartTime = startTime
	}

	// Global flags
	if cmd.Flags().Changed("verbose") {
		// Verbose is handled separately in the main function
	}
}

// handleDeviceSelection handles RTL-SDR device selection with proper precedence
func handleDeviceSelection(cfg *config.Config, cmd *cobra.Command) {
	if cmd.Flags().Changed("device") {
		// Device flag explicitly set - override config file values
		deviceSelection := device

		// Treat as serial number if it contains non-digit characters or is longer than reasonable for an index
		// Also treat leading zeros as indication of serial number (e.g., "00000001")
		isSerial := false
		if len(deviceSelection) > 2 || strings.HasPrefix(deviceSelection, "0") && len(deviceSelection) > 1 {
			isSerial = true
		} else {
			// Check if it contains non-digit characters
			for _, r := range deviceSelection {
				if r < '0' || r > '9' {
					isSerial = true
					break
				}
			}
		}

		if isSerial {
			// It's a serial number
			cfg.RTLSDR.SerialNumber = deviceSelection
			cfg.RTLSDR.DeviceIndex = -1 // Set to -1 to indicate serial number should be used
		} else {
			// Try to parse as device index
			if deviceIndex, err := strconv.Atoi(deviceSelection); err == nil {
				cfg.RTLSDR.DeviceIndex = deviceIndex
				cfg.RTLSDR.SerialNumber = "" // Clear serial number when using index
			} else {
				// Fallback to treating as serial number
				cfg.RTLSDR.SerialNumber = deviceSelection
				cfg.RTLSDR.DeviceIndex = -1
			}
		}
	}
}

// listDevices lists all available RTL-SDR devices with their information
func listDevices() error {
	devices, err := rtlsdr.ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list RTL-SDR devices: %w", err)
	}

	fmt.Printf("Available RTL-SDR Devices:\n")
	fmt.Printf("=============================\n\n")

	for _, device := range devices {
		fmt.Printf("Device %d:\n", device.Index)
		fmt.Printf("  Name:         %s\n", device.Name)
		fmt.Printf("  Manufacturer: %s\n", device.Manufacturer)
		fmt.Printf("  Product:      %s\n", device.Product)
		fmt.Printf("  Serial:       %s\n", device.SerialNumber)
		fmt.Printf("\n")
	}

	fmt.Printf("Configuration Examples:\n")
	fmt.Printf("======================\n")
	fmt.Printf("# Use device by index (traditional method)\n")
	fmt.Printf("rtlsdr:\n")
	fmt.Printf("  device_index: 0\n\n")
	fmt.Printf("# Use device by serial number (recommended)\n")
	fmt.Printf("rtlsdr:\n")
	fmt.Printf("  serial_number: \"00000001\"\n\n")

	return nil
}

// main is the entry point of the application
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
