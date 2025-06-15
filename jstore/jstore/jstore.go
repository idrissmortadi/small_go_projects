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
	timeout               = 5 * time.Minute
	scannerInitialBufSize = 64 * 1024   // 64KB
	scannerMaxBufSize     = 1024 * 1024 // 1MB
)

type JStore struct {
	Data map[string]string
	mu   sync.RWMutex
}
type Command struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type Response struct {
	Status string `json:"status"`
	Value  string `json:"value,omitempty"`
}

func (js *JStore) Execute(cmd Command) Response {
	switch strings.ToLower(cmd.Op) {
	case "set":
		js.mu.Lock()
		js.Data[cmd.Key] = cmd.Value
		js.mu.Unlock()
		return Response{Status: "success"}
	case "get":
		js.mu.RLock()
		data, exists := js.Data[cmd.Key]
		js.mu.RUnlock()
		if exists {
			return Response{Status: "success", Value: data}
		} else {
			return Response{Status: "error", Value: "key not found"}
		}
	case "delete":
		js.mu.Lock()
		deleted, exists := js.Data[cmd.Key]
		if exists {
			delete(js.Data, cmd.Key)
			response := Response{Status: "success"}
			response.Value = deleted
			js.mu.Unlock()
			return response
		} else {
			js.mu.Unlock()
			return Response{Status: "error", Value: "key not found"}
		}
	default:
		return Response{Status: "error", Value: "unknown operation"}
	}
}

func (js *JStore) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to listen: %v\n", err)
	}
	log.Printf("Listening on %s", addr)

	for {
		log.Println("Waiting for a connection...")
		lconn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go js.handleConnection(lconn)
	}
}

func (js *JStore) handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, scannerInitialBufSize), scannerMaxBufSize)
	for scanner.Scan() {
		var cmd Command
		cmdStr := scanner.Text()
		if err := json.Unmarshal([]byte(cmdStr), &cmd); err != nil {
			response := Response{
				Status: "error",
				Value:  "invalid command format",
			}
			jsonResponse, _ := json.Marshal(response)
			fmt.Fprintln(conn, string(jsonResponse))
			continue
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

func NewJStore() *JStore {
	return &JStore{
		Data: make(map[string]string),
		mu:   sync.RWMutex{},
	}
}
