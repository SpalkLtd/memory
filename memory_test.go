package memory_test

import (
	"log"
	"os/exec"
	"testing"

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

func TestGetCpuUsage(t *testing.T) {
	cpu, err := memory.GetCpuUsage()
	require.NoError(t, err)
	log.Printf("CPU Usage: %f", cpu)
}
