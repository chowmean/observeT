package sysinfo

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// LinuxCollector implements MetricsCollector for Linux
type LinuxCollector struct {
	sampleInterval int
}

// Create a new Linux collector
func newLinuxCollector(sampleInterval int) *LinuxCollector {
	return &LinuxCollector{
		sampleInterval: sampleInterval,
	}
}

// GetTotalMemoryMB returns the total physical memory in MB
func (lc *LinuxCollector) GetTotalMemoryMB() (int, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return 0, fmt.Errorf("unexpected format in /proc/meminfo")
			}

			memKB, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0, err
			}

			// Convert KB to MB
			return int(memKB / 1024), nil
		}
	}

	return 0, fmt.Errorf("could not find MemTotal in /proc/meminfo")
}

// GetCPUUsage returns the current CPU usage percentage
func (lc *LinuxCollector) GetCPUUsage() (float64, error) {
	// Read /proc/stat two times with a delay to calculate CPU usage
	cpu1, err := readProcStat()
	if err != nil {
		return 0, err
	}

	time.Sleep(time.Duration(lc.sampleInterval) * time.Second)

	cpu2, err := readProcStat()
	if err != nil {
		return 0, err
	}

	// Calculate CPU usage
	idle1 := cpu1["idle"] + cpu1["iowait"]
	idle2 := cpu2["idle"] + cpu2["iowait"]

	total1 := 0
	total2 := 0
	for _, v := range cpu1 {
		total1 += v
	}
	for _, v := range cpu2 {
		total2 += v
	}

	// Calculate the difference
	idleDiff := idle2 - idle1
	totalDiff := total2 - total1

	if totalDiff == 0 {
		return 0, fmt.Errorf("no CPU activity detected")
	}

	// Calculate CPU usage as (1 - idle/total) * 100
	cpuUsage := (1.0 - float64(idleDiff)/float64(totalDiff)) * 100.0
	return cpuUsage, nil
}

// Helper function to read CPU stats from /proc/stat
func readProcStat() (map[string]int, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	line := scanner.Text()

	if !strings.HasPrefix(line, "cpu ") {
		return nil, fmt.Errorf("unexpected format in /proc/stat")
	}

	fields := strings.Fields(line)
	if len(fields) < 8 {
		return nil, fmt.Errorf("not enough fields in /proc/stat")
	}

	result := make(map[string]int)
	names := []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq", "steal"}

	for i, name := range names {
		value, err := strconv.Atoi(fields[i+1])
		if err != nil {
			return nil, err
		}
		result[name] = value
	}

	return result, nil
}

// GetLoadAverage returns the 15-minute load average
func (lc *LinuxCollector) GetLoadAverage() (float64, error) {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	line := scanner.Text()

	fields := strings.Fields(line)
	if len(fields) < 3 {
		return 0, fmt.Errorf("unexpected format in /proc/loadavg")
	}

	// Get the 15-minute load average (third number)
	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return 0, err
	}

	return load15, nil
}

// GetMemoryUsage returns memory usage percentages (used and available)
func (lc *LinuxCollector) GetMemoryUsage() (float64, float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	memInfo := make(map[string]int64)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Remove colon from key
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}

		memInfo[key] = value
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	// Check if we have all the values we need
	requiredKeys := []string{"MemTotal", "MemFree", "Buffers", "Cached", "SReclaimable"}
	for _, key := range requiredKeys {
		if _, ok := memInfo[key]; !ok {
			return 0, 0, fmt.Errorf("missing key in /proc/meminfo: %s", key)
		}
	}

	// Calculate memory usage
	total := memInfo["MemTotal"]
	free := memInfo["MemFree"]
	buffers := memInfo["Buffers"]
	cached := memInfo["Cached"]
	sreclaimable := memInfo["SReclaimable"]

	// Available memory is free + buffers + cached + SReclaimable
	available := free + buffers + cached + sreclaimable
	used := total - available

	usedPercent := float64(used) / float64(total) * 100
	availablePercent := float64(available) / float64(total) * 100

	return usedPercent, availablePercent, nil
}

// GetDiskUtilization returns disk utilization percentages by device
func (lc *LinuxCollector) GetDiskUtilization() (map[string]float64, error) {
	// Use iostat to get disk utilization
	cmd := exec.Command("iostat", "-dx", "2", "1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If iostat is not available, try to use a simple fallback based on df
		return lc.getDiskUtilizationFallback()
	}

	lines := strings.Split(string(output), "\n")
	disks := make(map[string]float64)

	// Skip header lines and process disk stats
	startParsing := false
	for _, line := range lines {
		if strings.Contains(line, "Device") && strings.Contains(line, "util") {
			startParsing = true
			continue
		}

		if startParsing && len(line) > 0 {
			fields := strings.Fields(line)
			if len(fields) >= 14 {
				diskName := fields[0]
				
				// The last field is %util
				util, err := strconv.ParseFloat(fields[13], 64)
				if err == nil {
					disks[diskName] = util
				}
			}
		}
	}

	if len(disks) == 0 {
		return lc.getDiskUtilizationFallback()
	}

	return disks, nil
}

