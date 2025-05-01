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

// GetNetworkErrors returns network interface errors (in and out) per interface
func (dc *DarwinCollector) GetNetworkErrors() (map[string][2]int64, error) {
	cmd := exec.Command("netstat", "-ib")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(string(output), "\n")
	interfaces := make(map[string][2]int64)
	
	for i, line := range lines {
		if i > 0 && len(line) > 0 { // Skip header
			fields := strings.Fields(line)
			if len(fields) >= 8 {
				ifaceName := fields[0]
				// Skip loopback interface
				if ifaceName == "lo0" {
					continue
				}
				
				ierrors, err1 := strconv.ParseInt(fields[7], 10, 64) // Input errors
				oerrors := int64(0)
				
				// Output errors are not directly provided in netstat -ib
				// Use netstat -i for output errors
				if err1 == nil {
					interfaces[ifaceName] = [2]int64{ierrors, oerrors}
				}
			}
		}
	}
	
	// Get output errors separately using netstat -i
	cmd = exec.Command("netstat", "-i")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return interfaces, nil // Return what we have so far
	}
	
	lines = strings.Split(string(output), "\n")
	
	for i, line := range lines {
		if i > 0 && len(line) > 0 { // Skip header
			fields := strings.Fields(line)
			if len(fields) >= 11 { // netstat -i has a different format
				ifaceName := fields[0]
				// Skip loopback and entries with <Link>
				if ifaceName == "lo0" || strings.Contains(line, "<Link>") {
					continue
				}
				
				if errors, exists := interfaces[ifaceName]; exists {
					oerrors, err := strconv.ParseInt(fields[7], 10, 64) // Output errors
					if err == nil {
						interfaces[ifaceName] = [2]int64{errors[0], oerrors}
					}
				}
			}
		}
	}
	
	return interfaces, nil
}

// GetNetworkDropped returns dropped packets (in and out) per interface
func (dc *DarwinCollector) GetNetworkDropped() (map[string][2]int64, error) {
	cmd := exec.Command("netstat", "-ib")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(string(output), "\n")
	interfaces := make(map[string][2]int64)
	
	for i, line := range lines {
		if i > 0 && len(line) > 0 { // Skip header
			fields := strings.Fields(line)
			if len(fields) >= 11 { // Make sure we have enough fields
				ifaceName := fields[0]
				// Skip loopback interface
				if ifaceName == "lo0" {
					continue
				}
				
				// On macOS, netstat -ib shows dropped packets in column 8 for input
				idrops, err1 := strconv.ParseInt(fields[8], 10, 64) // Input drops
				odrops := int64(0) // macOS doesn't show output drops directly
				
				if err1 == nil {
					interfaces[ifaceName] = [2]int64{idrops, odrops}
				}
			}
		}
	}
	
	return interfaces, nil
}

// GetNetworkLatency measures latency to common destinations
func (dc *DarwinCollector) GetNetworkLatency() (map[string]float64, error) {
	destinations := []string{"8.8.8.8", "1.1.1.1", "github.com", "google.com"}
	results := make(map[string]float64)
	
	for _, dest := range destinations {
		// Use ping with 3 packets and timeout after 2 seconds
		cmd := exec.Command("ping", "-c", "3", "-W", "2000", dest)
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			results[dest] = -1 // Mark as failed
			continue
		}
		
		// Parse ping output for round-trip time
		outputStr := string(output)
		latency := extractPingLatency(outputStr)
		results[dest] = latency
	}
	
	return results, nil
}

// GetPacketLoss calculates packet loss percentage to common destinations
func (dc *DarwinCollector) GetPacketLoss() (map[string]float64, error) {
	destinations := []string{"8.8.8.8", "1.1.1.1", "github.com", "google.com"}
	results := make(map[string]float64)
	
	for _, dest := range destinations {
		// Use ping with 5 packets
		cmd := exec.Command("ping", "-c", "5", "-W", "2000", dest)
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			results[dest] = 100.0 // 100% packet loss on error
			continue
		}
		
		// Parse ping output for packet loss
		outputStr := string(output)
		packetLoss := extractPacketLoss(outputStr)
		results[dest] = packetLoss
	}
	
	return results, nil
}

// Helper function to extract average latency from ping output
func extractPingLatency(pingOutput string) float64 {
	// Look for the statistics line with avg=
	lines := strings.Split(pingOutput, "\n")
	for _, line := range lines {
		if strings.Contains(line, "round-trip") || strings.Contains(line, "rtt") {
			// Format: round-trip min/avg/max/stddev = 21.752/35.286/56.280/14.092 ms
			parts := strings.Split(line, "=")
			if len(parts) < 2 {
				return -1
			}
			
			stats := strings.Split(parts[1], "/")
			if len(stats) < 4 {
				return -1
			}
			
			avg, err := strconv.ParseFloat(strings.TrimSpace(stats[1]), 64)
			if err != nil {
				return -1
			}
			
			return avg // Return average latency in ms
		}
	}
	
	return -1 // Could not parse
}

// Helper function to extract packet loss percentage from ping output
func extractPacketLoss(pingOutput string) float64 {
	// Look for the packet loss line
	lines := strings.Split(pingOutput, "\n")
	for _, line := range lines {
		if strings.Contains(line, "packet loss") {
			// Format: "5 packets transmitted, 5 received, 0.0% packet loss"
			parts := strings.Split(line, ",")
			for _, part := range parts {
				if strings.Contains(part, "packet loss") {
					lossStr := strings.TrimSpace(strings.Split(part, "%")[0])
					loss, err := strconv.ParseFloat(lossStr, 64)
					if err == nil {
						return loss
					}
					return 100.0 // Default to 100% on parsing error
				}
			}
		}
	}
	
	return 100.0 // Default to 100% if we couldn't parse
}