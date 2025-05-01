package sysinfo

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// WindowsCollector implements MetricsCollector for Windows
type WindowsCollector struct {
	sampleInterval int
}

// Create a new Windows collector
func newWindowsCollector(sampleInterval int) *WindowsCollector {
	return &WindowsCollector{
		sampleInterval: sampleInterval,
	}
}

// GetTotalMemoryMB returns the total physical memory in MB
func (wc *WindowsCollector) GetTotalMemoryMB() (int, error) {
	// Use PowerShell to get total physical memory
	cmd := exec.Command("powershell", "-Command", "(Get-CimInstance Win32_PhysicalMemory | Measure-Object -Property capacity -Sum).Sum / 1MB")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	memStr := strings.TrimSpace(string(output))
	totalMB, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		return 0, err
	}

	return int(totalMB), nil
}

// GetCPUUsage returns the current CPU usage percentage
func (wc *WindowsCollector) GetCPUUsage() (float64, error) {
	// Use PowerShell to get CPU usage
	cmd := exec.Command("powershell", "-Command", "Get-Counter '\\Processor(_Total)\\% Processor Time' | Select-Object -ExpandProperty CounterSamples | Select-Object -ExpandProperty CookedValue")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	cpuStr := strings.TrimSpace(string(output))
	cpuUsage, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		return 0, err
	}

	return cpuUsage, nil
}

// GetLoadAverage returns a simulated load average on Windows
// Windows doesn't have a direct equivalent of load average
// We'll use CPU queue length as an approximation
func (wc *WindowsCollector) GetLoadAverage() (float64, error) {
	// Get the processor queue length as a Windows approximation of load
	cmd := exec.Command("powershell", "-Command", "Get-Counter '\\System\\Processor Queue Length' | Select-Object -ExpandProperty CounterSamples | Select-Object -ExpandProperty CookedValue")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	queueStr := strings.TrimSpace(string(output))
	queueLen, err := strconv.ParseFloat(queueStr, 64)
	if err != nil {
		return 0, err
	}

	// Return the processor queue length (it's not exactly load average, but it's similar in concept)
	return queueLen, nil
}

// GetMemoryUsage returns memory usage percentages (used and available)
func (wc *WindowsCollector) GetMemoryUsage() (float64, float64, error) {
	// Get memory information using PowerShell
	cmd := exec.Command("powershell", "-Command", "$os = Get-Ciminstance Win32_OperatingSystem; $usedMem = $os.TotalVisibleMemorySize - $os.FreePhysicalMemory; $usedPercent = ($usedMem / $os.TotalVisibleMemorySize) * 100; $availablePercent = ($os.FreePhysicalMemory / $os.TotalVisibleMemorySize) * 100; Write-Output \"$usedPercent $availablePercent\"")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("unexpected output format from memory query")
	}

	usedPercent, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, err
	}

	availablePercent, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, err
	}

	return usedPercent, availablePercent, nil
}

// GetDiskUtilization returns disk utilization percentages by device
func (wc *WindowsCollector) GetDiskUtilization() (map[string]float64, error) {
	// Get disk utilization using PowerShell
	cmd := exec.Command("powershell", "-Command", "Get-Volume | Select-Object DriveLetter, @{Name='UsedSpace';Expression={$_.Size - $_.SizeRemaining}}, Size | ForEach-Object {$letter = $_.DriveLetter; if ($letter -ne $null) { $usedPercent = ($_.UsedSpace / $_.Size) * 100; Write-Output \"$letter $usedPercent\" }}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	disks := make(map[string]float64)

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			drive := parts[0]
			utilization, err := strconv.ParseFloat(parts[1], 64)
			if err == nil {
				disks[drive+":"] = utilization
			}
		}
	}

	return disks, nil
}

// GetNetworkBandwidth returns network bandwidth in Mbps by interface
func (wc *WindowsCollector) GetNetworkBandwidth() (map[string][2]float64, error) {
	// Get initial network stats
	rxBytes1, txBytes1, err := wc.getNetworkStats()
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Duration(wc.sampleInterval) * time.Second)

	// Get network stats after interval
	rxBytes2, txBytes2, err := wc.getNetworkStats()
	if err != nil {
		return nil, err
	}

	// Calculate bandwidth for each interface
	interfaces := make(map[string][2]float64)
	for iface, rxBytes := range rxBytes2 {
		if txBytes, ok := txBytes2[iface]; ok {
			if rxBytes1Val, ok := rxBytes1[iface]; ok {
				if txBytes1Val, ok := txBytes1[iface]; ok {
					rxRate := (float64(rxBytes) - float64(rxBytes1Val)) / float64(wc.sampleInterval) * 8 / 1000000
					txRate := (float64(txBytes) - float64(txBytes1Val)) / float64(wc.sampleInterval) * 8 / 1000000
					interfaces[iface] = [2]float64{rxRate, txRate}
				}
			}
		}
	}

	return interfaces, nil
}

// Helper function to get network stats
func (wc *WindowsCollector) getNetworkStats() (map[string]int64, map[string]int64, error) {
	// Get network interface stats using PowerShell
	cmd := exec.Command("powershell", "-Command", "Get-NetAdapter | ForEach-Object { $name = $_.Name; $stats = Get-NetAdapterStatistics -Name $name; Write-Output \"$name,$($stats.ReceivedBytes),$($stats.SentBytes)\" }")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	rxBytes := make(map[string]int64)
	txBytes := make(map[string]int64)

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			iface := parts[0]
			rx, err1 := strconv.ParseInt(parts[1], 10, 64)
			tx, err2 := strconv.ParseInt(parts[2], 10, 64)

			if err1 == nil && err2 == nil {
				rxBytes[iface] = rx
				txBytes[iface] = tx
			}
		}
	}

	return rxBytes, txBytes, nil
}

