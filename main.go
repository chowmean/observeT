package main

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func printAlert(message string) {
	fmt.Printf("%s%s%s\n", AppConfig.ColorRed, message, AppConfig.ColorReset)
}

func printOk(message string) {
	fmt.Printf("%s%s%s\n", AppConfig.ColorGreen, message, AppConfig.ColorReset)
}

func getCPUCores() int {
	return runtime.NumCPU()
}

func getTotalMemoryMB() (int, error) {
	// For macOS, use sysctl to get memory information
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	memBytes, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, err
	}

	// Convert bytes to MB
	return int(memBytes / 1024 / 1024), nil
}

func getCPUUsage() (float64, error) {
	// For macOS, use top command to get CPU usage
	cmd := exec.Command("top", "-l", "2", "-n", "0", "-s", strconv.Itoa(AppConfig.SampleInterval))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	var cpuLine string
	
	// Find the CPU usage line from the second sample
	secondSample := false
	for _, line := range lines {
		if strings.Contains(line, "CPU usage") {
			if secondSample {
				cpuLine = line
				break
			}
			secondSample = true
		}
	}

	if cpuLine == "" {
		return 0, fmt.Errorf("could not find CPU usage in top output")
	}

	// Parse the CPU usage line
	// Format is typically: "CPU usage: x.xx% user, y.yy% sys, z.zz% idle"
	parts := strings.Split(cpuLine, ",")
	if len(parts) < 3 {
		return 0, fmt.Errorf("unexpected format in top CPU output")
	}

	// Get the idle percentage
	idlePart := strings.TrimSpace(parts[2])
	idleStr := strings.Split(idlePart, "%")[0]
	idle, err := strconv.ParseFloat(strings.Split(idleStr, " ")[0], 64)
	if err != nil {
		return 0, err
	}

	// Calculate CPU usage as 100 - idle
	return 100 - idle, nil
}

func getLoadAverage() (float64, error) {
	// For macOS, use sysctl to get load average
	cmd := exec.Command("sysctl", "-n", "vm.loadavg")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	// Output format is: { x.xx y.yy z.zz }
	parts := strings.Fields(string(output))
	if len(parts) < 5 {
		return 0, fmt.Errorf("unexpected format in load average output")
	}

	// Get the 15-minute load average (third number)
	load15, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return 0, err
	}

	return load15, nil
}

func getMemoryUsage() (float64, float64, error) {
	// For macOS, use vm_stat to get memory information
	cmd := exec.Command("vm_stat")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(output), "\n")
	
	var pageSize int64 = 4096 // Default page size, could be different
	var free, inactive, active, wired, compressed int64

	// Get page size from vm_stat output or use sysctl as fallback
	for _, line := range lines {
		if strings.Contains(line, "page size of") {
			parts := strings.Fields(line)
			if len(parts) >= 7 {
				pageSize, err = strconv.ParseInt(parts[7], 10, 64)
				if err != nil {
					pageSize = 4096 // Default to 4KB if parsing fails
				}
			}
			break
		}
	}

	// Get memory usage statistics
	for _, line := range lines {
		if strings.HasPrefix(line, "Pages free:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				val, err := strconv.ParseInt(strings.ReplaceAll(parts[2], ".", ""), 10, 64)
				if err == nil {
					free = val
				}
			}
		} else if strings.HasPrefix(line, "Pages inactive:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				val, err := strconv.ParseInt(strings.ReplaceAll(parts[2], ".", ""), 10, 64)
				if err == nil {
					inactive = val
				}
			}
		} else if strings.HasPrefix(line, "Pages active:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				val, err := strconv.ParseInt(strings.ReplaceAll(parts[2], ".", ""), 10, 64)
				if err == nil {
					active = val
				}
			}
		} else if strings.HasPrefix(line, "Pages wired down:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				val, err := strconv.ParseInt(strings.ReplaceAll(parts[3], ".", ""), 10, 64)
				if err == nil {
					wired = val
				}
			}
		} else if strings.HasPrefix(line, "Pages occupied by compressor:") {
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				val, err := strconv.ParseInt(strings.ReplaceAll(parts[4], ".", ""), 10, 64)
				if err == nil {
					compressed = val
				}
			}
		}
	}

	// Get total physical memory
	cmdMem := exec.Command("sysctl", "-n", "hw.memsize")
	outputMem, err := cmdMem.CombinedOutput()
	if err != nil {
		return 0, 0, err
	}
	totalBytes, err := strconv.ParseInt(strings.TrimSpace(string(outputMem)), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	
	totalPages := totalBytes / pageSize
	
	// Calculate memory usage
	usedPages := active + wired + compressed
	availablePages := free + inactive
	
	usedPercent := float64(usedPages) / float64(totalPages) * 100
	availablePercent := float64(availablePages) / float64(totalPages) * 100
	
	return usedPercent, availablePercent, nil
}

