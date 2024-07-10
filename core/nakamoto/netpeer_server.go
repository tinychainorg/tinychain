package nakamoto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"time"
)

// PeerServer is an RPC server running over HTTP.
// Peers send messages to http://<host>:<port>/peerapi/inbox and receive response messages.
// All messages are encoded using JSON.
type PeerServer struct {
	config          PeerConfig
	messageHandlers map[string]PeerMessageHandler
	log             log.Logger
	server          *http.Server
}

func NewPeerServer(config PeerConfig) *PeerServer {
	s := PeerServer{
		config:          config,
		messageHandlers: make(map[string]PeerMessageHandler),
		log:             *NewLogger("peer-server", fmt.Sprintf(":%s", config.port)),
	}

	// Get the port from the environment variable
	addr := s.config.address
	port := s.config.port

	// Setup HTTP server mux.
	mux := http.NewServeMux()
	mux.Handle("/peerapi/inbox", http.HandlerFunc(s.inboxHandler))

	// Configure server with no transfer limits and gracious timeouts
	s.server = &http.Server{
		Addr:         addr + ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &s
}

type PeerMessageHandler = func(message []byte) (interface{}, error)

func (s *PeerServer) RegisterMesageHandler(messageKey string, handler PeerMessageHandler) {
	s.log.Printf("Registering message handler for '%s'\n", messageKey)
	s.messageHandlers[messageKey] = handler
}

func (s *PeerServer) Start() error {
	// Log all handlers on one line separated by commas.
	s.log.Printf("Handling message types: %v\n", func() []string {
		handlers := make([]string, 0, len(s.messageHandlers))
		for k := range s.messageHandlers {
			handlers = append(handlers, k)
		}
		sort.Strings(handlers)
		return handlers
	}())

	if err := s.server.ListenAndServe(); err != nil {
		s.log.Printf("Peer server listening on http://%s\n", s.server.Addr)
		s.log.Println("Error starting server:", err)
		return err
	}

	return nil
}

func (s *PeerServer) Stop() {
	s.log.Println("Stopping peer server")
	s.server.Shutdown(context.Background())
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
	s.log.Printf("Received '%s' message\n", messageType)

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

func SendMessageToPeer(peerUrl string, message any, log *log.Logger) ([]byte, error) {
	// Dial on HTTP.
	url := fmt.Sprintf("%s/peerapi/inbox", peerUrl)
	log.Printf("Sending message to peer at %s\n", url)

	// JSON encode message.
	messageJson, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %v", err)
	}

	// Print json.
	log.Printf("Sending message: %s\n", messageJson)

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
