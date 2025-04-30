// Package sysinfo provides information about the host system
package sysinfo

import (
	"fmt"
	"runtime"
)

// SupportedOS represents an operating system supported by the application
type SupportedOS string

// SupportedArch represents a CPU architecture supported by the application
type SupportedArch string

const (
	// OS constants
	OSDarwin SupportedOS = "darwin"
	OSLinux  SupportedOS = "linux"
	OSWindows SupportedOS = "windows"
	OSUnknown SupportedOS = "unknown"
	
	// Architecture constants
	ArchAMD64 SupportedArch = "amd64"
	ArchARM64 SupportedArch = "arm64"
	ArchX86   SupportedArch = "386"
	ArchARM   SupportedArch = "arm"
	ArchUnknown SupportedArch = "unknown"
)

// SystemInfo contains basic information about the host system
type SystemInfo struct {
	OS          SupportedOS
	Arch        SupportedArch
	IsSupported bool
	CPUCores    int
}

// GetSystemInfo returns information about the host system
func GetSystemInfo() SystemInfo {
	si := SystemInfo{
		OS:          getOS(),
		Arch:        getArch(),
		CPUCores:    runtime.NumCPU(),
	}
	
	// Check if the combination is supported
	si.IsSupported = isSupported(si.OS, si.Arch)
	
	return si
}

// getOS returns the current operating system
func getOS() SupportedOS {
	switch runtime.GOOS {
	case "darwin":
		return OSDarwin
	case "linux":
		return OSLinux
	case "windows":
		return OSWindows
	default:
		return OSUnknown
	}
}

// getArch returns the current CPU architecture
func getArch() SupportedArch {
	switch runtime.GOARCH {
	case "amd64":
		return ArchAMD64
	case "arm64":
		return ArchARM64
	case "386":
		return ArchX86
	case "arm":
		return ArchARM
	default:
		return ArchUnknown
	}
}

// isSupported checks if the OS and architecture are supported
func isSupported(os SupportedOS, arch SupportedArch) bool {
	switch os {
	case OSDarwin:
		return arch == ArchAMD64 || arch == ArchARM64
	case OSLinux:
		return arch == ArchAMD64 || arch == ArchARM64 || arch == ArchX86 || arch == ArchARM
	case OSWindows:
		return arch == ArchAMD64 || arch == ArchX86
	default:
		return false
	}
}

// String returns the string representation of the OS
func (os SupportedOS) String() string {
	switch os {
	case OSDarwin:
		return "macOS"
	case OSLinux:
		return "Linux"
	case OSWindows:
		return "Windows"
	default:
		return "Unknown OS"
	}
}

// String returns the string representation of the architecture
func (arch SupportedArch) String() string {
	switch arch {
	case ArchAMD64:
		return "x86_64"
	case ArchARM64:
		return "ARM64"
	case ArchX86:
		return "x86"
	case ArchARM:
		return "ARM"
	default:
		return "Unknown Architecture"
	}
}

// SystemDescription returns a human-readable description of the system
func (si SystemInfo) SystemDescription() string {
	return fmt.Sprintf("%s on %s (%d CPU cores)", si.OS, si.Arch, si.CPUCores)
}