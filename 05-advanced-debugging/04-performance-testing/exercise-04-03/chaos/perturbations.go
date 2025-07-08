// chaos/perturbations.go
package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// NetworkLatencyPerturbation adds network latency to target host
func NetworkLatencyPerturbation(targetHost string, latency time.Duration) Perturbation {
	return Perturbation{
		Name: "network_latency",
		Type: "network",
		Action: func(ctx context.Context) error {
			// Simulate network latency using tc (traffic control) on Linux
			if runtime.GOOS == "linux" {
				cmd := exec.CommandContext(ctx, "tc", "qdisc", "add", "dev", "eth0", "root", "netem", "delay", fmt.Sprintf("%dms", latency.Milliseconds()))
				return cmd.Run()
			}
			// For non-Linux systems, simulate with sleep
			time.Sleep(100 * time.Millisecond)
			return nil
		},
		Cleanup: func(ctx context.Context) error {
			if runtime.GOOS == "linux" {
				cmd := exec.CommandContext(ctx, "tc", "qdisc", "del", "dev", "eth0", "root")
				return cmd.Run()
			}
			return nil
		},
		Duration:    5 * time.Minute,
		Probability: 1.0,
		Scope: PerturbationScope{
			Targets:    []string{targetHost},
			Percentage: 100.0,
			Selection:  "specific",
		},
	}
}

// CPUStressPerturbation stresses CPU to specified percentage
func CPUStressPerturbation(cpuPercent int) Perturbation {
	var stressCancel context.CancelFunc
	var stressMu sync.Mutex

	return Perturbation{
		Name: "cpu_stress",
		Type: "cpu",
		Action: func(ctx context.Context) error {
			stressMu.Lock()
			defer stressMu.Unlock()

			// Create stress context
			stressCtx, cancel := context.WithCancel(ctx)
			stressCancel = cancel

			// Start CPU stress goroutines
			numCPU := runtime.NumCPU()
			for i := 0; i < numCPU; i++ {
				go func() {
					for {
						select {
						case <-stressCtx.Done():
							return
						default:
							// Busy loop to consume CPU
							for j := 0; j < cpuPercent*1000; j++ {
								_ = j * j
							}
							// Small sleep to allow other processes
							time.Sleep(time.Duration(100-cpuPercent) * time.Microsecond)
						}
					}
				}()
			}

			return nil
		},
		Cleanup: func(ctx context.Context) error {
			stressMu.Lock()
			defer stressMu.Unlock()
			if stressCancel != nil {
				stressCancel()
			}
			return nil
		},
		Duration:    3 * time.Minute,
		Probability: 1.0,
		Scope: PerturbationScope{
			Targets:    []string{"localhost"},
			Percentage: 100.0,
			Selection:  "all",
		},
	}
}

// MemoryStressPerturbation consumes specified amount of memory
func MemoryStressPerturbation(memoryMB int) Perturbation {
	var memoryBlocks [][]byte
	var memoryMu sync.Mutex

	return Perturbation{
		Name: "memory_stress",
		Type: "memory",
		Action: func(ctx context.Context) error {
			memoryMu.Lock()
			defer memoryMu.Unlock()

			// Allocate memory in chunks
			chunkSize := 1024 * 1024 // 1MB chunks
			numChunks := memoryMB

			memoryBlocks = make([][]byte, numChunks)
			for i := 0; i < numChunks; i++ {
				memoryBlocks[i] = make([]byte, chunkSize)
				// Write to memory to ensure allocation
				for j := 0; j < chunkSize; j += 4096 {
					memoryBlocks[i][j] = byte(i % 256)
				}
			}

			return nil
		},
		Cleanup: func(ctx context.Context) error {
			memoryMu.Lock()
			defer memoryMu.Unlock()
			memoryBlocks = nil
			runtime.GC()
			return nil
		},
		Duration:    2 * time.Minute,
		Probability: 1.0,
		Scope: PerturbationScope{
			Targets:    []string{"localhost"},
			Percentage: 100.0,
			Selection:  "all",
		},
	}
}

