package infrastructure

import (
	"fmt"
	"strings"
)

func encodeMachineID(partition, machineID string) string {
	return fmt.Sprintf("metal:///%s/%s", partition, machineID)
}

func decodeMachineID(id string) string {
	splitProviderID := strings.Split(id, "/")
	return splitProviderID[len(splitProviderID)-1]
}
