# Linux Performance Tools Cheat Sheet

Quick reference for essential Linux performance monitoring and debugging tools.

## System Overview

### htop / top
```bash
# Interactive process viewer
htop
top

# Sort by CPU usage
top -o %CPU

# Sort by memory usage
top -o %MEM

# Show specific user processes
top -u username

# Batch mode (non-interactive)
top -b -n 1
```

### System Load
```bash
# Load averages
uptime
w

# Detailed system stats
vmstat 1        # Every second
vmstat 5 10     # Every 5 seconds, 10 times

# CPU statistics
mpstat 1        # Per-CPU stats
mpstat -P ALL 1 # All CPUs
```

## CPU Monitoring

### CPU Usage
```bash
# Real-time CPU usage
sar -u 1        # CPU utilization
sar -u 1 10     # 10 samples

# CPU usage by process
ps aux --sort=-%cpu | head -10

# CPU usage over time
sar -u -f /var/log/sysstat/saXX  # Historical data
```

### CPU Performance
```bash
# CPU frequency
cpufreq-info
cat /proc/cpuinfo | grep MHz

# CPU topology
lscpu
nproc

# CPU cache info
lscpu | grep cache
```

## Memory Monitoring

### Memory Usage
```bash
# Memory overview
free -h         # Human readable
free -m         # In MB
free -s 1       # Every second

# Detailed memory info
cat /proc/meminfo

# Memory usage by process
ps aux --sort=-%mem | head -10
```

### Memory Analysis
```bash
# Virtual memory statistics
vmstat 1
vmstat -s       # Summary

# Memory mapping
pmap <pid>      # Process memory map
pmap -x <pid>   # Extended info

# Swap usage
swapon -s
cat /proc/swaps
```

## I/O Monitoring

### Disk I/O
```bash
# I/O statistics
iostat 1        # Every second
iostat -x 1     # Extended stats
iostat -d 1     # Device stats only

# I/O by process
iotop           # Interactive
iotop -o        # Only active processes
iotop -a        # Accumulated I/O
```

### Disk Usage
```bash
# Disk space
df -h           # Human readable
df -i           # Inode usage

# Directory sizes
du -sh *        # Current directory
du -sh /path/*  # Specific path

# Largest files
find / -type f -size +100M 2>/dev/null
```

## Network Monitoring

### Network Statistics
```bash
# Network interfaces
ip addr show
ifconfig

# Network statistics
ss -tuln        # Listening ports
netstat -tuln   # Alternative
ss -i           # Interface stats
```

### Network Traffic
```bash
# Real-time network usage
iftop           # Interactive bandwidth usage
nload           # Network load
bmon            # Bandwidth monitor

# Network connections
ss -tupln       # All connections
netstat -tupln  # Alternative
lsof -i         # Files/processes using network
```

### Packet Analysis
```bash
# Capture packets
tcpdump -i any                    # All interfaces
tcpdump -i eth0 port 80          # HTTP traffic
tcpdump -i eth0 host 192.168.1.1 # Specific host

# Save to file
tcpdump -i eth0 -w capture.pcap

# Read from file
tcpdump -r capture.pcap
```

## Process Monitoring

### Process Information
```bash
# Process tree
pstree
pstree -p       # With PIDs

# Process details
ps aux
ps -ef
ps -eLf         # Include threads

# Process by name
pgrep <name>
pkill <name>
```

### Process Resources
```bash
# Open files
lsof -p <pid>   # Files opened by process
lsof /path/file # Processes using file
lsof -i :80     # Processes using port 80

# Process limits
ulimit -a       # Current limits
cat /proc/<pid>/limits  # Process limits
```

## Advanced Profiling

### perf Tool
```bash
# CPU profiling
perf top                    # Real-time profiling
perf top -p <pid>          # Specific process
perf record ./program      # Record profile
perf report                # View recorded profile

# System-wide profiling
perf record -g -a sleep 10 # 10 seconds, all CPUs
perf report -g             # Call graph

# Memory profiling
perf record -e cache-misses ./program
perf record -e page-faults ./program
```

### strace
```bash
# System call tracing
strace ./program           # Trace program
strace -p <pid>           # Attach to running process
strace -c ./program       # Count system calls
strace -e open ./program  # Trace specific calls
strace -f ./program       # Follow forks
```

