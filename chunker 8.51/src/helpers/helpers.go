package helpers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
)

func JSONFromBytes[T any](data []byte) T {
	var target T
	_ = json.Unmarshal(data, &target)
	return target
}

func MD5(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}
