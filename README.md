# System Monitor

A lightweight system monitoring tool for macOS that collects system metrics, exposes them via Prometheus, and visualizes them with Grafana.

## Features

- Real-time monitoring of system resources:
  - CPU usage 
  - Load average
  - Memory usage and availability
  - Disk utilization
  - Network bandwidth (RX/TX)
  - Context switches
  - Page faults
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
   git clone https://github.com/yourusername/system-monitor.git
   cd system-monitor
   ```

2. Build the monitoring tool:
   ```
   go build -o system-monitor main.go config.go
   ```

3. Build the memory stress test tool (optional):
   ```
   go build -o memory-stress memory-stress.go
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

## Running the System Monitor

```
./system-monitor
```

By default, the system monitor exposes metrics on http://localhost:9095/metrics.

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

## Testing Threshold Breaches

The repository includes a memory stress testing tool to help you test threshold breach alerts:

```
./memory-stress --help

Usage:
  ./memory-stress [flags]

Flags:
  --duration int     Duration in seconds to run the memory stress test (default 60)
  --interval int     Interval in seconds between memory allocations (default 5)
  --percent int      Target memory usage percentage (default 90)
```

Example:
```
./memory-stress --percent=95 --duration=120 --interval=2
```

This will gradually allocate memory until your system reaches 95% memory usage, maintain it for 120 seconds, and allocate memory in chunks every 2 seconds.

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