// DiskStressPerturbation fills disk with temporary files
func DiskStressPerturbation(diskMB int, targetPath string) Perturbation {
	var tempFiles []string
	var diskMu sync.Mutex

	return Perturbation{
		Name: "disk_stress",
		Type: "disk",
		Action: func(ctx context.Context) error {
			diskMu.Lock()
			defer diskMu.Unlock()

			// Create temporary files
			chunkSize := 1024 * 1024 // 1MB chunks
			numFiles := diskMB
			data := make([]byte, chunkSize)

			for i := 0; i < numFiles; i++ {
				filename := fmt.Sprintf("%s/chaos_disk_stress_%d.tmp", targetPath, i)
				file, err := os.Create(filename)
				if err != nil {
					return fmt.Errorf("failed to create stress file: %w", err)
				}

				if _, err := file.Write(data); err != nil {
					file.Close()
					return fmt.Errorf("failed to write stress file: %w", err)
				}

				file.Close()
				tempFiles = append(tempFiles, filename)
			}

			return nil
		},
		Cleanup: func(ctx context.Context) error {
			diskMu.Lock()
			defer diskMu.Unlock()

			for _, filename := range tempFiles {
				os.Remove(filename)
			}
			tempFiles = nil
			return nil
		},
		Duration:    2 * time.Minute,
		Probability: 1.0,
		Scope: PerturbationScope{
			Targets:    []string{targetPath},
			Percentage: 100.0,
			Selection:  "specific",
		},
	}
}

// ServiceKillPerturbation kills and restarts a service process
func ServiceKillPerturbation(serviceName string) Perturbation {
	return Perturbation{
		Name: "service_kill",
		Type: "service",
		Action: func(ctx context.Context) error {
			// Kill service using pkill
			cmd := exec.CommandContext(ctx, "pkill", "-f", serviceName)
			if err := cmd.Run(); err != nil {
				// Service might not be running, which is okay
				return nil
			}
			return nil
		},
		Cleanup: func(ctx context.Context) error {
			// Restart service (implementation depends on service manager)
			if runtime.GOOS == "linux" {
				cmd := exec.CommandContext(ctx, "systemctl", "restart", serviceName)
				return cmd.Run()
			}
			return nil
		},
		Duration:    1 * time.Minute,
		Probability: 0.5,
		Scope: PerturbationScope{
			Targets:    []string{serviceName},
			Percentage: 100.0,
			Selection:  "specific",
		},
	}
}

// NetworkPartitionPerturbation simulates network partition
func NetworkPartitionPerturbation(targetHosts []string) Perturbation {
	return Perturbation{
		Name: "network_partition",
		Type: "network",
		Action: func(ctx context.Context) error {
			// Block traffic to target hosts using iptables
			for _, host := range targetHosts {
				if runtime.GOOS == "linux" {
					cmd := exec.CommandContext(ctx, "iptables", "-A", "OUTPUT", "-d", host, "-j", "DROP")
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("failed to block traffic to %s: %w", host, err)
					}
				}
			}
			return nil
		},
		Cleanup: func(ctx context.Context) error {
			// Remove iptables rules
			for _, host := range targetHosts {
				if runtime.GOOS == "linux" {
					cmd := exec.CommandContext(ctx, "iptables", "-D", "OUTPUT", "-d", host, "-j", "DROP")
					cmd.Run() // Ignore errors during cleanup
				}
			}
			return nil
		},
		Duration:    3 * time.Minute,
		Probability: 0.7,
		Scope: PerturbationScope{
			Targets:    targetHosts,
			Percentage: 100.0,
			Selection:  "all",
		},
	}
}

