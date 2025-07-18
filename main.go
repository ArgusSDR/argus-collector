// Argus Collector - RTL-SDR signal collection tool for TDOA analysis
// This program captures radio frequency signals using RTL-SDR hardware
// and GPS positioning data for Time Difference of Arrival analysis.
package main

import (
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
	gpsPort     string  // GPS device serial port
	verbose     bool    // Enable verbose logging
	syncedStart bool    // Enable synchronized start timing
	disableGPS  bool    // Disable GPS hardware and use manual coordinates
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
		if err := runCollector(); err != nil {
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
	rootCmd.Flags().StringVarP(&gpsPort, "gps-port", "p", "/dev/ttyUSB0", "GPS serial port")
	rootCmd.Flags().BoolVar(&syncedStart, "synced-start", true, "enable delayed/synchronized start time (true|false)")
	
	// GPS override options
	rootCmd.Flags().BoolVar(&disableGPS, "disable-gps", false, "disable GPS hardware and use manual coordinates")
	rootCmd.Flags().Float64Var(&latitude, "latitude", 0.0, "manual latitude in decimal degrees (requires --disable-gps)")
	rootCmd.Flags().Float64Var(&longitude, "longitude", 0.0, "manual longitude in decimal degrees (requires --disable-gps)")
	rootCmd.Flags().Float64Var(&altitude, "altitude", 0.0, "manual altitude in meters (requires --disable-gps)")
	
	// Bind command line flags to viper configuration keys
	viper.BindPFlag("rtlsdr.frequency", rootCmd.Flags().Lookup("frequency"))
	viper.BindPFlag("collection.duration", rootCmd.Flags().Lookup("duration"))
	viper.BindPFlag("collection.output_dir", rootCmd.Flags().Lookup("output"))
	viper.BindPFlag("gps.port", rootCmd.Flags().Lookup("gps-port"))
	viper.BindPFlag("collection.synced_start", rootCmd.Flags().Lookup("synced-start"))
	viper.BindPFlag("gps.disable", rootCmd.Flags().Lookup("disable-gps"))
	viper.BindPFlag("gps.manual_latitude", rootCmd.Flags().Lookup("latitude"))
	viper.BindPFlag("gps.manual_longitude", rootCmd.Flags().Lookup("longitude"))
	viper.BindPFlag("gps.manual_altitude", rootCmd.Flags().Lookup("altitude"))
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
func runCollector() error {
	// Load default configuration
	cfg := config.DefaultConfig()
	
	// Override with values from config file and command line flags
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override GPS settings with command line values if provided
	if disableGPS {
		cfg.GPS.Disable = true
		cfg.GPS.ManualLatitude = latitude
		cfg.GPS.ManualLongitude = longitude
		cfg.GPS.ManualAltitude = altitude
	} else if cfg.GPS.Disable {
		// GPS is disabled via config file, use viper values directly since Unmarshal didn't work properly
		cfg.GPS.ManualLatitude = viper.GetFloat64("gps.manual_latitude")
		cfg.GPS.ManualLongitude = viper.GetFloat64("gps.manual_longitude")
		cfg.GPS.ManualAltitude = viper.GetFloat64("gps.manual_altitude")
	}

	// Parse duration string into time.Duration
	durationParsed, err := time.ParseDuration(viper.GetString("collection.duration"))
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}
	cfg.Collection.Duration = durationParsed

	// Validate GPS configuration
	if cfg.GPS.Disable {
		// When GPS is disabled, validate manual coordinates
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
	}

	// Display startup information
	fmt.Printf("Argus Collector starting...\n")
	fmt.Printf("Frequency: %.2f MHz\n", cfg.RTLSDR.Frequency/1e6)
	fmt.Printf("Duration: %v\n", cfg.Collection.Duration)
	fmt.Printf("Output: %s\n", cfg.Collection.OutputDir)
	
	if cfg.GPS.Disable {
		fmt.Printf("GPS: DISABLED (using manual coordinates)\n")
		fmt.Printf("Location: %.8f°, %.8f° (%.1f m)\n", 
			cfg.GPS.ManualLatitude, cfg.GPS.ManualLongitude, cfg.GPS.ManualAltitude)
	} else {
		fmt.Printf("GPS Port: %s\n", cfg.GPS.Port)
	}

	// Create and initialize collector
	c := collector.NewCollector(cfg)
	
	if err := c.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize collector: %w", err)
	}
	defer c.Close()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Handle interrupt signals in a separate goroutine
	go func() {
		<-sigChan
		fmt.Printf("\nReceived interrupt signal, shutting down...\n")
		c.Stop()
	}()

	// Wait for GPS fix before starting collection
	if err := c.WaitForGPSFix(); err != nil {
		return fmt.Errorf("GPS initialization failed: %w", err)
	}

	// Perform signal collection
	if err := c.Collect(); err != nil {
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
