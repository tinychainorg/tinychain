package nakamoto

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/pion/stun"
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

func Bytes32ToHexString(b [32]byte) string {
	return hex.EncodeToString(b[:])
}

func HexStringToBytes32(s string) [32]byte {
	b, _ := hex.DecodeString(s)
	var fbuf [32]byte
	copy(fbuf[:], b)
	return fbuf
}

func HexStringToBigInt(s string) big.Int {
	return Bytes32ToBigInt(HexStringToBytes32(s))
}

func StringToBytes32(s string) [32]byte {
	var b [32]byte
	copy(b[:], s)
	return b
}

func PadBytes(src []byte, length int) []byte {
	if len(src) >= length {
		return src
	}
	padding := make([]byte, length-len(src))
	return append(padding, src...)
}

func DiscoverIP() (string, int, error) {
	// Create a UDP listener
	localAddr := "[::]:0" // Change port if needed
	conn, err := net.ListenPacket("udp", localAddr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP port: %v", err)
		return "", 0, err
	}
	defer conn.Close()
	// localAddr2 := conn.LocalAddr().(*net.UDPAddr)
	// fmt.Printf("Random UDP port: %d\n", localAddr2.Port)
	// fmt.Printf("Listening on %s\n", localAddr)

	// Parse a STUN URI
	u, err := stun.ParseURI("stun:stun.l.google.com:19302")
	if err != nil {
		return "", 0, err
	}

	// Creating a "connection" to STUN server.
	c, err := stun.DialURI(u, &stun.DialConfig{})
	if err != nil {
		return "", 0, err
	}
	// Building binding request with random transaction id.
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	cbChan := make(chan stun.Event, 1)

	// Sending request to STUN server, waiting for response message.
	if err := c.Do(message, func(res stun.Event) {
		cbChan <- res
	}); err != nil {
		return "", 0, err
	}

	// Waiting for response message.
	res := <-cbChan
	if res.Error != nil {
		return "", 0, res.Error
	}
	// Decoding XOR-MAPPED-ADDRESS attribute from message.
	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(res.Message); err != nil {
		return "", 0, res.Error
	}

	// Print the external IP and port
	// peerLogger.Printf("External IP: %s\n", xorAddr.IP)
	// peerLogger.Printf("External Port: %d\n", xorAddr.Port)

	return xorAddr.IP.String(), xorAddr.Port, nil
}

// Constructs a new logger with the given `prefix` and an optional `prefix2`.
//
// Format 1:
// NewLogger("prefix", "")
// 2024/06/30 00:56:06 [prefix] message
//
// Format 2:
// NewLogger("prefix", "prefix2")
// 2024/06/30 00:56:06 [prefix] (prefix2) message
func NewLogger(prefix string, prefix2 string) *log.Logger {
	prefixFull := color.HiGreenString(fmt.Sprintf("[%s] ", prefix))
	if prefix2 != "" {
		prefixFull += color.HiYellowString(fmt.Sprintf("(%s) ", prefix2))
	}
	return log.New(os.Stdout, prefixFull, log.Ldate|log.Ltime|log.Lmsgprefix)
}

func randomNonce() uint64 {
	var nonce uint64
	err := binary.Read(rand.Reader, binary.BigEndian, &nonce)
	if err != nil {
		panic(err)
	}
	return nonce
}
