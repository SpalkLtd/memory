package memory_test

import (
	"log"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/SpalkLtd/memory"
	"github.com/stretchr/testify/require"
)

func TestGetMemory(t *testing.T) {
	usage, err := memory.GetMemoryUsage()
	require.NoError(t, err)
	log.Printf("%#v\n", usage)
}

func TestGetMemoryOfChildProcess(t *testing.T) {
	cmd := exec.Command("cat")
	cmd.Start()
	usage, err := memory.GetMemoryUsageOfPid(cmd.Process.Pid)
	require.NoError(t, err)
	log.Printf("%#v\n", usage)
	cmd.Process.Kill()
}

func TestGetContainerMemory(t *testing.T) {
	usage, err := memory.GetMemUsageOfContainer()
	require.NoError(t, err)
	log.Printf("%#v\n", usage)
}

func TestGetCpuUsage(t *testing.T) {
	cpu, err := memory.GetCpuUsage()
	require.NoError(t, err)
	log.Printf("CPU Usage: %f", cpu)
}

func TestGetTcpConnStats(t *testing.T) {
	http.Get("https://www.google.com")
	stats, err := memory.GetTCPConnStats()
	require.NoError(t, err)
	log.Printf("%#v\n", stats)
}

func TestGetCPUUsage2(t *testing.T) {
	sample, err := memory.SampleCPUUsage()
	require.NoError(t, err)
	require.NotNil(t, sample)

	// Should have total CPU
	require.NotZero(t, sample.Total.Total, "Total CPU ticks should be non-zero")

	// Should have at least one core
	require.NotEmpty(t, sample.PerCore, "Should have at least one CPU core")

	// Each core should have non-zero total
	for name, core := range sample.PerCore {
		require.NotZero(t, core.Total, "Core %s should have non-zero total ticks", name)
	}

	// Timestamp should be recent
	require.WithinDuration(t, time.Now(), sample.Timestamp, time.Second)
}

func TestGetCPUUsage2Consistency(t *testing.T) {
	sample, err := memory.SampleCPUUsage()
	require.NoError(t, err)

	// Sum of per-core totals should roughly equal total CPU
	// (not exact due to rounding in kernel)
	var perCoreSum uint64
	for _, core := range sample.PerCore {
		perCoreSum += core.Total
	}

	// Allow 1% tolerance
	tolerance := sample.Total.Total / 100
	diff := absDiff(perCoreSum, sample.Total.Total)
	require.LessOrEqual(t, diff, tolerance,
		"Per-core sum (%d) should be close to total (%d)", perCoreSum, sample.Total.Total)
}

func TestCalculateGetCPUUsage2(t *testing.T) {
	// Take two samples with a small delay
	sample1, err := memory.SampleCPUUsage()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	sample2, err := memory.SampleCPUUsage()
	require.NoError(t, err)

	usage := memory.CalculateCpuUsage(sample1, sample2)

	// Usage should be between 0 and 1
	require.GreaterOrEqual(t, usage.Total.Usage, 0.0)
	require.LessOrEqual(t, usage.Total.Usage, 1.0)

	// Steal should be between 0 and 1
	require.GreaterOrEqual(t, usage.Total.Steal, 0.0)
	require.LessOrEqual(t, usage.Total.Steal, 1.0)

	// Should have per-core usage
	require.NotEmpty(t, usage.PerCore)
	for name, coreUsage := range usage.PerCore {
		require.GreaterOrEqual(t, coreUsage.Usage, 0.0, "Core %s usage should be >= 0", name)
		require.LessOrEqual(t, coreUsage.Usage, 1.0, "Core %s usage should be <= 1", name)
		require.GreaterOrEqual(t, coreUsage.Steal, 0.0, "Core %s steal should be >= 0", name)
		require.LessOrEqual(t, coreUsage.Steal, 1.0, "Core %s steal should be <= 1", name)
	}
}

func TestSampleCpuCoreCount(t *testing.T) {
	sample, err := memory.SampleCPUUsage()
	require.NoError(t, err)

	// Should have at least 1 core, probably more on any modern system
	coreCount := len(sample.PerCore)
	require.GreaterOrEqual(t, coreCount, 1)

	t.Logf("Detected %d CPU cores", coreCount)
}

func absDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}
