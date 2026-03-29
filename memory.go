package memory

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
 * Measures the current (and peak) resident and virtual memories
 * usage of your linux process, in kB
 */

type MemoryUsage struct {
	CurrRealMem int
	PeakRealMem int
	CurrVirtMem int
	PeakVirtMem int
}

type ContainerMemoryUsage struct {
	MemTotal     int
	MemFree      int
	MemAvailable int
	Buffers      int
	Cached       int
}

type CpuUsage struct {
	Usage float64
	Steal float64
}

// GetMemUsageOfContainer does just that
func GetMemUsageOfContainer() (usage ContainerMemoryUsage, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()

	// read the entire file
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			_, err = fmt.Sscanf(line, "MemTotal: %d kB", &usage.MemTotal)
			if err != nil {
				return
			}
		}
		if strings.HasPrefix(line, "MemFree:") {
			_, err = fmt.Sscanf(line, "MemFree: %d kB", &usage.MemFree)
			if err != nil {
				return
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			_, err = fmt.Sscanf(line, "MemAvailable: %d kB", &usage.MemAvailable)
			if err != nil {
				return
			}
		}
		if strings.HasPrefix(line, "Buffers:") {
			_, err = fmt.Sscanf(line, "Buffers: %d kB", &usage.Buffers)
			if err != nil {
				return
			}
		}
		if strings.HasPrefix(line, "Cached:") {
			_, err = fmt.Sscanf(line, "Cached: %d kB", &usage.Cached)
			if err != nil {
				return
			}
		}
	}
	if err = scanner.Err(); err != nil {
		return
	}
	return
}

func GetMemoryUsage() (MemoryUsage, error) {
	return GetMemoryUsageOfPid(-1)
}

func GetMemoryUsageOfPid(id int) (usage MemoryUsage, err error) {

	// linux file contains this-process info
	idString := ""
	if id <= 0 {
		idString = "self"
	} else {
		idString = fmt.Sprintf("%d", id)
	}

	f, err := os.Open("/proc/" + idString + "/status")
	if err != nil {
		return
	}
	defer f.Close()

	// read the entire file
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			_, err = fmt.Sscanf(line, "VmRSS: %d kB", &usage.CurrRealMem)
			if err != nil {
				return
			}
		}
		if strings.HasPrefix(line, "VmHWM:") {
			_, err = fmt.Sscanf(line, "VmHWM: %d kB", &usage.PeakRealMem)
			if err != nil {
				return
			}
		}
		if strings.HasPrefix(line, "VmSize:") {
			_, err = fmt.Sscanf(line, "VmSize: %d kB", &usage.CurrVirtMem)
			if err != nil {
				return
			}
		}
		if strings.HasPrefix(line, "VmPeak:") {
			_, err = fmt.Sscanf(line, "VmPeak: %d kB", &usage.PeakVirtMem)
			if err != nil {
				return
			}
		}
	}

	if err = scanner.Err(); err != nil {
		return
	}
	return
}

func getCpuSample() (idle, total, steal uint64, err error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if fields[0] == "cpu" {
			numFields := len(fields)
			var guest, guestNice uint64
			for i := 1; i < numFields; i++ {
				val, parseErr := strconv.ParseUint(fields[i], 10, 64)
				if parseErr != nil {
					fmt.Println("Error: ", i, fields[i], parseErr)
					continue
				}
				total += val // tally up all the numbers to get total ticks
				if i == 4 {  // idle is the 5th field in the cpu line
					idle = val
				}
				if i == 5 { // iowait is the 6th field - also idle time
					idle += val
				}
				if i == 8 { // steal is the 9th field in the cpu line
					steal = val
				}
				if i == 9 { // guest is the 10th field (already counted in user)
					guest = val
				}
				if i == 10 { // guest_nice is the 11th field (already counted in nice)
					guestNice = val
				}
			}
			// guest and guest_nice are already included in user and nice,
			// so subtract them to avoid double-counting.
			total -= guest + guestNice
			break
		}
	}

	err = scanner.Err()
	return
}

