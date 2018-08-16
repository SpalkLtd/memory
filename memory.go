package memory

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
