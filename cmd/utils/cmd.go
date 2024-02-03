package utils

import (
	"fmt"
	"strings"
)

// ParseCluster takes an array of strings of the form "ID:Address"
// and parses it into a map where the keys are IDs and the values
// are arguments.
func ParseCluster(cluster []string) (map[string]string, error) {
	configuration := make(map[string]string, len(cluster))
	for _, member := range cluster {
		id, address, ok := strings.Cut(member, ":")
		if !ok {
			return nil, fmt.Errorf("failed to parse cluster: cluster = %v", cluster)
		}
		configuration[id] = address
	}
	return configuration, nil
}
