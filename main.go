package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	
	"observe/sysinfo"
)

// The Config struct and related functions are now in config.go

func printAlert(message string) {
	fmt.Printf("%s%s%s\n", AppConfig.ColorRed, message, AppConfig.ColorReset)
}

func printOk(message string) {
	fmt.Printf("%s%s%s\n", AppConfig.ColorGreen, message, AppConfig.ColorReset)
}

// Prometheus metrics definitions
var (
	cpuUsageGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_cpu_usage_percent",
		Help: "Current CPU usage in percent",
	})
	
	loadAvgGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_load_average_15m",
		Help: "15-minute load average",
	})
	
	memUsedGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_memory_used_percent",
		Help: "Memory usage in percent",
	})
	
	memAvailableGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_memory_available_percent",
		Help: "Available memory in percent",
	})
	
	diskUtilGauges = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_disk_utilization_percent",
			Help: "Disk utilization in percent",
		},
		[]string{"disk"},
	)
	
	networkRxGauges = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_network_rx_mbps",
			Help: "Network receive bandwidth in Mbps",
		},
		[]string{"interface"},
	)
	
	networkTxGauges = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_network_tx_mbps",
			Help: "Network transmit bandwidth in Mbps",
		},
		[]string{"interface"},
	)
	
	contextSwitchesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_context_switches_per_second",
		Help: "Context switches per second",
	})
	
	pageFaultsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_page_faults_per_second",
		Help: "Page faults per second",
	})
	
	// Threshold breach metrics (1 if breached, 0 otherwise)
	cpuThresholdBreached = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_cpu_threshold_breached",
		Help: "1 if CPU usage exceeds threshold, 0 otherwise",
	})
	
	loadThresholdBreached = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_load_threshold_breached",
		Help: "1 if 15-minute load average exceeds threshold, 0 otherwise",
	})
	
	memUsageThresholdBreached = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_memory_usage_threshold_breached",
		Help: "1 if memory usage exceeds threshold, 0 otherwise",
	})
	
	memAvailableThresholdBreached = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_memory_available_threshold_breached",
		Help: "1 if available memory is below threshold, 0 otherwise",
	})
	
	diskUtilThresholdBreached = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_disk_threshold_breached",
			Help: "1 if disk utilization exceeds threshold, 0 otherwise",
		},
		[]string{"disk"},
	)
	
	networkRxThresholdBreached = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_network_rx_threshold_breached",
			Help: "1 if network receive bandwidth exceeds threshold, 0 otherwise",
		},
		[]string{"interface"},
	)
	
	networkTxThresholdBreached = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_network_tx_threshold_breached",
			Help: "1 if network transmit bandwidth exceeds threshold, 0 otherwise",
		},
		[]string{"interface"},
	)
	
	contextSwitchesThresholdBreached = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_context_switches_threshold_breached",
		Help: "1 if context switches per second exceeds threshold, 0 otherwise",
	})
	
	pageFaultsThresholdBreached = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_page_faults_threshold_breached",
		Help: "1 if page faults per second exceeds threshold, 0 otherwise",
	})
)

