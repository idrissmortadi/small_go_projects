package jstore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	CONN_TIMEOUT             = 5 * time.Minute
	DEFAULT_CLEANUP_INTERVAL = 10 * time.Second
	SCANNER_INIT_BUFFER_SIZE = 64 * 1024   // 64KB
	SCANNER_MAX_BUFFER_SIZE  = 1024 * 1024 // 1MB
	DEFAULT_TTL              = 60          // Default TTL in seconds
)

type StoreItem struct {
	Value     string
	ExpiresAt int64
}

type JStore struct {
	Data map[string]StoreItem
	mu   sync.RWMutex
}

type Command struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
	TTL   int64  `json:"ttl,omitempty"`
}

type Response struct {
	Status string `json:"status"`
	Value  string `json:"value,omitempty"`
}

func (js *JStore) Execute(cmd Command) Response {
	switch strings.ToLower(cmd.Op) {
	case "set":
		if cmd.TTL < 0 {
			return Response{Status: "error", Value: "ttl must be greater than or equal to 0"}
		}
		js.mu.Lock()
		// Set to default TTL if not specified
		if cmd.TTL == 0 {
			cmd.TTL = DEFAULT_TTL
		}
		js.Data[cmd.Key] = StoreItem{
			Value:     cmd.Value,
			ExpiresAt: time.Now().Unix() + cmd.TTL,
		}
		js.mu.Unlock()
		return Response{Status: "success"}
	case "get":
		js.mu.RLock()
		data, exists := js.Data[cmd.Key]
		// inside case "get"
		now := time.Now().Unix()
		if data.ExpiresAt > 0 && data.ExpiresAt < now {
			js.mu.Lock()
			delete(js.Data, cmd.Key)
			js.mu.Unlock()
			return Response{Status: "error", Value: "key not found"}
		}
		js.mu.RUnlock()
		if !exists {
			return Response{Status: "error", Value: "key not found"}
		}
		return Response{Status: "success", Value: data.Value}
	case "delete":
		js.mu.Lock()
		defer js.mu.Unlock()
		if _, exists := js.Data[cmd.Key]; !exists {
			return Response{Status: "error", Value: "key not found"}
		}
		delete(js.Data, cmd.Key)
		return Response{Status: "success"}
	default:
		return Response{Status: "error", Value: "unknown operation"}
	}
}

func (js *JStore) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to listen: %v\n", err)
		return err
	}
	log.Printf("Listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go js.handleConnection(conn)
	}
}

func (js *JStore) handleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, SCANNER_INIT_BUFFER_SIZE), SCANNER_MAX_BUFFER_SIZE)
	for scanner.Scan() {
		conn.SetDeadline(time.Now().Add(CONN_TIMEOUT))
		var cmd Command
		cmdStr := scanner.Text()
		if err := json.Unmarshal([]byte(cmdStr), &cmd); err != nil {
			response := Response{
				Status: "error",
				Value:  "invalid command format",
			}
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				log.Println("Error marshalling error response:", err)
				fmt.Fprintln(conn, "{\"status\":\"error\",\"value\":\"internal error\"}")
			}
			fmt.Fprintln(conn, string(jsonResponse))
			continue // to next command
		}
		response := js.Execute(cmd)
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			log.Println("Error marshalling response:", err)
			return
		}
		fmt.Fprintln(conn, string(jsonResponse))
	}
	if scanner.Err() != nil {
		log.Println("Error reading from connection:", scanner.Err())
	}
}

func (js *JStore) backgroundCleanup(cleanupInterval time.Duration) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		js.mu.Lock()
		now := time.Now().Unix()
		for key, item := range js.Data {
			if item.ExpiresAt > 0 && item.ExpiresAt < now {
				delete(js.Data, key)
			}
		}
		js.mu.Unlock()
	}
}

func NewJStore() *JStore {
	js := &JStore{
		Data: make(map[string]StoreItem),
		mu:   sync.RWMutex{},
	}
	go js.backgroundCleanup(DEFAULT_CLEANUP_INTERVAL)
	return js
}
