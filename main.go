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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	
	// Command-specific flags
	rootCmd.Flags().Float64VarP(&frequency, "frequency", "f", 433.92e6, "frequency to monitor (Hz)")
	rootCmd.Flags().StringVarP(&duration, "duration", "d", "60s", "collection duration")
	rootCmd.Flags().StringVarP(&output, "output", "o", "./data", "output directory")
	rootCmd.Flags().StringVar(&gpsPort, "gps-port", "/dev/ttyUSB0", "GPS serial port")
	rootCmd.Flags().BoolVar(&syncedStart, "synced-start", true, "enable delayed/synchronized start time (true|false)")
	
	// Bind command line flags to viper configuration keys
	viper.BindPFlag("rtlsdr.frequency", rootCmd.Flags().Lookup("frequency"))
	viper.BindPFlag("collection.duration", rootCmd.Flags().Lookup("duration"))
	viper.BindPFlag("collection.output_dir", rootCmd.Flags().Lookup("output"))
	viper.BindPFlag("gps.port", rootCmd.Flags().Lookup("gps-port"))
	viper.BindPFlag("collection.synced_start", rootCmd.Flags().Lookup("synced-start"))
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

	// Parse duration string into time.Duration
	durationParsed, err := time.ParseDuration(viper.GetString("collection.duration"))
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}
	cfg.Collection.Duration = durationParsed

	// Display startup information
	fmt.Printf("Argus Collector starting...\n")
	fmt.Printf("Frequency: %.2f MHz\n", cfg.RTLSDR.Frequency/1e6)
	fmt.Printf("Duration: %v\n", cfg.Collection.Duration)
	fmt.Printf("Output: %s\n", cfg.Collection.OutputDir)
	fmt.Printf("GPS Port: %s\n", cfg.GPS.Port)

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