func collectMetricsLoop(collector sysinfo.MetricsCollector) {
	for {
		// Collect and update CPU usage metric
		if AppConfig.CollectCpuMetrics {
			cpuUsage, err := collector.GetCPUUsage()
			if err == nil {
				cpuUsageGauge.Set(cpuUsage)
				
				// Set CPU threshold breach metric
				if cpuUsage > AppConfig.ThresholdCpuUsage {
					cpuThresholdBreached.Set(1)
				} else {
					cpuThresholdBreached.Set(0)
				}
			}
		}
		
		// Collect and update load average metric
		if AppConfig.CollectLoadMetrics {
			load15, err := collector.GetLoadAverage()
			if err == nil {
				loadAvgGauge.Set(load15)
				
				// Set load threshold breach metric
				// Load threshold = number of CPU cores * multiplier
				loadThreshold := float64(sysInfo.CPUCores) * AppConfig.ThresholdLoadMultiplier
				if load15 > loadThreshold {
					loadThresholdBreached.Set(1)
				} else {
					loadThresholdBreached.Set(0)
				}
			}
		}
		
		// Collect and update memory usage metrics
		if AppConfig.CollectMemoryMetrics {
			memUsed, memAvailable, err := collector.GetMemoryUsage()
			if err == nil {
				memUsedGauge.Set(memUsed)
				memAvailableGauge.Set(memAvailable)
				
				// Set memory usage threshold breach metrics
				if memUsed > AppConfig.ThresholdMemUsage {
					memUsageThresholdBreached.Set(1)
				} else {
					memUsageThresholdBreached.Set(0)
				}
				
				if memAvailable < AppConfig.ThresholdMemAvailablePercent {
					memAvailableThresholdBreached.Set(1)
				} else {
					memAvailableThresholdBreached.Set(0)
				}
			}
		}
		
		// Collect and update disk utilization metrics
		if AppConfig.CollectDiskMetrics {
			diskUtils, err := collector.GetDiskUtilization()
			if err == nil {
				for disk, util := range diskUtils {
					diskUtilGauges.WithLabelValues(disk).Set(util)
					
					// Set disk utilization threshold breach metric
					if util > AppConfig.ThresholdDiskUtil {
						diskUtilThresholdBreached.WithLabelValues(disk).Set(1)
					} else {
						diskUtilThresholdBreached.WithLabelValues(disk).Set(0)
					}
				}
			}
		}
		
		// Collect and update network bandwidth metrics
		if AppConfig.CollectNetworkMetrics {
			netBandwidth, err := collector.GetNetworkBandwidth()
			if err == nil {
				for iface, rates := range netBandwidth {
					rxRate := rates[0]
					txRate := rates[1]
					
					networkRxGauges.WithLabelValues(iface).Set(rxRate)
					networkTxGauges.WithLabelValues(iface).Set(txRate)
					
					// Set network threshold breach metrics
					// Network threshold is based on a percentage of a typical 1Gbps link
					netThreshold := AppConfig.ThresholdNetUtil
					
					if rxRate > netThreshold {
						networkRxThresholdBreached.WithLabelValues(iface).Set(1)
					} else {
						networkRxThresholdBreached.WithLabelValues(iface).Set(0)
					}
					
					if txRate > netThreshold {
						networkTxThresholdBreached.WithLabelValues(iface).Set(1)
					} else {
						networkTxThresholdBreached.WithLabelValues(iface).Set(0)
					}
				}
			}
		}
		
		// Collect and update context switches metric
		if AppConfig.CollectContextSwitchMetrics {
			contextSwitches, err := collector.GetContextSwitches()
			if err == nil {
				contextSwitchesGauge.Set(float64(contextSwitches))
				
				// Set context switches threshold breach metric
				// Context switches threshold is per core
				csThreshold := AppConfig.ThresholdContextSwitchPerCore * sysInfo.CPUCores
				
				if contextSwitches > csThreshold {
					contextSwitchesThresholdBreached.Set(1)
				} else {
					contextSwitchesThresholdBreached.Set(0)
				}
			}
		}
		
		// Collect and update page faults metric
		if AppConfig.CollectPageFaultMetrics {
			pageFaults, err := collector.GetPageFaults()
			if err == nil {
				pageFaultsGauge.Set(float64(pageFaults))
				
				// Set page faults threshold breach metric
				if pageFaults > AppConfig.ThresholdPageFaultRate {
					pageFaultsThresholdBreached.Set(1)
				} else {
					pageFaultsThresholdBreached.Set(0)
				}
			}
		}
		
		// Sleep for the sample interval before collecting metrics again
		time.Sleep(time.Duration(AppConfig.SampleInterval) * time.Second)
	}
}

// Global variables for system information
var sysInfo sysinfo.SystemInfo

