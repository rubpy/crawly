package csync

import (
	"fmt"
	"math/rand"
	"time"
)

//////////////////////////////////////////////////

func uniqueHex() string {
	return fmt.Sprintf("%016x", uniqueUint64())
}

func uniqueUint64() uint64 {
	v := uint64(time.Now().UnixMilli())
	r := uint64(rand.Uint32()) & 0x3fffff
	v = (v << 22) | r

	return v
}
