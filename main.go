// Argus Collector - RTL-SDR signal collection tool for TDOA analysis
// This program captures radio frequency signals using RTL-SDR hardware
// and GPS positioning data for Time Difference of Arrival analysis.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"argus-collector/internal/collector"
	"argus-collector/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Command line flag variables
var (
	cfgFile     string  // Configuration file path
	frequency   float64 // RF frequency to monitor in Hz
	duration    string  // Collection duration (e.g., "60s")
	output      string  // Output directory for data files
	gpsMode     string  // GPS mode: nmea, gpsd, or manual
	gpsPort     string  // GPS device serial port (for NMEA mode)
	gpsdHost    string  // GPSD host address (for gpsd mode)
	gpsdPort    string  // GPSD port (for gpsd mode)
	verbose     bool    // Enable verbose logging
	syncedStart bool    // Enable synchronized start timing
	disableGPS  bool    // Disable GPS hardware and use manual coordinates (deprecated)
	latitude    float64 // Manual latitude in decimal degrees
	longitude   float64 // Manual longitude in decimal degrees
	altitude    float64 // Manual altitude in meters
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "argus-collector",
	Short: "RTL-SDR signal collection tool for TDOA analysis",
	Long: `Argus Collector captures radio frequency signals using RTL-SDR hardware
and GPS positioning data for Time Difference of Arrival (TDOA) analysis.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runCollector(cmd); err != nil {
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
	
	// Command-specific flags
	rootCmd.Flags().Float64VarP(&frequency, "frequency", "f", 433.92e6, "frequency to monitor (Hz)")
	rootCmd.Flags().StringVarP(&duration, "duration", "d", "60s", "collection duration")
	rootCmd.Flags().StringVarP(&output, "output", "o", "./data", "output directory")
	rootCmd.Flags().BoolVar(&syncedStart, "synced-start", true, "enable delayed/synchronized start time (true|false)")
	
	// GPS configuration options
	rootCmd.Flags().StringVar(&gpsMode, "gps-mode", "nmea", "GPS mode: nmea, gpsd, or manual")
	rootCmd.Flags().StringVarP(&gpsPort, "gps-port", "p", "/dev/ttyUSB0", "GPS serial port (for NMEA mode)")
	rootCmd.Flags().StringVar(&gpsdHost, "gpsd-host", "localhost", "GPSD host address (for gpsd mode)")
	rootCmd.Flags().StringVar(&gpsdPort, "gpsd-port", "2947", "GPSD port (for gpsd mode)")
	
	// Manual GPS coordinates (for manual mode)
	rootCmd.Flags().Float64Var(&latitude, "latitude", 0.0, "manual latitude in decimal degrees (for manual mode)")
	rootCmd.Flags().Float64Var(&longitude, "longitude", 0.0, "manual longitude in decimal degrees (for manual mode)")
	rootCmd.Flags().Float64Var(&altitude, "altitude", 0.0, "manual altitude in meters (for manual mode)")
	
	// Deprecated GPS options (for backward compatibility)
	rootCmd.Flags().BoolVar(&disableGPS, "disable-gps", false, "disable GPS hardware and use manual coordinates (deprecated: use --gps-mode=manual)")
	
	// Bind command line flags to viper configuration keys
	viper.BindPFlag("rtlsdr.frequency", rootCmd.Flags().Lookup("frequency"))
	viper.BindPFlag("collection.duration", rootCmd.Flags().Lookup("duration"))
	viper.BindPFlag("collection.output_dir", rootCmd.Flags().Lookup("output"))
	viper.BindPFlag("collection.synced_start", rootCmd.Flags().Lookup("synced-start"))
	viper.BindPFlag("gps.mode", rootCmd.Flags().Lookup("gps-mode"))
	viper.BindPFlag("gps.port", rootCmd.Flags().Lookup("gps-port"))
	viper.BindPFlag("gps.gpsd_host", rootCmd.Flags().Lookup("gpsd-host"))
	viper.BindPFlag("gps.gpsd_port", rootCmd.Flags().Lookup("gpsd-port"))
	viper.BindPFlag("gps.manual_latitude", rootCmd.Flags().Lookup("latitude"))
	viper.BindPFlag("gps.manual_longitude", rootCmd.Flags().Lookup("longitude"))
	viper.BindPFlag("gps.manual_altitude", rootCmd.Flags().Lookup("altitude"))
	viper.BindPFlag("gps.disable", rootCmd.Flags().Lookup("disable-gps"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
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
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}

// runCollector is the main application logic
func runCollector(cmd *cobra.Command) error {
	// Load default configuration
	cfg := config.DefaultConfig()
	
	// Check if debug output is enabled
	debugEnabled := viper.GetBool("verbose") || viper.GetString("logging.level") == "debug"
	
	if debugEnabled {
		// Debug: Check if config file was found and loaded
		configFile := viper.ConfigFileUsed()
		if configFile != "" {
			fmt.Printf("Debug: Using config file: %s\n", configFile)
		} else {
			fmt.Printf("Debug: No config file found, using defaults\n")
		}
		
		// Debug: Test if we can read the file directly
		if _, err := os.Stat("./config.yaml"); err == nil {
			fmt.Printf("Debug: config.yaml file exists in current directory\n")
		} else {
			fmt.Printf("Debug: config.yaml file NOT found in current directory: %v\n", err)
		}
		
		// Debug: Check raw viper values before unmarshaling
		fmt.Printf("Debug: Raw viper values - gps.mode: '%s', gps.manual_latitude: %f, gps.manual_longitude: %f\n",
			viper.GetString("gps.mode"), viper.GetFloat64("gps.manual_latitude"), viper.GetFloat64("gps.manual_longitude"))
	}
	
	// Override with values from config file and command line flags
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	if debugEnabled {
		// Debug: Show what viper loaded from config file
		fmt.Printf("Debug: After viper.Unmarshal - GPS Mode: '%s', Lat: %.8f, Lon: %.8f, Alt: %.1f\n",
			cfg.GPS.Mode, cfg.GPS.ManualLatitude, cfg.GPS.ManualLongitude, cfg.GPS.ManualAltitude)
	}
	
	// Fix: Manually override GPS coordinates from viper since unmarshal isn't working for nested fields
	if cfg.GPS.ManualLatitude == 0.0 && cfg.GPS.ManualLongitude == 0.0 {
		viperLat := viper.GetFloat64("gps.manual_latitude")
		viperLon := viper.GetFloat64("gps.manual_longitude")
		viperAlt := viper.GetFloat64("gps.manual_altitude")
		
		if viperLat != 0.0 || viperLon != 0.0 {
			cfg.GPS.ManualLatitude = viperLat
			cfg.GPS.ManualLongitude = viperLon
			cfg.GPS.ManualAltitude = viperAlt
			if debugEnabled {
				fmt.Printf("Debug: Fixed GPS coordinates from viper - Lat: %.8f, Lon: %.8f, Alt: %.1f\n",
					cfg.GPS.ManualLatitude, cfg.GPS.ManualLongitude, cfg.GPS.ManualAltitude)
			}
		}
	}

	// Handle explicit command line flags that should override config file
	// Check if synced-start flag was explicitly set
	if cmd.Flags().Changed("synced-start") {
		// Command line flag overrides config file
		cfg.Collection.SyncedStart = syncedStart
		if debugEnabled {
			fmt.Printf("Debug: --synced-start flag explicitly set to: %t (overriding config file)\n", syncedStart)
		}
	} else {
		// Use viper value (config file or default)
		cfg.Collection.SyncedStart = viper.GetBool("collection.synced_start")
		if debugEnabled {
			fmt.Printf("Debug: Using config file/default synced_start: %t\n", cfg.Collection.SyncedStart)
		}
	}
	
	// Handle GPS mode configuration and backward compatibility
	if disableGPS {
		// Backward compatibility: --disable-gps flag overrides mode
		cfg.GPS.Mode = "manual"
		cfg.GPS.Disable = true // Keep for backward compatibility
		cfg.GPS.ManualLatitude = latitude
		cfg.GPS.ManualLongitude = longitude
		cfg.GPS.ManualAltitude = altitude
		if debugEnabled {
			fmt.Printf("Debug: GPS mode set to 'manual' via --disable-gps flag\n")
		}
	} else if cmd.Flags().Changed("gps-mode") {
		// GPS mode explicitly specified via --gps-mode flag
		cfg.GPS.Mode = gpsMode
		if gpsMode == "manual" {
			cfg.GPS.ManualLatitude = latitude
			cfg.GPS.ManualLongitude = longitude
			cfg.GPS.ManualAltitude = altitude
		}
		if debugEnabled {
			fmt.Printf("Debug: GPS mode explicitly set to '%s' via --gps-mode flag\n", gpsMode)
		}
	} else if cfg.GPS.Disable {
		// Backward compatibility: config file has disable: true
		cfg.GPS.Mode = "manual"
		cfg.GPS.ManualLatitude = viper.GetFloat64("gps.manual_latitude")
		cfg.GPS.ManualLongitude = viper.GetFloat64("gps.manual_longitude")
		cfg.GPS.ManualAltitude = viper.GetFloat64("gps.manual_altitude")
		if debugEnabled {
			fmt.Printf("Debug: GPS mode set to 'manual' via config file disable: true\n")
		}
	} else {
		// Use config file GPS mode (or default if not specified)
		// GPS mode, coordinates already loaded via viper.Unmarshal(cfg)
		if debugEnabled {
			fmt.Printf("Debug: Using GPS mode from config file: '%s'\n", cfg.GPS.Mode)
			fmt.Printf("Debug: Config file coordinates: lat=%.8f, lon=%.8f, alt=%.1f\n", 
				cfg.GPS.ManualLatitude, cfg.GPS.ManualLongitude, cfg.GPS.ManualAltitude)
		}
	}

	// Parse duration string into time.Duration
	durationParsed, err := time.ParseDuration(viper.GetString("collection.duration"))
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}
	cfg.Collection.Duration = durationParsed

	if debugEnabled {
		// Debug: Show final GPS configuration before validation
		fmt.Printf("Debug: Final GPS config before validation - Mode: '%s', Lat: %.8f, Lon: %.8f, Alt: %.1f\n",
			cfg.GPS.Mode, cfg.GPS.ManualLatitude, cfg.GPS.ManualLongitude, cfg.GPS.ManualAltitude)
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
	fmt.Printf("Argus Collector starting...\n")
	fmt.Printf("Frequency: %.2f MHz\n", cfg.RTLSDR.Frequency/1e6)
	fmt.Printf("Duration: %v\n", cfg.Collection.Duration)
	fmt.Printf("Output: %s\n", cfg.Collection.OutputDir)
	fmt.Printf("Synchronized Start: %t\n", cfg.Collection.SyncedStart)
	
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
		cancel() // Cancel the context to stop all operations
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
	
	// Enable GPS debug mode if verbose logging is enabled
	if viper.GetBool("verbose") {
		c.SetGPSDebug(true)
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

	fmt.Printf("Collection completed successfully.\n")
	return nil
}

// main is the entry point of the application
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
