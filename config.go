package main

// Config contains all configurable settings for the application
type Config struct {
	// Color codes
	ColorRed   string
	ColorGreen string
	ColorReset string

	// Thresholds
	ThresholdCpuUsage            float64 // in %
	ThresholdLoadMultiplier      float64 // load average threshold = CPU_CORES * multiplier
	ThresholdMemUsage            float64 // in %
	ThresholdMemAvailablePercent float64 // less than 5% available memory is critical
	ThresholdDiskUtil            float64 // in %
	ThresholdNetUtil             float64 // in %
	ThresholdContextSwitchPerCore int     // per second
	ThresholdPageFaultRate       int     // per second (approximate)

	// Duration for sampling (seconds)
	SampleInterval int
	SampleCount    int
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() *Config {
	return &Config{
		// Color codes
		ColorRed:   "\033[31m",
		ColorGreen: "\033[32m",
		ColorReset: "\033[0m",

		// Thresholds
		ThresholdCpuUsage:            90.0,  // in %
		ThresholdLoadMultiplier:      2.0,   // load average threshold = CPU_CORES * multiplier
		ThresholdMemUsage:            75.0,  // in %
		ThresholdMemAvailablePercent: 5.0,   // less than 5% available memory is critical
		ThresholdDiskUtil:            90.0,  // in %
		ThresholdNetUtil:             70.0,  // in %
		ThresholdContextSwitchPerCore: 10000, // per second
		ThresholdPageFaultRate:       500,   // per second (approximate)

		// Duration for sampling (seconds)
		SampleInterval: 1,
		SampleCount:    3,
	}
}

// Global config instance
var AppConfig = GetDefaultConfig()