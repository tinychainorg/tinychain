package nakamoto

import (
	"math/big"
	"time"
	"encoding/hex"
)

func Timestamp() uint64 {
	now := time.Now()
	milliseconds := now.UnixMilli()
	return uint64(milliseconds)
}

func BigIntToBytes32(i big.Int) (fbuf [32]byte) {
	buf := make([]byte, 32)
	i.FillBytes(buf)
	copy(fbuf[:], buf)
	return fbuf
}

func Bytes32ToBigInt(b [32]byte) big.Int {
	return *new(big.Int).SetBytes(b[:])
}

func Bytes32ToString(b [32]byte) string {
	sl := b[:]
	return hex.EncodeToString(sl)
}