func getDiskUtilization() (map[string]float64, error) {
	// For macOS, use iostat to get disk utilization
	cmd := exec.Command("iostat", "-d", "-c", "2")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	disks := make(map[string]float64)

	// Skip header lines and process disk stats
	inDisks := false
	for _, line := range lines {
		if strings.Contains(line, "disk") && strings.Contains(line, "KB/t") {
			inDisks = true
			continue
		}
		
		if inDisks && len(line) > 0 && !strings.HasPrefix(line, "disk") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				diskName := fields[0]
				// On macOS, iostat doesn't directly give utilization percentage
				// We use the busy column (KB/t * tps) as an approximation
				if len(fields) >= 4 {
					tps, err := strconv.ParseFloat(fields[2], 64)
					if err == nil {
						// Use tps (transfers per second) as a rough approximation of utilization
						// Since macOS doesn't provide direct % utilization
						utilEstimate := math.Min(tps/10, 100) // Scale tps to rough % (capped at 100%)
						disks[diskName] = utilEstimate
					}
				}
			}
		}
	}

	return disks, nil
}

func getNetworkBandwidth() (map[string][2]float64, error) {
	// For macOS, use netstat to get network bandwidth
	cmd := exec.Command("netstat", "-ib")
	output1, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	
	time.Sleep(time.Duration(AppConfig.SampleInterval) * time.Second)
	
	cmd = exec.Command("netstat", "-ib")
	output2, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	
	lines1 := strings.Split(string(output1), "\n")
	lines2 := strings.Split(string(output2), "\n")
	
	// Parse both outputs to compare bytes in and out
	interfaces := make(map[string][2]float64)
	stats1 := make(map[string][2]int64)
	stats2 := make(map[string][2]int64)
	
	// Process first sample
	for i, line := range lines1 {
		if i > 0 && len(line) > 0 { // Skip header
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				ifaceName := fields[0]
				// Skip loopback interface
				if ifaceName == "lo0" {
					continue
				}
				
				ibytes, err1 := strconv.ParseInt(fields[6], 10, 64)
				obytes, err2 := strconv.ParseInt(fields[9], 10, 64)
				
				if err1 == nil && err2 == nil {
					stats1[ifaceName] = [2]int64{ibytes, obytes}
				}
			}
		}
	}
	
	// Process second sample
	for i, line := range lines2 {
		if i > 0 && len(line) > 0 { // Skip header
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				ifaceName := fields[0]
				// Skip loopback
				if ifaceName == "lo0" {
					continue
				}
				
				ibytes, err1 := strconv.ParseInt(fields[6], 10, 64)
				obytes, err2 := strconv.ParseInt(fields[9], 10, 64)
				
				if err1 == nil && err2 == nil {
					stats2[ifaceName] = [2]int64{ibytes, obytes}
				}
			}
		}
	}
	
	// Calculate rates
	for iface, stat2 := range stats2 {
		if stat1, ok := stats1[iface]; ok {
			rxRate := float64(stat2[0]-stat1[0]) / float64(AppConfig.SampleInterval) * 8 / 1000000 // Convert to Mbps
			txRate := float64(stat2[1]-stat1[1]) / float64(AppConfig.SampleInterval) * 8 / 1000000 // Convert to Mbps
			interfaces[iface] = [2]float64{rxRate, txRate}
		}
	}
	
	return interfaces, nil
}

func getContextSwitches() (int, error) {
	// For macOS, we use the "sysctlbyname" with "vm.stats.sys.v_swtch"
	// But since we can't directly call this from Go, we'll use the sysctl command
	cmd1 := exec.Command("sysctl", "-n", "vm.stats.sys.v_swtch")
	output1, err := cmd1.CombinedOutput()
	if err != nil {
		return 0, err
	}
	
	cs1, err := strconv.Atoi(strings.TrimSpace(string(output1)))
	if err != nil {
		return 0, err
	}
	
	time.Sleep(time.Duration(AppConfig.SampleInterval) * time.Second)
	
	cmd2 := exec.Command("sysctl", "-n", "vm.stats.sys.v_swtch")
	output2, err := cmd2.CombinedOutput()
	if err != nil {
		return 0, err
	}
	
	cs2, err := strconv.Atoi(strings.TrimSpace(string(output2)))
	if err != nil {
		return 0, err
	}
	
	return cs2 - cs1, nil
}

