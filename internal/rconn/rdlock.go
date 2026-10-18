package rconn

import (
	"context"

	"github.com/bsm/redislock"
	"github.com/pkg/errors"
	redis "github.com/redis/go-redis/v9"
)

// 基于redis的分布式锁
var Lock *redislock.Client

func ConnectLock(redisAddress string) error {
	conn := redis.NewClient(&redis.Options{Addr: redisAddress})
	if err := conn.Ping(context.Background()).Err(); err != nil {
		return errors.Wrapf(err, "build redis lock failed, %s", redisAddress)
	}

	Lock = redislock.New(conn)
	return nil
}
