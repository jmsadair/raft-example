package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCluster(t *testing.T) {
	cluster := []string{"n1:127.0.0.0:8080", "n2:127.0.0.1:8080", "n3:127.0.0.2:8080"}
	cluserMap, err := ParseCluster(cluster)
	require.NoError(t, err)

	require.Equal(t, "127.0.0.0:8080", cluserMap["n1"])
	require.Equal(t, "127.0.0.1:8080", cluserMap["n2"])
	require.Equal(t, "127.0.0.2:8080", cluserMap["n3"])
}
