package rconn

import (
	"context"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

var Rconn *redis.Client

func Connect(redisAddress string) error {
	Rconn = redis.NewClient(
		&redis.Options{
			Addr:       redisAddress,
			MaxRetries: -1,
		},
	)
	if err := Rconn.Ping(context.Background()).Err(); err != nil {
		return errors.Wrapf(err, "redis connect failed, %s", redisAddress)
	}

	return nil
}

func Close() {
	if Rconn != nil {
		Rconn.Close()
		Rconn = nil
	}
}
