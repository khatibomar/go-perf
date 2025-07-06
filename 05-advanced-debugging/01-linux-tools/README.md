# Chapter 01: Linux Performance Tools for Go

**Focus:** System-level debugging and performance analysis

## Chapter Overview

This chapter focuses on mastering Linux performance tools specifically for Go application debugging and optimization. You'll learn to use system-level monitoring tools, analyze process and memory behavior, debug network and I/O issues, and leverage advanced profiling techniques including BPF/eBPF.

## Learning Objectives

By the end of this chapter, you will:

- ✅ Master essential Linux performance monitoring tools
- ✅ Analyze Go process and memory behavior at system level
- ✅ Debug network and I/O performance issues
- ✅ Use advanced profiling tools including BPF/eBPF
- ✅ Correlate system metrics with Go application performance
- ✅ Implement automated monitoring and alerting
- ✅ Troubleshoot production performance issues

## Prerequisites

- Advanced Linux command line knowledge
- Understanding of operating system concepts
- Go application deployment experience
- Basic knowledge of system administration

## Chapter Structure

### Exercise 01-01: System Performance Monitoring
**Objective:** Master essential system monitoring tools for Go applications

**Key Concepts:**
- CPU, memory, and disk monitoring
- System load analysis
- Resource utilization tracking
- Performance baseline establishment

**Performance Targets:**
- Monitor 100+ metrics simultaneously
- Alert response time <30 seconds
- Monitoring overhead <1%
- 99.9% monitoring uptime

### Exercise 01-02: Process and Memory Analysis
**Objective:** Deep dive into Go process behavior and memory management

**Key Concepts:**
- Process lifecycle monitoring
- Memory allocation analysis
- Garbage collection impact
- Resource leak detection

**Performance Targets:**
- Detect memory leaks within 5 minutes
- Process analysis overhead <0.5%
- Memory usage accuracy >99%
- Real-time process monitoring

### Exercise 01-03: Network and I/O Debugging
**Objective:** Debug network and I/O performance issues at system level

**Key Concepts:**
- Network traffic analysis
- I/O performance monitoring
- Connection tracking
- Bandwidth utilization

**Performance Targets:**
- Network latency measurement accuracy <1ms
- I/O throughput monitoring >1GB/s
- Connection tracking for 10K+ connections
- Real-time traffic analysis

### Exercise 01-04: Advanced Profiling with BPF/eBPF
**Objective:** Implement advanced profiling using BPF/eBPF tools

**Key Concepts:**
- eBPF program development
- Kernel-level profiling
- Custom metric collection
- Zero-overhead monitoring

**Performance Targets:**
- Custom profiling overhead <0.1%
- Real-time kernel event monitoring
- Custom metric collection >1M events/sec
- Production-safe profiling

## Key Technologies

### Essential System Tools
- `htop/top` - Process monitoring
- `vmstat` - Virtual memory statistics
- `iostat` - I/O statistics
- `sar` - System activity reporter
- `free` - Memory usage

### Process Analysis Tools
- `ps` - Process status
- `pstree` - Process tree
- `lsof` - List open files
- `strace` - System call tracer
- `ltrace` - Library call tracer

### Network Tools
- `netstat/ss` - Network statistics
- `tcpdump` - Packet capture
- `wireshark` - Network protocol analyzer
- `iperf3` - Network performance testing
- `nload` - Network load monitor

### Advanced Tools
- `perf` - Performance analysis
- `ftrace` - Function tracer
- `bpftrace` - eBPF scripting
- `bcc-tools` - BPF Compiler Collection

## Implementation Patterns

