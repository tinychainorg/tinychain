package nakamoto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeerServerStartFailsForPortInUse(t *testing.T) {
	assert := assert.New(t)

	// Start a server on a random port.
	port := getRandomPort()
	peer1 := NewPeerServer(PeerConfig{address: "127.0.0.1", port: port})
	peer2 := NewPeerServer(PeerConfig{address: "127.0.0.1", port: port})

	errChan := make(chan error)

	// Start the first server.
	go func() {
		err := peer1.Start()
		errChan <- err
	}()

	// Start the second server.
	go func() {
		err := peer2.Start()
		errChan <- err
	}()

	err := <-errChan
	assert.Equal(fmt.Sprintf("listen tcp 127.0.0.1:%s: bind: address already in use", port), err.Error())
}
