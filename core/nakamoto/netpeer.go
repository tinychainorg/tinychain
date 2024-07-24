package nakamoto

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/liamzebedee/tinychain-go/core"
)

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

// The PeerCore is the local peer for the blockchain network.
// It handles bootstrapping, peer discovery, gossip routines for transactions and blocks.
// It implements the wire protocol for the network, providing API's to send messages to other peers, and callbacks to handle messages sent to us.
type PeerCore struct {
	peers        []Peer
	server       *PeerServer
	config       PeerConfig
	externalIp   string
	externalPort string

	peerId string

	GossipPeersIntervalSeconds int

	OnNewBlock          func(block RawBlock)
	OnNewTransaction    func(tx RawTransaction)
	OnGetBlocks         func(msg GetBlocksMessage) ([][]byte, error)
	OnGetTip            func(msg GetTipMessage) (BlockHeader, error)
	OnSyncGetTipAtDepth func(msg SyncGetTipAtDepthMessage) (SyncGetTipAtDepthReply, error)
	OnSyncGetData       func(msg SyncGetBlockDataMessage) (SyncGetBlockDataReply, error)

	peerLogger log.Logger
}

type Peer struct {
	url           string
	addr          string
	port          string
	lastSeen      uint64
	clientVersion string
}

func (peer *Peer) String() string {
	return peer.url // TODO url not always available.
}

