package nakamoto

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// healthCheck dials an HTTP server and checks if it is running by calling the /health endpoint
func healthCheck(peerUrl string) error {
	// Set a timeout for the connection attempt
	timeout := 1 * time.Second

	// Parse URL and get address and port.
	url_, err := url.Parse(peerUrl)
	if err != nil {
		return err
	}

	// Attempt to establish a TCP connection
	conn, err := net.DialTimeout("tcp", url_.Host, timeout)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()
	return nil
}

func getRandomPort() string {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	portStr := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	fmt.Printf("got random port: %s\n", portStr)
	return portStr
}

func waitForPeersOnline(peers []*PeerCore) {
	ready := make(chan bool, 1)

	go func() {
		for {
			numOnline := 0

			fmt.Printf("waiting for %d peers to come online\n", len(peers)-numOnline)
			for _, peer := range peers {
				fmt.Printf("  pinging peer %s\n", peer.GetLocalAddr())
				// Dial each peer using TCP to check connection.
				err := healthCheck(peer.GetLocalAddr())
				if err == nil {
					numOnline += 1
				} else {
					fmt.Printf("  peer %s is not online: %s\n", peer.GetLocalAddr(), err.Error())
				}
			}

			time.Sleep(500 * time.Millisecond)

			if numOnline == len(peers) {
				ready <- true
				return
			}
		}
	}()

	<-ready
}

func TestStartPeer(t *testing.T) {
	ch := make(chan bool)
	peer1 := NewPeerCore(PeerConfig{address: "127.0.0.1", port: getRandomPort()})
	go peer1.Start()

	// Setup timeout.
	go func() {
		time.Sleep(1500 * time.Millisecond)
		ch <- true
	}()

	<-ch
}

func TestStartPeerHeartbeat(t *testing.T) {
	assert := assert.New(t)

	peer1 := NewPeerCore(PeerConfig{address: "127.0.0.1", port: getRandomPort()})
	peer2 := NewPeerCore(PeerConfig{address: "127.0.0.1", port: getRandomPort()})

	// Override message handler.
	heartbeatChan := make(chan HeartbeatMesage, 1)
	peer1.server.RegisterMesageHandler("heartbeat", func(message []byte) (interface{}, error) {
		t.Logf("Received heartbeat message: %s", message)

		// Decode message into HeartbeatMessage.
		var hb HeartbeatMesage
		if err := json.Unmarshal(message, &hb); err != nil {
			t.Fatalf(err.Error())
			return nil, err
		}

		heartbeatChan <- hb
		return nil, nil
	})

	go peer1.Start()
	go peer2.Start()

	// Wait until both servers are up.
	waitForPeersOnline([]*PeerCore{peer1, peer2})

	// Test bootstrapping.
	t.Log(peer1.GetLocalAddr())
	peer2BootstrapInfo := []string{
		peer1.GetLocalAddr(),
	}

	// Instruct peer 2 to begin bootstrapping.
	t.Logf("Bootstrapping peer 2...")
	peer2.Bootstrap(peer2BootstrapInfo)

	t.Logf("Waiting for peer to receive heartbeat...")

	// Wait for heartbeat.
	select {
	case hb := <-heartbeatChan:
		assert.Equal("heartbeat", hb.Type)
	case <-time.After(15 * time.Second):
		t.Error("Timed out waiting for heartbeat.")
	}

	// Wait for other thread to resume.
	time.Sleep(1 * time.Second)
	// Check peer added to peer list.
	assert.Equal(1, len(peer2.peers))
}

func TestPeerSyncNoReorg(t *testing.T) {}


func TestPeerSyncReorg(t *testing.T) {
	// Two peers for test:
	// - peer A. Has mined a chain of 10 blocks. 
	// - peer B. Has mined an alternative chain of 5 blocks.
	// 
	// Test routine:
	// Peer A starts up, mines 10 blocks.
	// Peer B starts up, mines 5 blocks, then connects to peer B.
	// Peer B syncs with peer A.
	// Peer B downloads the headers of peer A, and then downloads the bodies, ingests the blocks.
	// Assert: peer B tip == peer A tip (longest chain)
	// Assert: peer B state == peer A state (longest chain)
}

func TestPeerBuildState(t *testing.T) {
	// Peer A. Mines chain of 10 blocks.
	// Assert that the state is correctly computed.
}

func TestPeerBuildStateReorg(t *testing.T) {
	// Peer A. Mines chain of 10 blocks.
	// Assert that the state is correctly computed.
}

func TestPeerGossip(t *testing.T) {
	// assert := assert.New(t)
	peer1 := NewPeerCore(PeerConfig{address: "127.0.0.1", port: getRandomPort()})
	go peer1.Start()

	peer2 := NewPeerCore(PeerConfig{address: "127.0.0.1", port: getRandomPort()})
	go peer2.Start()

	// Gossip a block from peer 1 to peer 2.
	// raw := RawBlock{}
}
