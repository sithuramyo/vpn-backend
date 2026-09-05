package vpn

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SystemMetrics is a dependency-free reader of host-level health, used by
// the /servers/:id/health endpoint when the API runs on the same VPS as
// the VPN data plane (the recommended single-VPS deployment). It reads
// /proc directly rather than pulling in a large metrics library, and
// degrades to zero values on non-Linux hosts (e.g. local development on
// Windows/macOS) instead of failing.
type SystemMetrics struct {
	CPUUsagePercent float64
	MemoryUsedPct   float64
	DiskUsedPct     float64
	UptimeSeconds   float64
}

func ReadSystemMetrics() (*SystemMetrics, error) {
	if runtime.GOOS != "linux" {
		return &SystemMetrics{}, nil
	}

	cpu, err := readCPUUsage()
	if err != nil {
		return nil, fmt.Errorf("read cpu usage: %w", err)
	}
	mem, err := readMemoryUsage()
	if err != nil {
		return nil, fmt.Errorf("read memory usage: %w", err)
	}
	uptime, err := readUptime()
	if err != nil {
		return nil, fmt.Errorf("read uptime: %w", err)
	}

	return &SystemMetrics{
		CPUUsagePercent: cpu,
		MemoryUsedPct:   mem,
		UptimeSeconds:   uptime,
	}, nil
}

func readCPUUsage() (float64, error) {
	sample := func() ([]uint64, error) {
		f, err := os.Open("/proc/stat")
		if err != nil {
			return nil, err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		if !scanner.Scan() {
			return nil, fmt.Errorf("empty /proc/stat")
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 || fields[0] != "cpu" {
			return nil, fmt.Errorf("unexpected /proc/stat format")
		}

		values := make([]uint64, 0, len(fields)-1)
		for _, f := range fields[1:] {
			v, err := strconv.ParseUint(f, 10, 64)
			if err != nil {
				return nil, err
			}
			values = append(values, v)
		}
		return values, nil
	}

	before, err := sample()
	if err != nil {
		return 0, err
	}
	time.Sleep(100 * time.Millisecond)
	after, err := sample()
	if err != nil {
		return 0, err
	}

	var totalBefore, totalAfter, idleBefore, idleAfter uint64
	for _, v := range before {
		totalBefore += v
	}
	for _, v := range after {
		totalAfter += v
	}
	idleBefore = before[3]
	idleAfter = after[3]

	totalDelta := float64(totalAfter - totalBefore)
	idleDelta := float64(idleAfter - idleBefore)
	if totalDelta <= 0 {
		return 0, nil
	}
	return (1 - idleDelta/totalDelta) * 100, nil
}

func readMemoryUsage() (float64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	values := make(map[string]uint64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		v, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		values[key] = v
	}

	total := values["MemTotal"]
	available := values["MemAvailable"]
	if total == 0 {
		return 0, fmt.Errorf("could not read MemTotal")
	}
	used := total - available
	return float64(used) / float64(total) * 100, nil
}

func readUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected /proc/uptime format")
	}
	return strconv.ParseFloat(fields[0], 64)
}

// IsServiceActive checks a systemd unit's status via `systemctl is-active`.
// The unit name is always one of a small set of hardcoded constants
// (never user input), so this exec call carries no injection risk.
func IsServiceActive(unit string) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	out, err := exec.Command("systemctl", "is-active", unit).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}
