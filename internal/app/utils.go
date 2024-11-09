package app

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func selectAddress(addresses []string) string {
	n := big.NewInt(int64(len(addresses)))
	i, err := rand.Int(rand.Reader, n)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	} else if !i.IsInt64() {
		panic("crypto/rand: out of range")
	}
	return addresses[i.Int64()]
}