### System Monitoring Script
```bash
#!/bin/bash
# comprehensive-monitor.sh

LOG_DIR="/var/log/go-app-monitoring"
APP_PID=$(pgrep -f "your-go-app")
INTERVAL=5

# Create log directory
mkdir -p $LOG_DIR

# Function to log with timestamp
log_with_timestamp() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $1" >> $LOG_DIR/$2
}

# Monitor system resources
monitor_system() {
    while true; do
        # CPU usage
        CPU_USAGE=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)
        log_with_timestamp "CPU_USAGE:$CPU_USAGE" "system.log"
        
        # Memory usage
        MEM_INFO=$(free -m | awk 'NR==2{printf "%.2f", $3*100/$2}')
        log_with_timestamp "MEMORY_USAGE:$MEM_INFO" "system.log"
        
        # Disk I/O
        DISK_IO=$(iostat -x 1 1 | awk 'NR>3 && $1!="" {print $1":"$4":"$5}' | head -1)
        log_with_timestamp "DISK_IO:$DISK_IO" "system.log"
        
        # Load average
        LOAD_AVG=$(uptime | awk -F'load average:' '{print $2}')
        log_with_timestamp "LOAD_AVERAGE:$LOAD_AVG" "system.log"
        
        sleep $INTERVAL
    done
}

# Monitor Go application process
monitor_go_process() {
    if [ -z "$APP_PID" ]; then
        log_with_timestamp "Go application not found" "process.log"
        return
    fi
    
    while true; do
        if ! kill -0 $APP_PID 2>/dev/null; then
            log_with_timestamp "Process $APP_PID no longer exists" "process.log"
            APP_PID=$(pgrep -f "your-go-app")
            continue
        fi
        
        # Process CPU and memory
        PROC_STATS=$(ps -p $APP_PID -o pid,pcpu,pmem,vsz,rss,etime --no-headers)
        log_with_timestamp "PROCESS_STATS:$PROC_STATS" "process.log"
        
        # File descriptors
        FD_COUNT=$(lsof -p $APP_PID 2>/dev/null | wc -l)
        log_with_timestamp "FILE_DESCRIPTORS:$FD_COUNT" "process.log"
        
        # Network connections
        NET_CONN=$(netstat -an | grep $APP_PID | wc -l)
        log_with_timestamp "NETWORK_CONNECTIONS:$NET_CONN" "process.log"
        
        sleep $INTERVAL
    done
}

# Start monitoring in background
monitor_system &
monitor_go_process &

echo "Monitoring started. Logs in $LOG_DIR"
wait
```

### Memory Analysis Script
```bash
#!/bin/bash
# memory-analyzer.sh

APP_PID=$(pgrep -f "your-go-app")

if [ -z "$APP_PID" ]; then
    echo "Go application not found"
    exit 1
fi

echo "Analyzing memory for PID: $APP_PID"

# Memory map analysis
echo "=== Memory Map ==="
pmap -x $APP_PID

# Detailed memory information
echo "\n=== Detailed Memory Info ==="
cat /proc/$APP_PID/smaps | awk '
/^[0-9a-f]/ { addr = $1 }
/^Size:/ { size += $2 }
/^Rss:/ { rss += $2 }
/^Pss:/ { pss += $2 }
/^Shared_Clean:/ { shared_clean += $2 }
/^Shared_Dirty:/ { shared_dirty += $2 }
/^Private_Clean:/ { private_clean += $2 }
/^Private_Dirty:/ { private_dirty += $2 }
END {
    printf "Total Size: %d KB\n", size
    printf "RSS: %d KB\n", rss
    printf "PSS: %d KB\n", pss
    printf "Shared Clean: %d KB\n", shared_clean
    printf "Shared Dirty: %d KB\n", shared_dirty
    printf "Private Clean: %d KB\n", private_clean
    printf "Private Dirty: %d KB\n", private_dirty
}'

# Go-specific memory stats
echo "\n=== Go Memory Stats ==="
curl -s http://localhost:6060/debug/vars | jq '.memstats' 2>/dev/null || echo "pprof endpoint not available"

# Memory growth tracking
echo "\n=== Memory Growth Tracking ==="
for i in {1..10}; do
    RSS=$(ps -p $APP_PID -o rss --no-headers)
    VSZ=$(ps -p $APP_PID -o vsz --no-headers)
    echo "$(date '+%H:%M:%S') RSS: ${RSS}KB VSZ: ${VSZ}KB"
    sleep 5
done
```

### Network Debugging Script
```bash
#!/bin/bash
# network-debugger.sh

APP_PID=$(pgrep -f "your-go-app")
APP_PORT=${1:-8080}

echo "Debugging network for PID: $APP_PID, Port: $APP_PORT"

# Active connections
echo "=== Active Connections ==="
netstat -tuln | grep :$APP_PORT
ss -tuln | grep :$APP_PORT

# Connection states
echo "\n=== Connection States ==="
ss -tan | awk 'NR>1 {state[$1]++} END {for (s in state) print s, state[s]}'

# Network traffic monitoring
echo "\n=== Network Traffic (10 seconds) ==="
tcpdump -i any port $APP_PORT -c 100 &
TCPDUMP_PID=$!
sleep 10
kill $TCPDUMP_PID 2>/dev/null

# Bandwidth monitoring
echo "\n=== Bandwidth Usage ==="
iftop -t -s 10 2>/dev/null || echo "iftop not available"

# Socket statistics
echo "\n=== Socket Statistics ==="
ss -s

# Network performance test
echo "\n=== Network Performance Test ==="
iperf3 -c localhost -p $APP_PORT -t 5 2>/dev/null || echo "iperf3 test not available"
```

