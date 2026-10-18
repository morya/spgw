package server

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cast"
	"github.com/urfave/cli/v2"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/util/k8s"
)

func initNodeID() error {
	if k8s.IsInK8s() {
		hostname := os.Getenv("HOSTNAME")

		arr := strings.Split(hostname, "-")
		if len(arr) < 2 {
			return fmt.Errorf("invalid hostname %s", hostname)
		}
		podID := cast.ToUint32(arr[len(arr)-1])
		if podID == 0 {
			return fmt.Errorf("invalid hostname %s: podID is 0", hostname)
		}
		NodeID = podID + 1

	} else {
		NodeID = 1
	}
	return nil
}

func before(ctx *cli.Context) error {
	if err := initNodeID(); err != nil {
		return err
	}

	return db.ConnectMainDB(gconfig.Cfg.MainDB)
}
