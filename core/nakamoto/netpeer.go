package nakamoto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"time"
)

var peerLogger = NewLogger("peer")
var peerServerLogger = NewLogger("peer-server")
var CLIENT_VERSION = "tinychain v0.0.0 / aggressive alpha"
var WIRE_PROTOCOL_VERSION = uint(1)

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

type PeerServer struct {
	config          PeerConfig
	messageHandlers map[string]PeerMessageHandler
}

func NewPeerServer(config PeerConfig) *PeerServer {
	return &PeerServer{
		config:          config,
		messageHandlers: make(map[string]PeerMessageHandler),
	}
}

type PeerMessageHandler = func(message []byte) (interface{}, error)

func (s *PeerServer) RegisterMesageHandler(messageKey string, handler PeerMessageHandler) {
	s.messageHandlers[messageKey] = handler
}

func (s *PeerServer) Start() {
	// Get the port from the environment variable
	port := s.config.port
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.Handle("/peerapi/inbox", http.HandlerFunc(s.inboxHandler))

	// Configure server with no transfer limits and gracious timeouts
	server := &http.Server{
		Addr:         "[::]:" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	peerServerLogger.Printf("Peer server listening on http://0.0.0.0:%s\n", port)

	// Log all handlers on one line separated by commas.
	peerServerLogger.Printf("Registered message handlers: %v\n", func() []string {
		handlers := make([]string, 0, len(s.messageHandlers))
		for k := range s.messageHandlers {
			handlers = append(handlers, k)
		}
		sort.Strings(handlers)
		return handlers
	}())

	if err := server.ListenAndServe(); err != nil {
		peerServerLogger.Println("Error starting server:", err)
	}
}

// Response struct for JSON response
type Response struct {
	Status string `json:"status"`
}

// Handler for /peerapi/inbox
func (s *PeerServer) inboxHandler(w http.ResponseWriter, r *http.Request) {
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

	// Check message type.
	if _, ok := payload["type"]; !ok {
		http.Error(w, "Missing 'type' field in payload", http.StatusBadRequest)
		return
	}
	// Log the message type.
	messageType := payload["type"].(string)
	peerServerLogger.Printf("Received '%s' message\n", messageType)

	// Check we have a message handler.
	if _, ok := s.messageHandlers[messageType]; !ok {
		http.Error(w, fmt.Sprintf("No message handler registered for '%s'", messageType), http.StatusInternalServerError)
		return
	}

	// Handle.
	res, err := s.messageHandlers[messageType](body)
	if err != nil {
		http.Error(w, "Failed to process message", http.StatusInternalServerError)
		return
	}

	if res == nil {
		// Send back HTTP 200 OK with empty JSON.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
		return
	} else {
		// Respond.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}

}

func SendMessageToPeer(peerUrl string, message any) ([]byte, error) {
	// Dial on HTTP.
	url := fmt.Sprintf("%s/peerapi/inbox", peerUrl)
	peerLogger.Printf("Sending message to peer at %s\n", url)

	// JSON encode message.
	messageJson, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %v", err)
	}

	// Print json.
	peerLogger.Printf("Sending message: %s\n", messageJson)

	// Create a new HTTP request.
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(messageJson))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers.
	req.Header.Set("Content-Type", "application/json")

	// Send request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Print response and status code.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error in request, status=%d, body=\"%s\"", resp.StatusCode, body)
	}

	return body, nil
}

type PeerCore struct {
	peers        []Peer
	server       *PeerServer
	config       PeerConfig
	externalIp   string
	externalPort string

	OnNewBlock  func(block RawBlock)
	OnGetBlocks func(msg GetBlocksMessage) ([][]byte, error)
	OnGetTip    func(msg GetTipMessage) ([32]byte, error)
}

type Peer struct {
	url           string
	addr          string
	port          string
	lastSeen      uint64
	clientVersion string
}

func NewPeerCore(config PeerConfig) *PeerCore {
	p := &PeerCore{
		peers:  []Peer{},
		server: nil,
		config: config,
	}

	externalIp, _, err := DiscoverIP()
	if err != nil {
		log.Fatalf("Failed to discover external IP: %v", err)
	}
	p.externalIp = externalIp
	// p.externalPort = fmt.Sprintf("%d", externalPort)
	p.externalPort = config.port

	p.server = NewPeerServer(p.config)

	// Message handlers.
	//

	p.server.RegisterMesageHandler("heartbeat", func(message []byte) (interface{}, error) {
		// Decode message into HeartbeatMessage.
		var msg HeartbeatMesage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		return nil, nil
	})

	p.server.RegisterMesageHandler("new_block", func(message []byte) (interface{}, error) {
		var msg NewBlockMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		// Call the OnNewBlock callback.
		if p.OnNewBlock != nil {
			p.OnNewBlock(msg.RawBlock)
		}
		return nil, nil
	})

	p.server.RegisterMesageHandler("get_blocks", func(message []byte) (interface{}, error) {
		var msg GetBlocksMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		if p.OnGetBlocks != nil {
			rawBlocksDatas, err := p.OnGetBlocks(msg)
			if err != nil {
				return nil, err
			}

			return GetBlocksReply{
				Type:          "get_blocks_reply",
				RawBlockDatas: rawBlocksDatas,
			}, nil
		}

		return nil, nil
	})

	p.server.RegisterMesageHandler("get_tip", func(message []byte) (interface{}, error) {
		var msg GetTipMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		if p.OnGetTip != nil {
			tip, err := p.OnGetTip(msg)
			if err != nil {
				return nil, err
			}

			return GetTipMessage{
				Type: "get_tip",
				Tip:  Bytes32ToHexString(tip),
			}, nil
		}

		return nil, nil
	})

	p.server.RegisterMesageHandler("gossip_peers", func(message []byte) (interface{}, error) {
		var msg GossipPeersMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		// Ingest new peers.
		// TODO.

		peers := []string{}
		for _, peer := range p.peers {
			peers = append(peers, peer.url)
		}

		return GossipPeersMessage{
			Type:  "gossip_peers",
			Peers: peers,
		}, nil
	})

	return p
}

