package billdumper

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/morya/spgw/internal/util/k8s"
)

var NodeID uint32 = 0

func initNodeID() error {
	if k8s.IsInK8s() {
		hostname := os.Getenv("HOSTNAME")
		arr := strings.Split(hostname, "-")
		if len(arr) < 2 {
			return fmt.Errorf("invalid hostname %s", hostname)
		}
		podID, err := strconv.Atoi(arr[len(arr)-1])
		if err != nil {
			return fmt.Errorf("invalid hostname %s: %w", hostname, err)
		}
		NodeID = uint32(uint(podID) + 1)
	} else {
		NodeID = 1
	}
	return nil
}