func NewPeerCore(config PeerConfig) *PeerCore {
	wallet, err := core.CreateRandomWallet()
	if err != nil {
		log.Fatalf("Failed to create internal peer keypair: %v", err)
	}

	p := &PeerCore{
		peers:                      []Peer{},
		server:                     nil,
		config:                     config,
		GossipPeersIntervalSeconds: 30,
		peerId:                     wallet.PubkeyStr(),
		peerLogger:                 *NewLogger("peer", fmt.Sprintf(":%s", config.port)),
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

		// Check if this peer is contactable, try to add it to our peers cache.
		// TODO engineer this better.
		p.AddPeer(msg.ClientAddress)

		// Send a heartbeat back.
		heartbeatReply := p.makeHeartbeat()

		return heartbeatReply, nil
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

	p.server.RegisterMesageHandler("new_tx", func(message []byte) (interface{}, error) {
		var msg NewTransactionMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		// Call the OnNewTransaction callback.
		if p.OnNewTransaction != nil {
			p.OnNewTransaction(msg.RawTransaction)
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

		if p.OnGetTip == nil {
			return nil, fmt.Errorf("GetTip callback not set")
		}

		tip, err := p.OnGetTip(msg)
		if err != nil {
			return nil, err
		}

		return GetTipMessage{
			Type: "get_tip",
			Tip:  tip,
		}, nil
	})

	p.server.RegisterMesageHandler("sync_get_tip_at_depth", func(message []byte) (interface{}, error) {
		var msg SyncGetTipAtDepthMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		if p.OnSyncGetTipAtDepth == nil {
			return nil, fmt.Errorf("OnSyncGetTipAtDepth callback not set")
		}

		reply, err := p.OnSyncGetTipAtDepth(msg)
		if err != nil {
			return nil, err
		}

		return reply, nil
	})

	p.server.RegisterMesageHandler("sync_get_data", func(message []byte) (interface{}, error) {
		var msg SyncGetBlockDataMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		if p.OnSyncGetData == nil {
			return nil, fmt.Errorf("SyncGetData callback not set")
		}

		reply, err := p.OnSyncGetData(msg)
		if err != nil {
			return nil, err
		}

		return reply, nil
	})

	p.server.RegisterMesageHandler("gossip_peers", func(message []byte) (interface{}, error) {
		var msg GossipPeersMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		// Ingest new peers.
		havePeers := make(map[string]bool)
		for _, peer := range p.peers {
			havePeers[peer.url] = true
		}
		for _, peerUrl := range msg.Peers {
			if _, ok := havePeers[peerUrl]; !ok {
				go p.AddPeer(peerUrl)
			}
		}

		// Reply with our peers.
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

func (p *PeerCore) Start() {
	go p.statusLoggerRoutine()
	go p.gossipPeersRoutine()

	err := p.server.Start()
	if err != nil {
		panic(fmt.Sprintf("Failed to start peer server: %v", err))
	}
}

func (p *PeerCore) gossipPeersRoutine() {
	for {
		p.peerLogger.Printf("gossip-peers-routine start\n")
		p.GossipPeers()
		p.peerLogger.Printf("gossip-peers-routine complete\n")
		time.Sleep(time.Duration(p.GossipPeersIntervalSeconds) * time.Second)
	}
}

func (p *PeerCore) statusLoggerRoutine() {
	for {
		// Set timeout.
		p.peerLogger.Printf("Connected to %d peers", len(p.peers))
		time.Sleep(30 * time.Second)
	}
}

func (p *PeerCore) GetLocalAddr() string {
	// TODO for now.
	return fmt.Sprintf("http://%s:%s", p.config.address, p.config.port)
}

func (p *PeerCore) GetExternalAddr() string {
	return fmt.Sprintf("http://%s:%s", p.externalIp, p.externalPort)
}

func (p *PeerCore) GetPeers() []Peer {
	return p.peers
}

func (p *PeerCore) GossipBlock(block RawBlock) {
	p.peerLogger.Printf("Gossiping block %s to %d peers\n", block.HashStr(), len(p.peers))

	// Send block to all peers.
	newBlockMsg := NewBlockMessage{
		Type:     "new_block",
		RawBlock: block,
	}
	for _, peer := range p.peers {
		// TODO gossip the block header but not the full block.
		// Let the peer decide on whether they need to download block.
		_, err := SendMessageToPeer(peer.url, newBlockMsg, &p.peerLogger)
		if err != nil {
			p.peerLogger.Printf("Failed to send block to peer: %v", err)
			continue
		}
	}
}

func (p *PeerCore) GossipPeers() {
	p.peerLogger.Printf("Gossiping peers list to %d peers\n", len(p.peers))

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
		reply, err := SendMessageToPeer(peer.url, gossipPeersMsg, &p.peerLogger)
		if err != nil {
			p.peerLogger.Printf("Failed to send block to peer: %v", err)
		}

		// Handle reply.
		var msg GossipPeersMessage
		if err := json.Unmarshal(reply, &msg); err != nil {
			p.peerLogger.Printf("Failed to unmarshal gossip peers reply: %v", err)
			continue
		}

		// Ingest new peers.
		havePeers := make(map[string]bool)
		for _, peer := range p.peers {
			havePeers[peer.url] = true
		}
		for _, peerUrl := range msg.Peers {
			if _, ok := havePeers[peerUrl]; !ok {
				go p.AddPeer(peerUrl)
			}
		}
	}
}

func (p *PeerCore) GetTip(peer Peer) (BlockHeader, error) {
	msg := GetTipMessage{
		Type: "get_tip",
		Tip:  BlockHeader{},
	}
	res, err := SendMessageToPeer(peer.url, msg, &p.peerLogger)
	if err != nil {
		p.peerLogger.Printf("Failed to send block to peer: %v", err)
		return BlockHeader{}, err
	}

	// Decode reply.
	var reply GetTipMessage
	if err := json.Unmarshal(res, &reply); err != nil {
		return reply.Tip, err
	}

	return reply.Tip, nil
}

func (p *PeerCore) SyncGetTipAtDepth(peer Peer, fromBlock [32]byte, depth uint64, dir int) ([32]byte, error) {
	msg := SyncGetTipAtDepthMessage{
		Type:      "sync_get_tip_at_depth",
		FromBlock: fromBlock,
		Depth:     depth,
		Direction: dir,
	}
	res, err := SendMessageToPeer(peer.url, msg, &p.peerLogger)
	if err != nil {
		p.peerLogger.Printf("Failed to send message to peer: %v", err)
		return [32]byte{}, err
	}

	// Decode reply.
	var reply SyncGetTipAtDepthReply
	if err := json.Unmarshal(res, &reply); err != nil {
		return reply.Tip, err
	}

	return reply.Tip, nil
}

func (p *PeerCore) SyncGetBlockData(peer Peer, fromBlock [32]byte, heights core.Bitset, inclHeaders bool, inclBodies bool) (SyncGetBlockDataReply, error) {
	msg := SyncGetBlockDataMessage{
		Type:      "sync_get_data",
		FromBlock: fromBlock,
		Heights:   heights,
		Headers:   inclHeaders,
		Bodies:    inclBodies,
	}
	res, err := SendMessageToPeer(peer.url, msg, &p.peerLogger)
	if err != nil {
		p.peerLogger.Printf("Failed to send message to peer: %v", err)
		return SyncGetBlockDataReply{}, err
	}

	// Decode reply.
	var reply SyncGetBlockDataReply
	if err := json.Unmarshal(res, &reply); err != nil {
		return reply, err
	}

	return reply, nil
}

func (p *PeerCore) HasBlock(peer Peer, blockhash [32]byte) (bool, error) {
	msg := HasBlockMessage{
		Type:      "has_block",
		BlockHash: fmt.Sprintf("%x", blockhash),
	}
	res, err := SendMessageToPeer(peer.url, msg, &p.peerLogger)
	if err != nil {
		p.peerLogger.Printf("Failed to send block to peer: %v", err)
		return false, err
	}

	// Decode reply.
	var reply HasBlockReply
	if err := json.Unmarshal(res, &reply); err != nil {
		return reply.Has, err
	}

	return reply.Has, nil
}

// Bootstraps the connection to the network.
func (p *PeerCore) Bootstrap(peerInfos []string) {
	// Contact all peers and exchange heartbeats.
	p.peerLogger.Println("Bootstrapping network from peers...")

	doneChan := make(chan bool, len(peerInfos))

	// Contact all peers and exchange heartbeats.
	for i, peerInfo := range peerInfos {
		p.peerLogger.Printf("Connecting to bootstrap peer #%d at %s\n", i, peerInfo)

		go (func() {
			p.AddPeer(peerInfo)
			doneChan <- true
		})()
	}

	// Wait for all peers to finish.
	for i := 0; i < len(peerInfos); i++ {
		<-doneChan
	}

	p.peerLogger.Println("Bootstrapping complete.")
}

func (p *PeerCore) makeHeartbeat() HeartbeatMesage {
	heartbeatMsg := HeartbeatMesage{
		Type:                "heartbeat",
		TipHash:             "",
		TipHeight:           0,
		ClientVersion:       CLIENT_VERSION,
		WireProtocolVersion: WIRE_PROTOCOL_VERSION,
		ClientAddress:       p.GetExternalAddr(),
		Time:                time.Now(),
		PeerId:              p.peerId,
	}
	return heartbeatMsg
}

func (p *PeerCore) AddPeer(peerInfo string) {
	// Check URL valid.
	_, err := url.Parse(peerInfo)
	if err != nil {
		p.peerLogger.Println("Failed to parse peer address: ", err)
		return
	}

	peer := Peer{
		url: peerInfo,
		// addr: url.Hostname(),
		// port: url.Port(),
		lastSeen:      0,
		clientVersion: "",
	}

	heartbeatMsg := p.makeHeartbeat()

	if peer.url == p.GetExternalAddr() || peer.url == p.GetLocalAddr() {
		// Skip self.
		p.peerLogger.Printf("AddPeer found peerInfo corresponding to our peer. Skipping.\n")
		return
	}

	// Send heartbeat message to peer.
	res, err := SendMessageToPeer(peer.url, heartbeatMsg, &p.peerLogger)
	if err != nil {
		p.peerLogger.Printf("Failed to send heartbeat to peer: %v", err)
		return
	}

	// Decode response.
	var heartbeatReply HeartbeatMesage
	if err := json.Unmarshal(res, &heartbeatReply); err != nil {
		p.peerLogger.Printf("Failed to unmarshal heartbeat reply: %v", err)
		return
	}

	p.peerLogger.Println("Peer is alive")

	// Now we check if this is our peer.
	if heartbeatReply.PeerId == p.peerId {
		p.peerLogger.Printf("AddPeer found peerInfo corresponding to our peer. Skipping.\n")
		return
	}

	// Add peer to list.
	p.peers = append(p.peers, peer)

	// Print.
	p.peerLogger.Printf("Added peer url=%s\n", peer.url)
}