func getPageFaults() (int, error) {
	// For macOS, we'll use vm.stats.vm.v_fault
	cmd1 := exec.Command("sysctl", "-n", "vm.stats.vm.v_fault")
	output1, err := cmd1.CombinedOutput()
	if err != nil {
		return 0, err
	}
	
	pf1, err := strconv.Atoi(strings.TrimSpace(string(output1)))
	if err != nil {
		return 0, err
	}
	
	time.Sleep(time.Duration(AppConfig.SampleInterval) * time.Second)
	
	cmd2 := exec.Command("sysctl", "-n", "vm.stats.vm.v_fault")
	output2, err := cmd2.CombinedOutput()
	if err != nil {
		return 0, err
	}
	
	pf2, err := strconv.Atoi(strings.TrimSpace(string(output2)))
	if err != nil {
		return 0, err
	}
	
	return pf2 - pf1, nil
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

func collectMetricsLoop() {
	for {
		// Collect and update CPU usage metric
		cpuUsage, err := getCPUUsage()
		if err == nil {
			cpuUsageGauge.Set(cpuUsage)
			
			// Set CPU threshold breach metric
			if cpuUsage > AppConfig.ThresholdCpuUsage {
				cpuThresholdBreached.Set(1)
			} else {
				cpuThresholdBreached.Set(0)
			}
		}
		
		// Collect and update load average metric
		load15, err := getLoadAverage()
		if err == nil {
			loadAvgGauge.Set(load15)
			
			// Set load threshold breach metric
			// Load threshold = number of CPU cores * multiplier
			cpuCores := getCPUCores()
			loadThreshold := float64(cpuCores) * AppConfig.ThresholdLoadMultiplier
			if load15 > loadThreshold {
				loadThresholdBreached.Set(1)
			} else {
				loadThresholdBreached.Set(0)
			}
		}
		
		// Collect and update memory usage metrics
		memUsed, memAvailable, err := getMemoryUsage()
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
		
		// Collect and update disk utilization metrics
		diskUtils, err := getDiskUtilization()
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
		
		// Collect and update network bandwidth metrics
		netBandwidth, err := getNetworkBandwidth()
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
		
		// Collect and update context switches metric
		contextSwitches, err := getContextSwitches()
		if err == nil {
			contextSwitchesGauge.Set(float64(contextSwitches))
			
			// Set context switches threshold breach metric
			// Context switches threshold is per core
			cpuCores := getCPUCores()
			csThreshold := AppConfig.ThresholdContextSwitchPerCore * cpuCores
			
			if contextSwitches > csThreshold {
				contextSwitchesThresholdBreached.Set(1)
			} else {
				contextSwitchesThresholdBreached.Set(0)
			}
		}
		
		// Collect and update page faults metric
		pageFaults, err := getPageFaults()
		if err == nil {
			pageFaultsGauge.Set(float64(pageFaults))
			
			// Set page faults threshold breach metric
			if pageFaults > AppConfig.ThresholdPageFaultRate {
				pageFaultsThresholdBreached.Set(1)
			} else {
				pageFaultsThresholdBreached.Set(0)
			}
		}
		
		// Sleep for the sample interval before collecting metrics again
		time.Sleep(time.Duration(AppConfig.SampleInterval) * time.Second)
	}
}

func main() {
	// Register Prometheus metrics
	prometheus.MustRegister(cpuUsageGauge)
	prometheus.MustRegister(loadAvgGauge)
	prometheus.MustRegister(memUsedGauge)
	prometheus.MustRegister(memAvailableGauge)
	prometheus.MustRegister(diskUtilGauges)
	prometheus.MustRegister(networkRxGauges)
	prometheus.MustRegister(networkTxGauges)
	prometheus.MustRegister(contextSwitchesGauge)
	prometheus.MustRegister(pageFaultsGauge)
	
	// Register threshold breach metrics
	prometheus.MustRegister(cpuThresholdBreached)
	prometheus.MustRegister(loadThresholdBreached)
	prometheus.MustRegister(memUsageThresholdBreached)
	prometheus.MustRegister(memAvailableThresholdBreached)
	prometheus.MustRegister(diskUtilThresholdBreached)
	prometheus.MustRegister(networkRxThresholdBreached)
	prometheus.MustRegister(networkTxThresholdBreached)
	prometheus.MustRegister(contextSwitchesThresholdBreached)
	prometheus.MustRegister(pageFaultsThresholdBreached)
	
	// Start metrics collection in a goroutine
	go collectMetricsLoop()
	
	// Print startup information
	cpuCores := getCPUCores()
	totalMemMB, err := getTotalMemoryMB()
	if err != nil {
		fmt.Printf("Error getting total memory: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("=== System Monitoring Service Started ===")
	fmt.Printf("CPU cores: %d\n", cpuCores)
	fmt.Printf("Total Memory: %dMB\n", totalMemMB)
	fmt.Printf("Sample interval: %ds, Samples: %d\n", AppConfig.SampleInterval, AppConfig.SampleCount)
	fmt.Println("Metrics are available at http://localhost:9095/metrics")
	fmt.Println("Press Ctrl+C to stop the service")
	
	// Start HTTP server for Prometheus metrics (blocking call)
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Starting Prometheus metrics server on :9095")
	if err := http.ListenAndServe(":9095", nil); err != nil {
		fmt.Printf("Error starting Prometheus HTTP server: %v\n", err)
		os.Exit(1)
	}
}