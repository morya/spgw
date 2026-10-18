package rconn

import (
	"context"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// 从 key 读取结果并存入 to 结构
func ReadJson(key string, to any) error {
	data, err := Rconn.Get(context.TODO(), key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, to)
}

// dump json from v and save into redis `key`
// duration: could be 0, is the expiration time of the key
func SaveJson(key string, v any, duration time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return Rconn.Set(context.TODO(), key, data, duration).Err()
}

// Delete 删除指定 key
func Delete(key string) error {
	return Rconn.Del(context.TODO(), key).Err()
}

func SaveLastSeqID(typ string, nodeID uint32, seqID uint32) error {
	key := fmt.Sprintf("%s:%d", typ, nodeID)
	return Rconn.Set(context.TODO(), key, int64(seqID), 0).Err()
}

func GetLastSeqID(typ string, nodeID uint32) uint32 {
	key := fmt.Sprintf("%s:%d", typ, nodeID)
	seqID, _ := Rconn.Get(context.TODO(), key).Int64()
	return uint32(seqID)
}