type NetworkMessage struct {
	type_ string `json:"type"`
}

func (p *PeerCore) Start() {
	go p.StatusLogger()
	p.server.Start()
}

func (p *PeerCore) StatusLogger() {
	for {
		// Set timeout.
		peerLogger.Printf("Connected to %d peers", len(p.peers))
		time.Sleep(30 * time.Second)
	}
}

func (p *PeerCore) GetLocalAddr() string {
	// TODO for now.
	return fmt.Sprintf("http://%s:%s", "[::]", p.config.port)
}

func (p *PeerCore) GetExternalAddr() string {
	return fmt.Sprintf("http://%s:%s", p.externalIp, p.externalPort)
}

func (p *PeerCore) GossipBlock(block RawBlock) {
	peerLogger.Printf("Gossiping block %s to %d peers\n", block.HashStr(), len(p.peers))

	// Send block to all peers.
	newBlockMsg := NewBlockMessage{
		Type:     "new_block",
		RawBlock: block,
	}
	for _, peer := range p.peers {
		_, err := SendMessageToPeer(peer.url, newBlockMsg)
		if err != nil {
			peerLogger.Printf("Failed to send block to peer: %v", err)
		}
	}
}

func (p *PeerCore) GossipPeers() {
	peerLogger.Printf("Gossiping peers list to %d peers\n", len(p.peers))

	// Send list to all peers.
	peers := []string{}
	for _, peer := range p.peers {
		peers = append(peers, peer.url)
	}
	gossipPeersMsg := GossipPeersMessage{
		Type:  "gossip_peers",
		Peers: peers,
	}

	for _, peer := range p.peers {
		_, err := SendMessageToPeer(peer.url, gossipPeersMsg)
		if err != nil {
			peerLogger.Printf("Failed to send block to peer: %v", err)
		}
		// TODO handle peers_reply.
	}
}

// func (p *PeerCore) GetBlocks(block RawBlock) {
//     peerLogger.Printf("Asking peers for blocks\n", block.HashStr(), len(p.peers))

//     // Send block to all peers.
//     for _, peer := range p.peers {
//         newBlockMsg := NewBlockMessage{
//             Type: "new_block",
//             RawBlock: block,
//         }
//         _, err := SendMessageToPeer(peer.url, newBlockMsg)
//         if err != nil {
//             peerLogger.Printf("Failed to send block to peer: %v", err)
//         }
//     }
// }

// Bootstraps the connection to the network.
func (p *PeerCore) Bootstrap(peerInfos []string) {
	// Contact all peers and exchange heartbeats.
	peerLogger.Println("Bootstrapping network from peers...")

	// Contact all peers and exchange heartbeats.
	for i, peerInfo := range peerInfos {
		peerLogger.Printf("Connecting to bootstrap peer #%d at %s\n", i, peerInfo)

		// Check URL valid.
		_, err := url.Parse(peerInfo)
		if err != nil {
			peerLogger.Println("Failed to parse peer address: ", err)
			continue
		}

		peer := Peer{
			url: peerInfo,
			// addr: url.Hostname(),
			// port: url.Port(),
			lastSeen:      0,
			clientVersion: "",
		}

		heartbeatMsg := HeartbeatMesage{
			Type:                "heartbeat",
			TipHash:             "",
			TipHeight:           0,
			ClientVersion:       CLIENT_VERSION,
			WireProtocolVersion: WIRE_PROTOCOL_VERSION,
			ClientAddress:       p.GetExternalAddr(),
			Time:                time.Now(),
		}

		if peer.url == p.GetExternalAddr() || peer.url == p.GetLocalAddr() {
			// Skip self.
			peerLogger.Printf("Bootstrap found peerInfo corresponding to our peer. Skipping #%d.\n", i)
			continue
		}

		// Send heartbeat message to peer.
		_, err = SendMessageToPeer(peer.url, heartbeatMsg)
		if err != nil {
			peerLogger.Printf("Failed to send heartbeat to peer: %v", err)
			continue
		}

		peerLogger.Println("Peer is alive, adding to peer list")

		// Add peer to list.
		p.peers = append(p.peers, peer)
	}

	peerLogger.Println("Bootstrapping complete.")
}