// Fallback method for disk utilization when iostat is not available
func (lc *LinuxCollector) getDiskUtilizationFallback() (map[string]float64, error) {
	cmd := exec.Command("df", "-h")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	disks := make(map[string]float64)

	// Skip header line
	for i, line := range lines {
		if i == 0 || len(line) == 0 {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			// Get device name and usage percentage
			device := fields[0]
			usageStr := fields[4]
			usageStr = strings.TrimSuffix(usageStr, "%")
			
			usage, err := strconv.ParseFloat(usageStr, 64)
			if err == nil {
				disks[device] = usage
			}
		}
	}

	return disks, nil
}

// GetNetworkBandwidth returns network bandwidth in Mbps by interface
func (lc *LinuxCollector) GetNetworkBandwidth() (map[string][2]float64, error) {
	// Read network stats from /proc/net/dev
	stats1, err := readNetworkStats()
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Duration(lc.sampleInterval) * time.Second)

	stats2, err := readNetworkStats()
	if err != nil {
		return nil, err
	}

	// Calculate bandwidth for each interface
	interfaces := make(map[string][2]float64)
	for iface, stat2 := range stats2 {
		if stat1, ok := stats1[iface]; ok {
			// Skip loopback interface
			if iface == "lo" {
				continue
			}

			rxRate := float64(stat2[0]-stat1[0]) / float64(lc.sampleInterval) * 8 / 1000000 // Convert to Mbps
			txRate := float64(stat2[1]-stat1[1]) / float64(lc.sampleInterval) * 8 / 1000000 // Convert to Mbps
			interfaces[iface] = [2]float64{rxRate, txRate}
		}
	}

	return interfaces, nil
}

// Helper function to read network interface statistics
func readNetworkStats() (map[string][2]int64, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	stats := make(map[string][2]int64)

	// Skip first two header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 17 {
			continue
		}

		// Interface name is field[0] with colon
		iface := strings.TrimSuffix(fields[0], ":")
		rxBytes, err1 := strconv.ParseInt(fields[1], 10, 64)
		txBytes, err2 := strconv.ParseInt(fields[9], 10, 64)

		if err1 == nil && err2 == nil {
			stats[iface] = [2]int64{rxBytes, txBytes}
		}
	}

	return stats, nil
}

// GetContextSwitches returns the number of context switches per second
func (lc *LinuxCollector) GetContextSwitches() (int, error) {
	count1, err := readContextSwitchCount()
	if err != nil {
		return 0, err
	}

	time.Sleep(time.Duration(lc.sampleInterval) * time.Second)

	count2, err := readContextSwitchCount()
	if err != nil {
		return 0, err
	}

	return count2 - count1, nil
}

// Helper function to read context switch count
func readContextSwitchCount() (int, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ctxt ") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0, fmt.Errorf("unexpected format in /proc/stat for context switches")
			}

			count, err := strconv.Atoi(fields[1])
			if err != nil {
				return 0, err
			}

			return count, nil
		}
	}

	return 0, fmt.Errorf("could not find context switch count in /proc/stat")
}

// GetPageFaults returns the number of page faults per second
func (lc *LinuxCollector) GetPageFaults() (int, error) {
	// Try to use vmstat first
	cmd := exec.Command("vmstat", "-s")
	output, err := cmd.CombinedOutput()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "page faults") {
				fields := strings.Fields(line)
				if len(fields) >= 1 {
					count, err := strconv.Atoi(fields[0])
					if err == nil {
						// Wait for a second interval and get the count again
						time.Sleep(time.Duration(lc.sampleInterval) * time.Second)
						
						cmd = exec.Command("vmstat", "-s")
						output, err = cmd.CombinedOutput()
						if err != nil {
							return 0, err
						}
						
						lines = strings.Split(string(output), "\n")
						for _, line := range lines {
							if strings.Contains(line, "page faults") {
								fields = strings.Fields(line)
								if len(fields) >= 1 {
									count2, err := strconv.Atoi(fields[0])
									if err == nil {
										return count2 - count, nil
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	// Fallback to /proc/vmstat
	return lc.getPageFaultsFallback()
}

// Fallback method for page faults when vmstat is not available
func (lc *LinuxCollector) getPageFaultsFallback() (int, error) {
	count1, err := readPageFaultCount()
	if err != nil {
		return 0, err
	}

	time.Sleep(time.Duration(lc.sampleInterval) * time.Second)

	count2, err := readPageFaultCount()
	if err != nil {
		return 0, err
	}

	return count2 - count1, nil
}

// Helper function to read page fault count from /proc/vmstat
func readPageFaultCount() (int, error) {
	file, err := os.Open("/proc/vmstat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	minorFaults := 0
	majorFaults := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "pgfault ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				count, err := strconv.Atoi(fields[1])
				if err == nil {
					minorFaults = count
				}
			}
		} else if strings.HasPrefix(line, "pgmajfault ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				count, err := strconv.Atoi(fields[1])
				if err == nil {
					majorFaults = count
				}
			}
		}
	}

	// Return the sum of minor and major faults
	return minorFaults + majorFaults, nil
}