package memory_test

import (
	"log"
	"testing"

	"github.com/SpalkLtd/synchroniser/memory"
	"github.com/stretchr/testify/require"
)

func TestGetMemory(t *testing.T) {
	usage, err := memory.GetMemoryUsage()
	require.NoError(t, err)
	log.Printf("%#v\n", usage)
}
