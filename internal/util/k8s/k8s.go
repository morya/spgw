package k8s

import (
	"os"
)

func IsInK8s() bool {
	_, found := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	return found
}
