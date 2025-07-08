# Tools Setup Guide

This guide helps you set up all the necessary tools for the Go Performance Mastery Course.

## Go Tools

### Required Go Version
- Go 1.24+ (latest stable recommended)
- Verify installation: `go version`

### Built-in Performance Tools
```bash
# CPU profiling
go tool pprof

# Memory profiling
go tool pprof -alloc_space
go tool pprof -inuse_space

# Execution tracing
go tool trace

# Race detection
go run -race

# Benchmarking
go test -bench=.
go test -benchmem

# Assembly inspection
go tool compile -S
go tool objdump
```

## Linux Performance Tools

### System Monitoring
```bash
# Install essential tools (Ubuntu/Debian)
sudo apt update
sudo apt install -y htop iotop sysstat perf-tools-unstable

# CPU monitoring
htop
top
vmstat 1
mpstat 1

# Memory monitoring
free -h
vmstat -s
/proc/meminfo

# I/O monitoring
iotop
iostat 1
lsof
```

### Network Tools
```bash
# Network monitoring
sudo apt install -y tcpdump wireshark-common netstat-nat

# Network analysis
tcpdump -i any
ss -tuln
netstat -i
iftop
```

### Advanced Profiling
```bash
# Install perf
sudo apt install -y linux-tools-common linux-tools-generic

# BPF tools (Ubuntu 20.04+)
sudo apt install -y bpfcc-tools

# FlameGraph
git clone https://github.com/brendangregg/FlameGraph.git
export PATH=$PATH:$(pwd)/FlameGraph
```

## Load Testing Tools

### HTTP Load Testing
```bash
# wrk - modern HTTP benchmarking tool
sudo apt install -y wrk

# hey - Go-based load testing
go install github.com/rakyll/hey@latest

# Apache Bench
sudo apt install -y apache2-utils

# k6 - JavaScript-based load testing
sudo apt install -y k6
```

### Database Tools
```bash
# PostgreSQL tools
sudo apt install -y postgresql-client

# MySQL tools
sudo apt install -y mysql-client

# Redis tools
sudo apt install -y redis-tools
```

## Monitoring and Observability

### Prometheus & Grafana
```bash
# Prometheus
wget https://github.com/prometheus/prometheus/releases/latest

# Grafana
sudo apt install -y software-properties-common
sudo add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
sudo apt update
sudo apt install -y grafana
```

### Jaeger (Distributed Tracing)
```bash
# Jaeger all-in-one
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest
```

## Development Environment

### VS Code Extensions
- Go (official)
- Go Outliner
- Go Test Explorer
- Flame Graph Visualizer

### Useful Aliases
```bash
# Add to ~/.bashrc or ~/.zshrc
alias gob='go build'
alias gor='go run'
alias got='go test -v'
alias gob='go test -bench=. -benchmem'
alias prof='go tool pprof'
```

## Verification

Run these commands to verify your setup:

```bash
# Go tools
go version
go env

# System tools
which htop iotop perf

# Load testing
which wrk hey ab

# Network tools
which tcpdump ss netstat
```

## Troubleshooting

### Common Issues

1. **Permission denied for perf**
   ```bash
   echo 0 | sudo tee /proc/sys/kernel/perf_event_paranoid
   ```

2. **Go module issues**
   ```bash
   go clean -modcache
   go mod download
   ```

3. **Missing kernel headers**
   ```bash
   sudo apt install -y linux-headers-$(uname -r)
   ```

## Next Steps

Once all tools are installed:
1. Complete Module 01 exercises
2. Practice with provided examples
3. Set up monitoring for your applications
4. Join the Go performance community

For detailed usage of each tool, refer to the individual module documentation.