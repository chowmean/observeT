package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConfigYAML represents the structure of the YAML configuration file
type ConfigYAML struct {
	Colors struct {
		Red   string `yaml:"red"`
		Green string `yaml:"green"`
		Reset string `yaml:"reset"`
	} `yaml:"colors"`
	Thresholds struct {
		CpuUsage            float64 `yaml:"cpuUsage"`
		LoadMultiplier      float64 `yaml:"loadMultiplier"`
		MemUsage            float64 `yaml:"memUsage"`
		MemAvailablePercent float64 `yaml:"memAvailablePercent"`
		DiskUtil            float64 `yaml:"diskUtil"`
		NetUtil             float64 `yaml:"netUtil"`
		ContextSwitchPerCore int    `yaml:"contextSwitchPerCore"`
		PageFaultRate       int    `yaml:"pageFaultRate"`
	} `yaml:"thresholds"`
	Sampling struct {
		Interval int `yaml:"interval"`
		Count    int `yaml:"count"`
	} `yaml:"sampling"`
	Metrics struct {
		CollectCpu           bool `yaml:"collectCpu"`
		CollectLoad          bool `yaml:"collectLoad"`
		CollectMemory        bool `yaml:"collectMemory"`
		CollectDisk          bool `yaml:"collectDisk"`
		CollectNetwork       bool `yaml:"collectNetwork"`
		CollectContextSwitch bool `yaml:"collectContextSwitch"`
		CollectPageFault     bool `yaml:"collectPageFault"`
	} `yaml:"metrics"`
}

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
	
	// Metric collection flags
	CollectCpuMetrics          bool
	CollectLoadMetrics         bool
	CollectMemoryMetrics       bool
	CollectDiskMetrics         bool
	CollectNetworkMetrics      bool
	CollectContextSwitchMetrics bool
	CollectPageFaultMetrics    bool
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
		
		// Metric collection flags - all enabled by default
		CollectCpuMetrics:          true,
		CollectLoadMetrics:         true,
		CollectMemoryMetrics:       true,
		CollectDiskMetrics:         true,
		CollectNetworkMetrics:      true,
		CollectContextSwitchMetrics: true,
		CollectPageFaultMetrics:    true,
	}
}

// LoadConfigFromYAML loads configuration from a YAML file
func LoadConfigFromYAML(configPath string) (*Config, error) {
	// Start with default config
	config := GetDefaultConfig()
	
	// Read YAML file
	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse YAML into ConfigYAML struct
	var yamlConfig ConfigYAML
	if err := yaml.Unmarshal(yamlFile, &yamlConfig); err != nil {
		return config, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Transfer values from YAML config to the application config
	config.ColorRed = yamlConfig.Colors.Red
	config.ColorGreen = yamlConfig.Colors.Green
	config.ColorReset = yamlConfig.Colors.Reset
	
	config.ThresholdCpuUsage = yamlConfig.Thresholds.CpuUsage
	config.ThresholdLoadMultiplier = yamlConfig.Thresholds.LoadMultiplier
	config.ThresholdMemUsage = yamlConfig.Thresholds.MemUsage
	config.ThresholdMemAvailablePercent = yamlConfig.Thresholds.MemAvailablePercent
	config.ThresholdDiskUtil = yamlConfig.Thresholds.DiskUtil
	config.ThresholdNetUtil = yamlConfig.Thresholds.NetUtil
	config.ThresholdContextSwitchPerCore = yamlConfig.Thresholds.ContextSwitchPerCore
	config.ThresholdPageFaultRate = yamlConfig.Thresholds.PageFaultRate
	
	config.SampleInterval = yamlConfig.Sampling.Interval
	config.SampleCount = yamlConfig.Sampling.Count
	
	config.CollectCpuMetrics = yamlConfig.Metrics.CollectCpu
	config.CollectLoadMetrics = yamlConfig.Metrics.CollectLoad
	config.CollectMemoryMetrics = yamlConfig.Metrics.CollectMemory
	config.CollectDiskMetrics = yamlConfig.Metrics.CollectDisk
	config.CollectNetworkMetrics = yamlConfig.Metrics.CollectNetwork
	config.CollectContextSwitchMetrics = yamlConfig.Metrics.CollectContextSwitch
	config.CollectPageFaultMetrics = yamlConfig.Metrics.CollectPageFault
	
	return config, nil
}

// Global config instance
var AppConfig = GetDefaultConfig()

// InitConfig attempts to load configuration from YAML file, falls back to defaults if not found
func InitConfig() {
	// Try to find config file in current directory
	configPath := "config.yaml"
	
	// Check for existence of the config file
	if _, err := os.Stat(configPath); err == nil {
		// Config file exists, try to load it
		config, err := LoadConfigFromYAML(configPath)
		if err != nil {
			fmt.Printf("Warning: Failed to load config from %s: %v\nUsing default config.\n", configPath, err)
		} else {
			AppConfig = config
			fmt.Printf("Loaded configuration from %s\n", configPath)
		}
	} else {
		fmt.Printf("No config file found at %s. Using default config.\n", configPath)
	}
}

// InitConfigFromPath loads configuration from a specified YAML file path
func InitConfigFromPath(configPath string) {
	// Check for existence of the config file
	if _, err := os.Stat(configPath); err == nil {
		// Config file exists, try to load it
		config, err := LoadConfigFromYAML(configPath)
		if err != nil {
			fmt.Printf("Warning: Failed to load config from %s: %v\nUsing default config.\n", configPath, err)
		} else {
			AppConfig = config
			fmt.Printf("Loaded configuration from %s\n", configPath)
		}
	} else {
		fmt.Printf("No config file found at %s. Using default config.\n", configPath)
	}
}