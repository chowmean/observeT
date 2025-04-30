# ObservT v0.0.2 Release

This release of ObservT focuses on performance improvements and memory optimizations through better pointer management.

## Features

- All the features from v0.0.1 plus:
  - Improved memory usage through optimized pointer management
  - Enhanced collector performance for all metrics
  - Reduced memory footprint during monitoring
  - Better handling of metric collection errors
  - More efficient resource utilization

## Installation

Build from source with the optimized pointer-based collectors:

```
go build -o observT main.go config.go
```

For memory stress testing (unchanged from v0.0.1):

```
go build -o memory-stress memory-stress.go
```

## Running

Usage remains the same as v0.0.1:

```
./observT
```

The monitoring service will start and expose metrics on http://localhost:9095/metrics.

## Changes in v0.0.2

- **Pointer Optimizations**: Replaced value copies with pointer references in all metric collectors
- **Memory Efficiency**: Reduced collector memory allocations by using pointer receivers consistently
- **Metric Collection Performance**: Faster metric gathering through improved pointer handling
- **Cross-Platform Improvements**: Better pointer management across all supported platforms (macOS, Linux, Windows)
- **Error Handling**: Enhanced error handling with pass-by-reference for error reporting
- **Sample Interval Management**: More efficient timer management using pointer references

## Technical Details

The following components were optimized with improved pointer usage:

- **Collectors**: All OS-specific collectors now use consistent pointer receivers
- **Memory Usage Tracking**: Optimized memory allocation with pointers in memory stats collection
- **Network Metrics**: Improved bandwidth tracking using pointer-based data structures
- **Disk Utilization**: Enhanced performance of disk metrics through pointer optimization
- **System Metrics**: More efficient context switch and page fault monitoring with streamlined pointer usage