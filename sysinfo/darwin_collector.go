package sysinfo

import (
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// DarwinCollector implements MetricsCollector for macOS
type DarwinCollector struct {
	sampleInterval int
}

// Create a new Darwin (macOS) collector
func newDarwinCollector(sampleInterval int) *DarwinCollector {
	return &DarwinCollector{
		sampleInterval: sampleInterval,
	}
}

// GetTotalMemoryMB returns the total physical memory in MB
func (dc *DarwinCollector) GetTotalMemoryMB() (int, error) {
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

// GetCPUUsage returns the current CPU usage percentage
func (dc *DarwinCollector) GetCPUUsage() (float64, error) {
	cmd := exec.Command("top", "-l", "2", "-n", "0", "-s", strconv.Itoa(dc.sampleInterval))
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

// GetLoadAverage returns the 15-minute load average
func (dc *DarwinCollector) GetLoadAverage() (float64, error) {
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

// GetMemoryUsage returns memory usage percentages (used and available)
func (dc *DarwinCollector) GetMemoryUsage() (float64, float64, error) {
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

// GetDiskUtilization returns disk utilization percentages by device
func (dc *DarwinCollector) GetDiskUtilization() (map[string]float64, error) {
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

// GetNetworkBandwidth returns network bandwidth in Mbps by interface
func (dc *DarwinCollector) GetNetworkBandwidth() (map[string][2]float64, error) {
	cmd := exec.Command("netstat", "-ib")
	output1, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	
	time.Sleep(time.Duration(dc.sampleInterval) * time.Second)
	
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
			rxRate := float64(stat2[0]-stat1[0]) / float64(dc.sampleInterval) * 8 / 1000000 // Convert to Mbps
			txRate := float64(stat2[1]-stat1[1]) / float64(dc.sampleInterval) * 8 / 1000000 // Convert to Mbps
			interfaces[iface] = [2]float64{rxRate, txRate}
		}
	}
	
	return interfaces, nil
}

// GetContextSwitches returns the number of context switches per second
func (dc *DarwinCollector) GetContextSwitches() (int, error) {
	cmd1 := exec.Command("sysctl", "-n", "vm.stats.sys.v_swtch")
	output1, err := cmd1.CombinedOutput()
	if err != nil {
		return 0, err
	}
	
	cs1, err := strconv.Atoi(strings.TrimSpace(string(output1)))
	if err != nil {
		return 0, err
	}
	
	time.Sleep(time.Duration(dc.sampleInterval) * time.Second)
	
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

// GetPageFaults returns the number of page faults per second
func (dc *DarwinCollector) GetPageFaults() (int, error) {
	cmd1 := exec.Command("sysctl", "-n", "vm.stats.vm.v_fault")
	output1, err := cmd1.CombinedOutput()
	if err != nil {
		return 0, err
	}
	
	pf1, err := strconv.Atoi(strings.TrimSpace(string(output1)))
	if err != nil {
		return 0, err
	}
	
	time.Sleep(time.Duration(dc.sampleInterval) * time.Second)
	
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