package validation

import (
	"fmt"
	"os"
	"runtime"
)

// HostConfig captures prerequisites required by the installer.
type HostConfig struct {
	RequireSudo     bool
	KernelModules   []string
	MinCPU          int
	MinMemoryGiB    int
	FilesystemPaths []string
}

// Result describes the outcome of the preflight run.
type Result struct {
	Passed bool
	Issues []string
}

// ValidateHost performs local host validation prior to bootstrap or reuse flows.
func ValidateHost(cfg HostConfig, sys SystemInspector) Result {
	if sys == nil {
		sys = DefaultInspector{}
	}

	issues := []string{}

	if cfg.MinCPU > 0 {
		cores := sys.CPUCount()
		if cores < cfg.MinCPU {
			issues = append(issues, fmt.Sprintf("require >= %d cpu cores, detected %d", cfg.MinCPU, cores))
		}
	}

	if cfg.MinMemoryGiB > 0 {
		mem := sys.MemoryGiB()
		if mem < cfg.MinMemoryGiB {
			issues = append(issues, fmt.Sprintf("require >= %d GiB memory, detected %d GiB", cfg.MinMemoryGiB, mem))
		}
	}

	for _, module := range cfg.KernelModules {
		if !sys.HasKernelModule(module) {
			issues = append(issues, fmt.Sprintf("missing kernel module %s", module))
		}
	}

	for _, path := range cfg.FilesystemPaths {
		if _, err := os.Stat(path); err != nil {
			issues = append(issues, fmt.Sprintf("path missing: %s", path))
		}
	}

	if cfg.RequireSudo && !sys.HasSudoPrivileges() {
		issues = append(issues, "requires sudo privileges")
	}

	return Result{Passed: len(issues) == 0, Issues: issues}
}

// SystemInspector models host interrogation functions, allowing tests to stub.
type SystemInspector interface {
	CPUCount() int
	MemoryGiB() int
	HasKernelModule(string) bool
	HasSudoPrivileges() bool
}

// DefaultInspector interrogates the running host.
type DefaultInspector struct{}

// CPUCount returns logical CPUs.
func (DefaultInspector) CPUCount() int { return runtime.NumCPU() }

// MemoryGiB returns available memory in GiB.
func (DefaultInspector) MemoryGiB() int {
	// Linux implementation: read from /proc/meminfo
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			var memTotalKB int
			for _, line := range splitLines(string(data)) {
				if n, _ := fmt.Sscanf(line, "MemTotal: %d kB", &memTotalKB); n == 1 {
					return memTotalKB / 1024 / 1024 // Convert kB to GiB
				}
			}
		}
	}
	// Fallback: return 0 if unable to detect
	return 0
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	lines := []string{}
	start := 0
	for i := range s {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// HasKernelModule always returns true until kernel module inspection is implemented.
func (DefaultInspector) HasKernelModule(string) bool { return true }

// HasSudoPrivileges checks if running as root.
func (DefaultInspector) HasSudoPrivileges() bool { return os.Geteuid() == 0 }
