package iagent

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/util/k8s"
)

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

func before(ctx *cli.Context) error {
	name := ctx.Command.Name
	glog.InitLoggers(ctx.String("loglevel"), ctx.String("logformat"), ctx.String("logdir"), name)

	if err := initNodeID(); err != nil {
		return err
	}

	return db.ConnectMainDB(gconfig.Cfg.MainDB)
}
