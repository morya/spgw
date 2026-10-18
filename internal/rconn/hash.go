package rconn

import "context"

func HashReadJson(key, fieldKey string, obj any) error {
	data, err := Rconn.HGet(context.TODO(), key, fieldKey).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

func HashSaveJson(key, fieldKey string, obj any) error {
	v, _ := json.Marshal(obj)
	return Rconn.HSet(context.TODO(), key, fieldKey, v).Err()
}