### ltrace
```bash
# Library call tracing
ltrace ./program          # Trace library calls
ltrace -p <pid>          # Attach to process
ltrace -c ./program      # Count calls
ltrace -e malloc ./program # Trace specific functions
```

## File System Monitoring

### File System Usage
```bash
# File system statistics
df -h           # Disk usage
df -i           # Inode usage

# File system types
mount | column -t
cat /proc/mounts
```

### File Access
```bash
# File access monitoring
inotifywait -m /path/to/watch  # Monitor file changes
inotifywatch /path/to/watch    # Statistics

# Find recently accessed files
find /path -atime -1           # Accessed in last day
find /path -mtime -1           # Modified in last day
```

## System Information

### Hardware Info
```bash
# CPU information
lscpu
cat /proc/cpuinfo

# Memory information
lsmem
cat /proc/meminfo

# Hardware details
lshw            # Detailed hardware
lshw -short     # Summary
dmidecode       # DMI/SMBIOS info
```

### Kernel Information
```bash
# Kernel version
uname -a
cat /proc/version

# Kernel modules
lsmod
modinfo <module>

# Kernel parameters
sysctl -a
cat /proc/sys/...
```

## Log Analysis

### System Logs
```bash
# System journal
journalctl -f           # Follow logs
journalctl -u service   # Specific service
journalctl --since "1 hour ago"

# Traditional logs
tail -f /var/log/syslog
tail -f /var/log/messages
```

### Log Analysis
```bash
# Search logs
grep "error" /var/log/syslog
zgrep "error" /var/log/syslog.*.gz

# Log statistics
awk '{print $1}' /var/log/access.log | sort | uniq -c
```

## Performance Tuning

### CPU Tuning
```bash
# CPU governor
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
echo performance > /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# CPU affinity
taskset -c 0,1 ./program  # Run on CPUs 0,1
taskset -p <pid>          # Show current affinity
```

### Memory Tuning
```bash
# Memory parameters
echo 1 > /proc/sys/vm/drop_caches     # Drop caches
echo 3 > /proc/sys/vm/drop_caches     # Drop all caches

# Swap settings
sysctl vm.swappiness=10               # Reduce swapping
sysctl vm.vfs_cache_pressure=50       # Cache pressure
```

### I/O Tuning
```bash
# I/O scheduler
cat /sys/block/sda/queue/scheduler
echo deadline > /sys/block/sda/queue/scheduler

# I/O priority
ionice -c 1 -n 4 ./program           # Real-time class
ionice -p <pid>                       # Show current priority
```

## Quick Diagnostics

### System Health Check
```bash
# Quick system overview
uptime && free -h && df -h

# Top processes by CPU
ps aux --sort=-%cpu | head -5

# Top processes by memory
ps aux --sort=-%mem | head -5

# Network connections
ss -tuln | wc -l

# Disk I/O
iostat -x 1 3
```

### Performance Baseline
```bash
# Collect baseline metrics
{
  echo "=== System Info ==="
  uname -a
  lscpu | grep -E '^CPU\(s\)|^Model name'
  free -h
  
  echo "=== Load Average ==="
  uptime
  
  echo "=== CPU Usage ==="
  mpstat 1 3
  
  echo "=== Memory Usage ==="
  vmstat 1 3
  
  echo "=== Disk I/O ==="
  iostat -x 1 3
  
  echo "=== Network ==="
  ss -s
} > baseline.txt
```

## Common Issues

### High CPU Usage
```bash
# Find CPU-intensive processes
top -o %CPU
ps aux --sort=-%cpu | head -10

# Profile with perf
perf top
perf record -g ./program
```

### Memory Issues
```bash
# Check for memory leaks
valgrind --leak-check=full ./program

# Monitor memory growth
watch -n 1 'ps aux --sort=-%mem | head -10'

# Check swap usage
swapon -s
cat /proc/swaps
```

### I/O Bottlenecks
```bash
# Find I/O-intensive processes
iotop -o

# Check disk utilization
iostat -x 1

# Find large files
find / -type f -size +1G 2>/dev/null
```

## Environment Variables

```bash
# Performance monitoring
export PERF_RECORD_RATE=1000
export MALLOC_CHECK_=1
export MALLOC_PERTURB_=1

# Debug settings
export GLIBC_TUNABLES=glibc.malloc.check=1
export LD_DEBUG=statistics
```

---

*For more details, see individual tool man pages: `man <tool>`*