// ProcessKillPerturbation kills random processes
func ProcessKillPerturbation(processPattern string, killSignal string) Perturbation {
	return Perturbation{
		Name: "process_kill",
		Type: "process",
		Action: func(ctx context.Context) error {
			// Kill processes matching pattern
			signal := "-TERM"
			if killSignal != "" {
				signal = "-" + killSignal
			}

			cmd := exec.CommandContext(ctx, "pkill", signal, "-f", processPattern)
			return cmd.Run()
		},
		Cleanup: func(ctx context.Context) error {
			// No cleanup needed for process kill
			return nil
		},
		Duration:    30 * time.Second,
		Probability: 0.3,
		Scope: PerturbationScope{
			Targets:    []string{processPattern},
			Percentage: 50.0,
			Selection:  "random",
		},
	}
}

// FileCorruptionPerturbation corrupts files in target directory
func FileCorruptionPerturbation(targetDir string, corruptionRate float64) Perturbation {
	var corruptedFiles []string
	var corruptMu sync.Mutex

	return Perturbation{
		Name: "file_corruption",
		Type: "disk",
		Action: func(ctx context.Context) error {
			corruptMu.Lock()
			defer corruptMu.Unlock()

			// Find files to corrupt
			files, err := os.ReadDir(targetDir)
			if err != nil {
				return fmt.Errorf("failed to read directory: %w", err)
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				// Randomly select files based on corruption rate
				if rand.Float64() > corruptionRate {
					continue
				}

				filePath := fmt.Sprintf("%s/%s", targetDir, file.Name())
				
				// Create backup
				backupPath := filePath + ".chaos_backup"
				if err := copyFile(filePath, backupPath); err != nil {
					continue
				}

				// Corrupt file by writing random data
				if err := corruptFile(filePath); err != nil {
					os.Remove(backupPath)
					continue
				}

				corruptedFiles = append(corruptedFiles, filePath)
			}

			return nil
		},
		Cleanup: func(ctx context.Context) error {
			corruptMu.Lock()
			defer corruptMu.Unlock()

			// Restore corrupted files from backup
			for _, filePath := range corruptedFiles {
				backupPath := filePath + ".chaos_backup"
				if err := copyFile(backupPath, filePath); err == nil {
					os.Remove(backupPath)
				}
			}
			corruptedFiles = nil
			return nil
		},
		Duration:    1 * time.Minute,
		Probability: 0.2,
		Scope: PerturbationScope{
			Targets:    []string{targetDir},
			Percentage: corruptionRate * 100,
			Selection:  "random",
		},
	}
}

// Helper functions
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	buffer := make([]byte, 4096)
	for {
		n, err := srcFile.Read(buffer)
		if n > 0 {
			if _, writeErr := dstFile.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			break
		}
	}
	return nil
}

func corruptFile(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write random data to corrupt the file
	corruptData := make([]byte, 1024)
	for i := range corruptData {
		corruptData[i] = byte(rand.Intn(256))
	}

	_, err = file.Write(corruptData)
	return err
}

// PortExhaustionPerturbation exhausts available ports
func PortExhaustionPerturbation(startPort, endPort int) Perturbation {
	var connections []interface{}
	var portMu sync.Mutex

	return Perturbation{
		Name: "port_exhaustion",
		Type: "network",
		Action: func(ctx context.Context) error {
			portMu.Lock()
			defer portMu.Unlock()

			// Bind to ports to exhaust them
			for port := startPort; port <= endPort; port++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				cmd := exec.CommandContext(ctx, "nc", "-l", "-p", strconv.Itoa(port))
				if err := cmd.Start(); err == nil {
					connections = append(connections, cmd)
				}
				
				// Limit to prevent system overload
				if len(connections) > 1000 {
					break
				}
			}

			return nil
		},
		Cleanup: func(ctx context.Context) error {
			portMu.Lock()
			defer portMu.Unlock()

			// Close all connections
			for _, conn := range connections {
				if cmd, ok := conn.(*exec.Cmd); ok {
					if cmd.Process != nil {
						cmd.Process.Kill()
					}
				}
			}
			connections = nil
			return nil
		},
		Duration:    2 * time.Minute,
		Probability: 0.3,
		Scope: PerturbationScope{
			Targets:    []string{fmt.Sprintf("ports_%d-%d", startPort, endPort)},
			Percentage: 100.0,
			Selection:  "all",
		},
	}
}