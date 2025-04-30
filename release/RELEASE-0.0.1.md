# ObservT v0.0.1 Release

Initial release of ObservT, a lightweight system monitoring tool for macOS.

## Features

- Real-time monitoring of system resources:
  - CPU usage 
  - Load average
  - Memory usage and availability
  - Disk utilization
  - Network bandwidth (RX/TX)
  - Context switches
  - Page faults
- Threshold-based alerting with binary metrics (1 = threshold breached, 0 = normal)
- Prometheus metrics endpoint
- Pre-configured Grafana dashboard
- Memory stress testing tool

## Installation

Simply download the binaries for your platform or build from source:

```
go build -o observT main.go config.go
go build -o memory-stress memory-stress.go
```

## Running

```
./observT
```

The monitoring service will start and expose metrics on http://localhost:9095/metrics.

For more detailed instructions, please refer to the README.md file.

## Changes in v0.0.1

- Initial system metrics collection (CPU, memory, disk, network, context switches, page faults)
- Prometheus metrics endpoint on port 9095
- Threshold configuration in config.go
- Threshold breach binary metrics (1/0) for all collected metrics
- Custom Grafana dashboard
- Memory stress testing tool