## Advanced Profiling with eBPF

### Custom eBPF Script for Go Profiling
```python
#!/usr/bin/env python3
# go-profiler.py

from bcc import BPF
import time
import signal
import sys

# eBPF program
bpf_program = """
#include <uapi/linux/ptrace.h>
#include <linux/sched.h>

struct data_t {
    u32 pid;
    u64 ts;
    char comm[TASK_COMM_LEN];
    u64 duration;
};

BPF_PERF_OUTPUT(events);
BPF_HASH(start, u32);

int trace_go_func_entry(struct pt_regs *ctx) {
    u32 pid = bpf_get_current_pid_tgid();
    u64 ts = bpf_ktime_get_ns();
    start.update(&pid, &ts);
    return 0;
}

int trace_go_func_return(struct pt_regs *ctx) {
    u32 pid = bpf_get_current_pid_tgid();
    u64 *tsp, delta;
    
    tsp = start.lookup(&pid);
    if (tsp == 0) {
        return 0;
    }
    
    delta = bpf_ktime_get_ns() - *tsp;
    start.delete(&pid);
    
    struct data_t data = {};
    data.pid = pid;
    data.ts = bpf_ktime_get_ns();
    data.duration = delta;
    bpf_get_current_comm(&data.comm, sizeof(data.comm));
    
    events.perf_submit(ctx, &data, sizeof(data));
    return 0;
}
"""

class GoProfiler:
    def __init__(self, pid=None):
        self.pid = pid
        self.b = BPF(text=bpf_program)
        self.function_stats = {}
        
    def attach_probes(self):
        # Attach to Go runtime functions
        try:
            self.b.attach_uprobe(name="/usr/local/go/bin/go", 
                               sym="runtime.mallocgc", 
                               fn_name="trace_go_func_entry",
                               pid=self.pid)
            self.b.attach_uretprobe(name="/usr/local/go/bin/go", 
                                  sym="runtime.mallocgc", 
                                  fn_name="trace_go_func_return",
                                  pid=self.pid)
        except Exception as e:
            print(f"Failed to attach probes: {e}")
            
    def print_event(self, cpu, data, size):
        event = self.b["events"].event(data)
        duration_ms = event.duration / 1000000
        
        func_name = event.comm.decode('utf-8', 'replace')
        if func_name not in self.function_stats:
            self.function_stats[func_name] = {
                'count': 0,
                'total_duration': 0,
                'min_duration': float('inf'),
                'max_duration': 0
            }
            
        stats = self.function_stats[func_name]
        stats['count'] += 1
        stats['total_duration'] += duration_ms
        stats['min_duration'] = min(stats['min_duration'], duration_ms)
        stats['max_duration'] = max(stats['max_duration'], duration_ms)
        
        print(f"PID: {event.pid}, Function: {func_name}, Duration: {duration_ms:.2f}ms")
        
    def start_profiling(self, duration=60):
        print(f"Starting Go profiling for {duration} seconds...")
        self.attach_probes()
        self.b["events"].open_perf_buffer(self.print_event)
        
        start_time = time.time()
        try:
            while time.time() - start_time < duration:
                self.b.perf_buffer_poll()
        except KeyboardInterrupt:
            pass
            
        self.print_summary()
        
    def print_summary(self):
        print("\n=== Profiling Summary ===")
        for func_name, stats in sorted(self.function_stats.items(), 
                                     key=lambda x: x[1]['total_duration'], 
                                     reverse=True):
            avg_duration = stats['total_duration'] / stats['count']
            print(f"Function: {func_name}")
            print(f"  Calls: {stats['count']}")
            print(f"  Total Duration: {stats['total_duration']:.2f}ms")
            print(f"  Average Duration: {avg_duration:.2f}ms")
            print(f"  Min Duration: {stats['min_duration']:.2f}ms")
            print(f"  Max Duration: {stats['max_duration']:.2f}ms")
            print()

if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description='Go Application Profiler using eBPF')
    parser.add_argument('-p', '--pid', type=int, help='Process ID to profile')
    parser.add_argument('-d', '--duration', type=int, default=60, help='Profiling duration in seconds')
    
    args = parser.parse_args()
    
    profiler = GoProfiler(pid=args.pid)
    
    def signal_handler(sig, frame):
        print('\nStopping profiler...')
        profiler.print_summary()
        sys.exit(0)
        
    signal.signal(signal.SIGINT, signal_handler)
    profiler.start_profiling(args.duration)
```

