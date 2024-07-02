package core

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"testing"
)

func uint16ToBytes(n uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, n)
	return b
}

func TestBTTracker(t *testing.T) {
	infohash := [20]byte{0xca, 0xfe, 0xba, 0xbe}
	ip := net.ParseIP("127.0.0.1").To4()
	port := uint16(8080)
	custom_blob := []byte{}
	custom_blob = append(custom_blob, ip...)
	custom_blob = append(custom_blob, uint16ToBytes(port)...)
	// assert length 20 sanity check
	exp_len := 4 + 2
	fmt.Println(len(custom_blob), exp_len)
	if len(custom_blob) != exp_len {
		t.Errorf("Expected length %d, got %d", exp_len, len(custom_blob))
	}

	custom_peer_id := [20]byte{}
	copy(custom_peer_id[:], custom_blob)

	peerID := string(custom_peer_id[:])
	infoHash := string(infohash[:])

	// RunTrackerDemo(
	// 	// string(custom_peer_id[:]),
	// 	string(custom_peer_id_hex),
	// 	string(infohash[:]),
	// )

	// Add peer to swarm
	logger.Println("TEST: Adding peer to swarm")
	err := addPeerToSwarm(peerID, infoHash)
	if err != nil {
		logger.Fatal("Error adding peer to swarm:", err)
	}

	// Get peers for infohash
	// var err error
	logger.Println("TEST: Getting peers")
	resp, err := getPeers(peerID, infoHash)
	if err != nil {
		logger.Fatal("Error getting peers:", err)
	}

	logger.Println("Peers for infohash", infoHash, ":", resp.Peers)
	for i, peer := range resp.Peers {
		// decode peer id
		decoded_peerID := []byte(peer.ID)

		logger.Printf("#%d Peer IP: %s, Port: %d, ID: %s\n", i, peer.IP, peer.Port, hex.EncodeToString(decoded_peerID))
	}
}
