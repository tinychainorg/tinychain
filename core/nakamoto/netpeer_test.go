package nakamoto

import (
	"testing"
	"time"
	"net"
	"fmt"
	"encoding/json"
	"github.com/stretchr/testify/assert"
)



// healthCheck dials an HTTP server and checks if it is running by calling the /health endpoint
func healthCheck(address string) error {
    // Set a timeout for the connection attempt
    timeout := 1 * time.Second

    // Attempt to establish a TCP connection
    conn, err := net.DialTimeout("tcp", address, timeout)
    if err != nil {
        return fmt.Errorf("failed to connect: %v", err)
    }
    defer conn.Close()
    return nil
}

func waitForPeersOnline(peers []*PeerCore) {
	ready := make(chan bool)

	go func() {
		goodflags := make([]bool, len(peers))
		for {
			for i, peer := range peers {
				// Dial each peer using TCP to check connection.
				if err := healthCheck(peer.GetLocalAddr()); err != nil {
					goodflags[i] = true
				}
			}

			// Reduce good.
			for _, good := range goodflags {
				if !good {
					break
				}
			}

			ready <- true
		}
	}()

	<-ready
}

func TestStartPeer(t *testing.T) {
	peer1 := NewPeerCore(PeerConfig{port: "8080"})
	peer1.Start()
	// Setup two peers and test one peer sending message to another.
}

func TestStartPeerHeartbeat(t *testing.T) {
	assert := assert.New(t)

	peer1 := NewPeerCore(PeerConfig{port: "8080"})
	go peer1.Start()

	peer2 := NewPeerCore(PeerConfig{port: "8081"})
	go peer2.Start()

	// Wait until both servers are up.
	waitForPeersOnline([]*PeerCore{peer1, peer2})

	// Test bootstrapping.
	t.Log(peer1.GetLocalAddr())
	peer2BootstrapInfo := []string{
		peer1.GetLocalAddr(),
	}

	// Override message handler.
	heartbeatChan := make(chan HeartbeatMesage)
	peer1.server.RegisterMesageHandler("heartbeat", func(message []byte) (interface{}, error) {
        // Decode message into HeartbeatMessage.
        var hb HeartbeatMesage
        if err := json.Unmarshal(message, &hb); err != nil {
            return nil, err
        }

		heartbeatChan <- hb
        return nil, nil
    })

	// Instruct peer 2 to begin bootstrapping.
	go peer2.Bootstrap(peer2BootstrapInfo)

	// Wait for heartbeat.
	select {
	case hb := <-heartbeatChan:
		assert.Equal("heartbeat", hb.Type)
	case <-time.After(5 * time.Second):
		t.Error("Timed out waiting for heartbeat.")
	}

	// Wait for other thread to resume.
	time.Sleep(1 * time.Second)
	// Check peer added to peer list.
	assert.Equal(1, len(peer2.peers))
}

func TestPeerGossip(t *testing.T) {
	// assert := assert.New(t)
	peer1 := NewPeerCore(PeerConfig{port: "8080"})
	go peer1.Start()

	peer2 := NewPeerCore(PeerConfig{port: "8081"})
	go peer2.Start()

	// Gossip a block from peer 1 to peer 2.
	// raw := RawBlock{}
}