func main() {
	// Define command-line flag for configuration file
	configFlag := flag.String("config", "config.yaml", "Path to YAML configuration file")
	
	// Define command-line flags to enable/disable metrics
	cpuFlag := flag.Bool("cpu", true, "Enable CPU metrics collection")
	loadFlag := flag.Bool("load", true, "Enable load average metrics collection")
	memoryFlag := flag.Bool("memory", true, "Enable memory metrics collection")
	diskFlag := flag.Bool("disk", true, "Enable disk utilization metrics collection")
	networkFlag := flag.Bool("network", true, "Enable network bandwidth metrics collection")
	contextSwitchFlag := flag.Bool("ctx-switch", true, "Enable context switch metrics collection")
	pageFaultFlag := flag.Bool("page-fault", true, "Enable page fault metrics collection")
	allFlag := flag.Bool("all", true, "Enable all metrics collection (overrides individual settings)")
	flag.Parse()
	
	// Load configuration from specified YAML file
	InitConfigFromPath(*configFlag)
	
	// Override configuration with command-line flags
	if !*allFlag {
		AppConfig.CollectCpuMetrics = *cpuFlag
		AppConfig.CollectLoadMetrics = *loadFlag
		AppConfig.CollectMemoryMetrics = *memoryFlag
		AppConfig.CollectDiskMetrics = *diskFlag
		AppConfig.CollectNetworkMetrics = *networkFlag
		AppConfig.CollectContextSwitchMetrics = *contextSwitchFlag
		AppConfig.CollectPageFaultMetrics = *pageFaultFlag
	} else {
		// Enable all metrics if -all flag is true
		AppConfig.CollectCpuMetrics = true
		AppConfig.CollectLoadMetrics = true
		AppConfig.CollectMemoryMetrics = true
		AppConfig.CollectDiskMetrics = true
		AppConfig.CollectNetworkMetrics = true
		AppConfig.CollectContextSwitchMetrics = true
		AppConfig.CollectPageFaultMetrics = true
	}

	// Register Prometheus metrics based on configuration flags
	if AppConfig.CollectCpuMetrics {
		prometheus.MustRegister(cpuUsageGauge)
		prometheus.MustRegister(cpuThresholdBreached)
	}
	
	if AppConfig.CollectLoadMetrics {
		prometheus.MustRegister(loadAvgGauge)
		prometheus.MustRegister(loadThresholdBreached)
	}
	
	if AppConfig.CollectMemoryMetrics {
		prometheus.MustRegister(memUsedGauge)
		prometheus.MustRegister(memAvailableGauge)
		prometheus.MustRegister(memUsageThresholdBreached)
		prometheus.MustRegister(memAvailableThresholdBreached)
	}
	
	if AppConfig.CollectDiskMetrics {
		prometheus.MustRegister(diskUtilGauges)
		prometheus.MustRegister(diskUtilThresholdBreached)
	}
	
	if AppConfig.CollectNetworkMetrics {
		prometheus.MustRegister(networkRxGauges)
		prometheus.MustRegister(networkTxGauges)
		prometheus.MustRegister(networkRxThresholdBreached)
		prometheus.MustRegister(networkTxThresholdBreached)
	}
	
	if AppConfig.CollectContextSwitchMetrics {
		prometheus.MustRegister(contextSwitchesGauge)
		prometheus.MustRegister(contextSwitchesThresholdBreached)
	}
	
	if AppConfig.CollectPageFaultMetrics {
		prometheus.MustRegister(pageFaultsGauge)
		prometheus.MustRegister(pageFaultsThresholdBreached)
	}
	
	// Get system information
	sysInfo = sysinfo.GetSystemInfo()
	
	// Check if the current OS and architecture are supported
	if !sysInfo.IsSupported {
		printAlert(fmt.Sprintf("Warning: Your system (%s) may not be fully supported. Metrics collection may not work correctly.", sysInfo.SystemDescription()))
	}
	
	// Create the appropriate metrics collector for the current OS
	collector, err := sysinfo.NewMetricsCollector(AppConfig.SampleInterval)
	if err != nil {
		printAlert(fmt.Sprintf("Error creating metrics collector: %v", err))
		os.Exit(1)
	}
	
	// Get total memory
	totalMemMB, err := collector.GetTotalMemoryMB()
	if err != nil {
		printAlert(fmt.Sprintf("Error getting total memory: %v", err))
		os.Exit(1)
	}
	
	// Start metrics collection in a goroutine
	go collectMetricsLoop(collector)
	
	// Print startup information
	fmt.Println("=== observeT Service Started ===")
	fmt.Printf("System: %s\n", sysInfo.SystemDescription())
	fmt.Printf("Total Memory: %dMB\n", totalMemMB)
	fmt.Printf("Sample interval: %ds, Samples: %d\n", AppConfig.SampleInterval, AppConfig.SampleCount)
	fmt.Println("Metrics are available at http://localhost:9095/metrics")
	fmt.Println("Press Ctrl+C to stop the service")
	
	// Start HTTP server for Prometheus metrics (blocking call)
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Starting Prometheus metrics server on :9095")
	if err := http.ListenAndServe(":9095", nil); err != nil {
		printAlert(fmt.Sprintf("Error starting Prometheus HTTP server: %v", err))
		os.Exit(1)
	}
}