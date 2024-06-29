package nakamoto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"
)

var peerServerLogger = NewLogger("peer-server")

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
