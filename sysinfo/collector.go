package sysinfo

// MetricsCollector defines an interface for OS-specific metric collection
type MetricsCollector interface {
	// Basic system information
	GetTotalMemoryMB() (int, error)
	
	// CPU metrics
	GetCPUUsage() (float64, error)
	GetLoadAverage() (float64, error) 
	
	// Memory metrics
	GetMemoryUsage() (float64, float64, error) // Returns used percent and available percent
	
	// Disk metrics
	GetDiskUtilization() (map[string]float64, error)
	
	// Network metrics
	GetNetworkBandwidth() (map[string][2]float64, error) // Returns map[interface][rx, tx]
	
	// System metrics
	GetContextSwitches() (int, error)
	GetPageFaults() (int, error)
}

// NewMetricsCollector creates an appropriate metrics collector for the current OS
func NewMetricsCollector(sampleInterval int) (MetricsCollector, error) {
	sysInfo := GetSystemInfo()
	
	switch sysInfo.OS {
	case OSDarwin:
		return newDarwinCollector(sampleInterval), nil
	case OSLinux:
		return newLinuxCollector(sampleInterval), nil
	case OSWindows:
		return newWindowsCollector(sampleInterval), nil
	default:
		// Fallback to Darwin collector since that's our original implementation
		return newDarwinCollector(sampleInterval), nil
	}
}