# ObservT

[![License](https://img.shields.io/github/license/chowmean/observeT)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/chowmean/observeT)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/chowmean/observeT)](https://github.com/chowmean/observeT/releases)
[![Build Status](https://img.shields.io/github/workflow/status/chowmean/observeT/build)](https://github.com/chowmean/observeT/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/chowmean/observeT)](https://goreportcard.com/report/github.com/chowmean/observeT)
[![Contributors](https://img.shields.io/github/contributors/chowmean/observeT)](https://github.com/chowmean/observeT/graphs/contributors)

A lightweight system monitoring tool for macOS that collects system metrics, exposes them via Prometheus, and visualizes them with Grafana.

**Current Version: 0.0.2**

## Features

- Real-time monitoring of system resources:
  - CPU usage 
  - Load average
  - Memory usage and availability
  - Disk utilization
  - Network bandwidth (RX/TX)
  - Context switches
  - Page faults
- Selective metrics collection (enable/disable specific metrics)
- Threshold-based alerting (configurable thresholds)
- Prometheus metrics endpoint
- Pre-configured Grafana dashboard
- Memory stress testing tool

## Prerequisites

- Go 1.18+ 
- Prometheus
- Grafana

## Installation

1. Clone this repository:
   ```
   git clone https://github.com/yourusername/observT.git
   cd observT
   ```

2. Build the monitoring tool:
   ```
   go build -o observT main.go config.go
   ```


## Configuration

Configuration is managed in `config.go`. The default configuration includes:

| Setting | Default Value | Description |
|---------|---------------|-------------|
| ThresholdCpuUsage | 90.0% | CPU usage threshold |
| ThresholdLoadMultiplier | 2.0 | Load average threshold = CPU_CORES * multiplier |
| ThresholdMemUsage | 95.0% | Memory usage threshold |
| ThresholdMemAvailablePercent | 5.0% | Available memory threshold (critical if below) |
| ThresholdDiskUtil | 90.0% | Disk utilization threshold |
| ThresholdNetUtil | 70.0% | Network utilization threshold |
| ThresholdContextSwitchPerCore | 10000 | Context switches per second per core |
| ThresholdPageFaultRate | 500 | Page faults per second |
| SampleInterval | 1 | Sampling interval in seconds |
| SampleCount | 3 | Number of samples to collect |

You can modify these settings directly in `config.go` before building the application.

## Running ObservT

```
./observT
```

By default, ObservT exposes metrics on http://localhost:9095/metrics.

## Command-line Flags

ObservT supports the following command-line flags to customize its behavior:

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | "config.yaml" | Path to YAML configuration file |
| `-cpu` | true | Enable/disable CPU metrics collection |
| `-load` | true | Enable/disable load average metrics collection |
| `-memory` | true | Enable/disable memory metrics collection |
| `-disk` | true | Enable/disable disk utilization metrics collection |
| `-network` | true | Enable/disable network bandwidth metrics collection |
| `-ctx-switch` | true | Enable/disable context switch metrics collection |
| `-page-fault` | true | Enable/disable page fault metrics collection |
| `-all` | true | Master switch to enable/disable all metrics |

Examples:

```
# Collect only CPU and memory metrics
./observT -all=false -cpu=true -memory=true

# Collect all metrics except context switches
./observT -ctx-switch=false 

# Run with all default metrics (everything enabled)
./observT

# Use a different configuration file
./observT -config=/path/to/custom-config.yaml
```

## Setting up Prometheus

1. Navigate to the prometheus directory:
   ```
   cd prometheus
   ```

2. Update the prometheus.yml file if needed to point to your metrics endpoint (default is configured)

3. Start Prometheus with the configuration:
   ```
   prometheus --config.file=prometheus.yml
   ```

## Setting up Grafana

1. Start Grafana:
   ```
   brew services start grafana
   ```
   
2. Access Grafana at http://localhost:3000 (default credentials: admin/admin)

3. Add Prometheus as a data source:
   - URL: http://localhost:9090
   - Access: Browser

4. Import the dashboard:
   - Navigate to Dashboards > Import
   - Upload the JSON file from grafana/system_monitor_dashboard.json

## Metrics

The system exposes the following Prometheus metrics:

### Regular Metrics
- `system_cpu_usage_percent` - Current CPU usage in percent
- `system_load_average_15m` - 15-minute load average
- `system_memory_used_percent` - Memory usage in percent
- `system_memory_available_percent` - Available memory in percent
- `system_disk_utilization_percent{disk="..."}` - Disk utilization in percent per disk
- `system_network_rx_mbps{interface="..."}` - Network receive bandwidth in Mbps per interface
- `system_network_tx_mbps{interface="..."}` - Network transmit bandwidth in Mbps per interface
- `system_context_switches_per_second` - Context switches per second
- `system_page_faults_per_second` - Page faults per second

### Threshold Breach Metrics
These metrics emit 1 when a threshold is breached, 0 otherwise:

- `system_cpu_threshold_breached` - CPU usage threshold breach
- `system_load_threshold_breached` - Load average threshold breach
- `system_memory_usage_threshold_breached` - Memory usage threshold breach
- `system_memory_available_threshold_breached` - Available memory threshold breach
- `system_disk_threshold_breached{disk="..."}` - Disk utilization threshold breach
- `system_network_rx_threshold_breached{interface="..."}` - Network receive threshold breach
- `system_network_tx_threshold_breached{interface="..."}` - Network transmit threshold breach
- `system_context_switches_threshold_breached` - Context switches threshold breach
- `system_page_faults_threshold_breached` - Page faults threshold breach

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.