func GetCpuUsage() (usage CpuUsage, err error) {
	idle0, total0, steal0, err0 := getCpuSample()
	<-time.After(3 * time.Second)
	idle1, total1, steal1, err1 := getCpuSample()

	// Handle errors after sampling as sampling is time-sensitive
	if err0 != nil {
		err = err0
		return
	}
	if err1 != nil {
		err = err1
		return
	}

	totalDelta := saturatingSub(total1, total0)
	if totalDelta == 0 {
		return
	}
	idleDelta := saturatingSub(idle1, idle0)
	stealDelta := saturatingSub(steal1, steal0)
	td := float64(totalDelta)
	usage.Usage = float64(totalDelta-idleDelta) / td
	usage.Steal = float64(stealDelta) / td
	return
}

type CpuSample struct {
	Total     CoreSample            // Aggregate "cpu" line
	PerCore   map[string]CoreSample // "cpu0", "cpu1", etc.
	Timestamp time.Time
}

type CoreSample struct {
	Idle  uint64
	Total uint64
	Steal uint64
}

type CpuUsage2 struct {
	Total   CoreUsage
	PerCore map[string]CoreUsage
}

type CoreUsage struct {
	Usage float64
	Steal float64
}

// SampleCpu takes an instantaneous reading from /proc/stat
func SampleCPUUsage() (*CpuSample, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sample := &CpuSample{
		PerCore:   make(map[string]CoreSample),
		Timestamp: time.Now(),
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}

		core := parseCoreSample(fields[1:])
		if fields[0] == "cpu" {
			sample.Total = core
		} else {
			sample.PerCore[fields[0]] = core
		}
	}

	return sample, scanner.Err()
}

func parseCoreSample(fields []string) CoreSample {
	// /proc/stat cpu fields (0-indexed after label):
	// 0:user 1:nice 2:system 3:idle 4:iowait 5:irq 6:softirq 7:steal 8:guest 9:guest_nice
	//
	// guest is already included in user; guest_nice is already included in nice.
	// iowait is idle time (CPU waiting for I/O, not executing).
	var core CoreSample
	var guest, guestNice uint64
	for i, field := range fields {
		val, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			continue
		}
		core.Total += val
		switch i {
		case 3: // idle
			core.Idle = val
		case 4: // iowait - also idle time
			core.Idle += val
		case 7: // steal
			core.Steal = val
		case 8: // guest (already counted in user)
			guest = val
		case 9: // guest_nice (already counted in nice)
			guestNice = val
		}
	}
	// Subtract guest/guest_nice to avoid double-counting.
	core.Total -= guest + guestNice
	return core
}

// CalculateCpuUsage computes usage between two samples
func CalculateCpuUsage(prev, curr *CpuSample) CpuUsage2 {
	usage := CpuUsage2{
		Total:   calculateCoreUsage(prev.Total, curr.Total),
		PerCore: make(map[string]CoreUsage),
	}

	for name, currCore := range curr.PerCore {
		if prevCore, ok := prev.PerCore[name]; ok {
			usage.PerCore[name] = calculateCoreUsage(prevCore, currCore)
		}
	}

	return usage
}

func calculateCoreUsage(prev, curr CoreSample) CoreUsage {
	totalDelta := saturatingSub(curr.Total, prev.Total)
	if totalDelta == 0 {
		return CoreUsage{}
	}
	idleDelta := saturatingSub(curr.Idle, prev.Idle)
	stealDelta := saturatingSub(curr.Steal, prev.Steal)
	td := float64(totalDelta)
	return CoreUsage{
		Usage: float64(totalDelta-idleDelta) / td,
		Steal: float64(stealDelta) / td,
	}
}

// saturatingSub returns a - b, clamped to 0 if b > a.
// This guards against apparent negative deltas from CPU hot-plug events.
func saturatingSub(a, b uint64) uint64 {
	if b > a {
		return 0
	}
	return a - b
}

type TcpConnStats struct {
	InUse    int
	Orphan   int
	TimeWait int
	Alloc    int
	MemoryKB int
}

func GetTCPConnStats() (stats TcpConnStats, err error) {
	f, err := os.Open("/proc/net/sockstat")
	if err != nil {
		return stats, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "TCP:") {
			_, err = fmt.Sscanf(line, "TCP: inuse %d orphan %d tw %d alloc %d mem %d", &stats.InUse, &stats.Orphan, &stats.TimeWait, &stats.Alloc, &stats.MemoryKB)
			if err != nil {
				return
			}
		}
	}
	return
}