## Profiling and Debugging

### System Performance Analysis
```bash
# Comprehensive system analysis
echo "=== System Overview ==="
uname -a
uptime
free -h
df -h

echo "\n=== CPU Information ==="
lscpu
cat /proc/cpuinfo | grep "model name" | head -1

echo "\n=== Memory Information ==="
cat /proc/meminfo | head -10

echo "\n=== Disk Information ==="
lsblk
iostat -x 1 1

echo "\n=== Network Information ==="
ip addr show
ss -tuln
```

### Go Application Analysis
```bash
# Go-specific analysis
echo "=== Go Application Analysis ==="
APP_PID=$(pgrep -f "your-go-app")

if [ ! -z "$APP_PID" ]; then
    echo "Application PID: $APP_PID"
    
    # Process information
    ps -p $APP_PID -o pid,ppid,pcpu,pmem,vsz,rss,etime,cmd
    
    # Memory details
    cat /proc/$APP_PID/status | grep -E "(VmPeak|VmSize|VmRSS|VmData|VmStk|VmExe)"
    
    # File descriptors
    echo "Open file descriptors: $(lsof -p $APP_PID 2>/dev/null | wc -l)"
    
    # Network connections
    echo "Network connections: $(netstat -an | grep $APP_PID | wc -l)"
    
    # Go runtime stats
    curl -s http://localhost:6060/debug/vars | jq '.memstats.Alloc, .memstats.Sys, .memstats.NumGC' 2>/dev/null
else
    echo "Go application not found"
fi
```

## Performance Optimization Techniques

### 1. System-Level Optimization
- Monitor system resource utilization
- Optimize kernel parameters
- Use appropriate CPU governors
- Configure memory management

### 2. Process Optimization
- Monitor process behavior
- Optimize resource allocation
- Implement proper cleanup
- Use process isolation

### 3. Network Optimization
- Monitor network performance
- Optimize network stack
- Use appropriate protocols
- Implement connection pooling

### 4. I/O Optimization
- Monitor I/O patterns
- Optimize file system usage
- Use appropriate I/O schedulers
- Implement caching strategies

## Common Pitfalls

❌ **Monitoring Issues:**
- Insufficient monitoring coverage
- High monitoring overhead
- Poor alert configuration
- Missing critical metrics

❌ **Analysis Problems:**
- Not correlating system and application metrics
- Focusing on wrong metrics
- Not understanding tool limitations
- Poor baseline establishment

❌ **Tool Usage:**
- Using inappropriate tools
- Not understanding tool overhead
- Poor tool configuration
- Not validating results

❌ **Production Debugging:**
- Invasive debugging techniques
- Not considering system impact
- Poor incident response
- Inadequate documentation

## Best Practices

### System Monitoring
- Establish performance baselines
- Monitor key system metrics
- Use multiple monitoring tools
- Implement automated alerting
- Document normal behavior

### Debugging Methodology
- Use systematic approaches
- Start with least invasive tools
- Correlate multiple data sources
- Document findings
- Validate hypotheses

### Tool Selection
- Choose appropriate tools for the task
- Understand tool overhead
- Use multiple tools for verification
- Consider production impact
- Keep tools updated

### Production Considerations
- Minimize debugging impact
- Use non-invasive techniques
- Implement proper access controls
- Document procedures
- Train team members

## Success Criteria

- [ ] System monitoring covers all critical metrics
- [ ] Process analysis detects issues within 5 minutes
- [ ] Network debugging resolves issues quickly
- [ ] eBPF profiling provides detailed insights
- [ ] Monitoring overhead <1%
- [ ] Tool usage is documented and standardized
- [ ] Team is trained on debugging procedures

## Real-World Applications

- **Production Monitoring:** 24/7 system health monitoring
- **Performance Debugging:** Identifying bottlenecks in live systems
- **Capacity Planning:** Understanding resource requirements
- **Incident Response:** Quick diagnosis of production issues
- **Optimization:** Data-driven performance improvements
- **Security Analysis:** Detecting anomalous behavior

## Next Steps

After completing this chapter:
1. Proceed to **Chapter 02: Distributed Tracing and Observability**
2. Apply Linux tools to your production systems
3. Study advanced eBPF programming
4. Explore cloud-native monitoring solutions
5. Implement comprehensive monitoring strategies

---

**Ready to master Linux performance tools?** Start with [Exercise 01-01: System Performance Monitoring](./exercise-01-01/README.md)

*Chapter 01 of Module 05: Advanced Debugging and Production*