package nakamoto

import (
	"fmt"
	"net/http"
    "encoding/json"
    "io/ioutil"
    "time"
)



// Bootstrap by connecting to peers.
// Fill your peer cache with 20 peers max.
// Do routines:
// - regular heartbeat every 30s to each peer, checking current tip. Insert into DB as last_seen, client_version, ip/port.
// - regular bootstrap, find peers in network. our cache can be large - 1000 peers big.
// Perform sync routine:
// - interactive bissect to find common ancestor. Then download missing blocks.
// - download missing blocks from peers.
// Then provide services:
// - serve blocks to peers.
// - gossip/relay txs to peers.
// Then hook up with node:
// - on receiving a new block, we ingest it. then we restart miner process.
// - on receiving new txs, we add them to our block. and restart miner process.

// What does the peer interface look like?
// DialPeer
// SendMsg
// RecvMsg

type PeerServer struct {
	config PeerConfig
}

type PeerConfig struct {
	port string
}

func NewPeerServer(config PeerConfig) *PeerServer {
	return &PeerServer{
		config: config,
	}
}

func (s *PeerServer) Start() {
	// Get the port from the environment variable
	port := s.config.port
    if port == "" {
        port = "8080"
    }

    mux := http.NewServeMux()
    mux.Handle("/peerapi/inbox", http.HandlerFunc(inboxHandler))

    // Configure server with no transfer limits and gracious timeouts
    server := &http.Server{
        Addr:         "[::]:" + port,
        Handler:      mux,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    fmt.Printf("Starting server on \n\t[::]:%s \n\thttp://0.0.0.0:%s\n", port, port)
    if err := server.ListenAndServe(); err != nil {
        fmt.Println("Error starting server:", err)
    }
}


// Response struct for JSON response
type Response struct {
    Status  string `json:"status"`
}


// Handler for /peerapi/inbox
func inboxHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }

    var payload map[string]interface{}
    if err := json.Unmarshal(body, &payload); err != nil {
        http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
        return
    }

    response := Response{
        Status:  "success",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