// GetContextSwitches returns the number of context switches per second
func (wc *WindowsCollector) GetContextSwitches() (int, error) {
	// Get context switches using PowerShell
	cmd := exec.Command("powershell", "-Command", "Get-Counter '\\System\\Context Switches/sec' | Select-Object -ExpandProperty CounterSamples | Select-Object -ExpandProperty CookedValue")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	csStr := strings.TrimSpace(string(output))
	csVal, err := strconv.ParseFloat(csStr, 64)
	if err != nil {
		return 0, err
	}

	return int(csVal), nil
}

// GetPageFaults returns the number of page faults per second
func (wc *WindowsCollector) GetPageFaults() (int, error) {
	// Get page faults using PowerShell
	cmd := exec.Command("powershell", "-Command", "Get-Counter '\\Memory\\Page Faults/sec' | Select-Object -ExpandProperty CounterSamples | Select-Object -ExpandProperty CookedValue")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	pfStr := strings.TrimSpace(string(output))
	pfVal, err := strconv.ParseFloat(pfStr, 64)
	if err != nil {
		return 0, err
	}

	return int(pfVal), nil
}

// GetNetworkErrors returns network interface errors (in and out) per interface
func (wc *WindowsCollector) GetNetworkErrors() (map[string][2]int64, error) {
	// Get network errors using PowerShell
	cmd := exec.Command("powershell", "-Command", 
		"Get-NetAdapterStatistics | ForEach-Object { $name = $_.Name; Write-Output \"$name,$($_.ReceivedErrors),$($_.OutboundErrors)\" }")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	interfaces := make(map[string][2]int64)

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			iface := parts[0]
			rxErrors, err1 := strconv.ParseInt(parts[1], 10, 64)
			txErrors, err2 := strconv.ParseInt(parts[2], 10, 64)

			if err1 == nil && err2 == nil {
				interfaces[iface] = [2]int64{rxErrors, txErrors}
			}
		}
	}

	return interfaces, nil
}

// GetNetworkDropped returns dropped packets (in and out) per interface
func (wc *WindowsCollector) GetNetworkDropped() (map[string][2]int64, error) {
	// Windows doesn't directly expose dropped packet counts through standard cmdlets
	// Use a custom query to approximate this information from performance counters
	cmd := exec.Command("powershell", "-Command", 
		"Get-Counter '\\Network Interface(*)\\Packets Received Discarded' -ErrorAction SilentlyContinue | ForEach-Object { $_.CounterSamples } | ForEach-Object { $name = ($_.Path -split '\\\\Network Interface\\(|\\)\\')[1]; $value = $_.CookedValue; Write-Output \"$name,$value,0\" }")
	output, err := cmd.CombinedOutput()
	
	// Return empty if the counter isn't available
	if err != nil {
		return make(map[string][2]int64), nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	interfaces := make(map[string][2]int64)

	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			iface := parts[0]
			rxDrops, err1 := strconv.ParseInt(parts[1], 10, 64)
			txDrops := int64(0) // Windows doesn't easily expose TX drops
			
			if err1 == nil {
				interfaces[iface] = [2]int64{rxDrops, txDrops}
			}
		}
	}

	return interfaces, nil
}

// GetNetworkLatency measures latency to common destinations
func (wc *WindowsCollector) GetNetworkLatency() (map[string]float64, error) {
	destinations := []string{"8.8.8.8", "1.1.1.1", "github.com", "google.com"}
	results := make(map[string]float64)
	
	for _, dest := range destinations {
		// Use ping with 3 packets and timeout after 2 seconds
		cmd := exec.Command("powershell", "-Command", 
			fmt.Sprintf("$ping = Test-Connection -ComputerName %s -Count 3 -ErrorAction SilentlyContinue; if ($ping) { ($ping | Measure-Object -Property ResponseTime -Average).Average } else { -1 }", dest))
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			results[dest] = -1 // Mark as failed
			continue
		}
		
		latencyStr := strings.TrimSpace(string(output))
		latency, err := strconv.ParseFloat(latencyStr, 64)
		if err != nil {
			results[dest] = -1
		} else {
			results[dest] = latency
		}
	}
	
	return results, nil
}

// GetPacketLoss calculates packet loss percentage to common destinations
func (wc *WindowsCollector) GetPacketLoss() (map[string]float64, error) {
	destinations := []string{"8.8.8.8", "1.1.1.1", "github.com", "google.com"}
	results := make(map[string]float64)
	
	for _, dest := range destinations {
		// Use ping with 5 packets to calculate packet loss
		cmd := exec.Command("powershell", "-Command", 
			fmt.Sprintf("$ping = ping -n 5 %s; $loss = [regex]::Match($ping, '(\\d+)%% loss').Groups[1].Value; if ($loss) { $loss } else { 100 }", dest))
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			results[dest] = 100.0 // 100% packet loss on error
			continue
		}
		
		lossStr := strings.TrimSpace(string(output))
		loss, err := strconv.ParseFloat(lossStr, 64)
		if err != nil {
			results[dest] = 100.0
		} else {
			results[dest] = loss
		}
	}
	
	return results, nil
}