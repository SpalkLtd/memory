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

//GetMemUsageOfContainer does just that
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
			for i := 1; i < numFields; i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					fmt.Println("Error: ", i, fields[i], err)
				}
				total += val // tally up all the numbers to get total ticks
				if i == 4 {  // idle is the 5th field in the cpu line
					idle = val
				}
				if i == 8 { // steal is the 9th field in the cpu line
					steal = val
				}
			}
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

	idleTicks := float64(idle1 - idle0)
	stealTicks := float64(steal1 - steal0)
	totalTicks := float64(total1 - total0)
	usage.Usage = (totalTicks - idleTicks) / totalTicks
	usage.Steal = stealTicks / totalTicks
	return
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
