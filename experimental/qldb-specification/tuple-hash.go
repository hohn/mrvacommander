package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func main() {
	atl_L := [][2]interface{}{
		{1, "s1"},
		{2, "str"},
	}

	var sl_hash []string

	for _, item := range atl_L {
		jsonBytes, err := json.Marshal(item)
		if err != nil {
			panic(err)
		}
		sum := md5.Sum(jsonBytes)
		sl_hash = append(sl_hash, hex.EncodeToString(sum[:]))
	}

	fmt.Println(sl_hash)
}
