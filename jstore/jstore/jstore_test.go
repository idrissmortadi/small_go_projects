package jstore

import (
	"bufio"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestJStore_Execute_Set(t *testing.T) {
	js := NewJStore()

	cmd := Command{
		Op:    "set",
		Key:   "test_key",
		Value: "test_value",
	}

	response := js.Execute(cmd)

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Verify the value was actually set
	if js.Data["test_key"].Value != "test_value" {
		t.Errorf("Expected value 'test_value', got '%s'", js.Data["test_key"].Value)
	}
}

func TestJStore_Execute_Delete(t *testing.T) {
	js := NewJStore()

	cmd := Command{
		Op:    "set",
		Key:   "test_key",
		Value: "test_value",
	}

	response := js.Execute(cmd)

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Verify the value was actually set
	if js.Data["test_key"].Value != "test_value" {
		t.Errorf("Expected value 'test_value', got '%s'", js.Data["test_key"].Value)
	}

	cmd = Command{
		Op:  "delete",
		Key: "test_key",
	}

	response = js.Execute(cmd)

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Verify the value was deleted
	if _, exists := js.Data["test_key"]; exists {
		t.Errorf("Expected key 'test_key' to be deleted, but it still exists")
	}
}

func TestJStore_Execute_Get_ExistingKey(t *testing.T) {
	js := NewJStore()
	js.Data["existing_key"] = StoreItem{
		Value:     "existing_value",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}

	cmd := Command{
		Op:  "get",
		Key: "existing_key",
	}

	response := js.Execute(cmd)

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	if response.Value != "existing_value" {
		t.Errorf("Expected value 'existing_value', got '%s'", response.Value)
	}
}

func TestJStore_Execute_Get_NonExistingKey(t *testing.T) {
	js := NewJStore()

	cmd := Command{
		Op:  "get",
		Key: "non_existing_key",
	}

	response := js.Execute(cmd)

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", response.Status)
	}

	if response.Value != "key not found" {
		t.Errorf("Expected value 'key not found', got '%s'", response.Value)
	}
}

func TestJStore_Execute_UnknownOperation(t *testing.T) {
	js := NewJStore()

	cmd := Command{
		Op:  "move",
		Key: "some_key",
	}

	response := js.Execute(cmd)

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", response.Status)
	}

	if response.Value != "unknown operation" {
		t.Errorf("Expected value 'unknown operation', got '%s'", response.Value)
	}
}

func TestJStore_Execute_CaseInsensitive(t *testing.T) {
	js := NewJStore()

	testCases := []string{"SET", "Set", "sEt", "GET", "Get", "gEt"}

	for _, op := range testCases {
		cmd := Command{
			Op:    op,
			Key:   "test_key",
			Value: "test_value",
		}

		response := js.Execute(cmd)

		if strings.ToLower(op) == "set" && response.Status != "success" {
			t.Errorf("Expected success for operation '%s', got '%s'", op, response.Status)
		}

		if strings.ToLower(op) == "get" && response.Status != "error" && response.Status != "success" {
			t.Errorf("Expected success or error for operation '%s', got '%s'", op, response.Status)
		}
	}
}

func TestJStore_ConcurrentAccess(t *testing.T) {
	js := NewJStore()
	var wg sync.WaitGroup

	// Test concurrent writes
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cmd := Command{
				Op:    "set",
				Key:   "key",
				Value: string(rune('a' + i%26)),
			}
			js.Execute(cmd)
		}(i)
	}

	// Test concurrent reads
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := Command{
				Op:  "get",
				Key: "key",
			}
			js.Execute(cmd)
		}()
	}

	wg.Wait()

	// Verify the store is in a consistent state
	cmd := Command{Op: "get", Key: "key"}
	response := js.Execute(cmd)

	if response.Status != "success" && response.Status != "error" {
		t.Errorf("Expected success or error after concurrent access, got '%s'", response.Status)
	}
}

func TestJStore_Execute_SetWithNegativeTTL(t *testing.T) {
	js := NewJStore()
	cmd := Command{
		Op:    "set",
		Key:   "test_key",
		Value: "test_value",
		TTL:   -1, // Negative TTL
	}
	response := js.Execute(cmd)
	if response.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", response.Status)
	}
}

func TestBackgroundCleanup(t *testing.T) {
	js := &JStore{
		Data: make(map[string]StoreItem),
		mu:   sync.RWMutex{},
	}
	js.Data["expired_key"] = StoreItem{
		Value:     "expired_value",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(), // Set to past time
	}
	go js.backgroundCleanup(1 * time.Second)
	time.Sleep(1 * time.Second) // Wait for cleanup to run
	// Verify the expired key is removed
	if _, exists := js.Data["expired_key"]; exists {
		t.Errorf("Expected 'expired_key' to be removed, but it still exists")
	}
}

func TestJStore_Integration(t *testing.T) {
	// Start server in a goroutine
	js := NewJStore()
	addr := ":0" // Use any available port

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	actualAddr := listener.Addr().String()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			go js.handleConnection(conn)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test client connection
	conn, err := net.Dial("tcp", actualAddr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Test SET command
	setCmd := Command{Op: "set", Key: "integration_key", Value: "integration_value", TTL: 60}
	setCmdJSON, _ := json.Marshal(setCmd)

	_, err = conn.Write(append(setCmdJSON, '\n'))
	if err != nil {
		t.Fatalf("Failed to write SET command: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatalf("Failed to read SET response")
	}

	var setResponse Response
	if err := json.Unmarshal([]byte(scanner.Text()), &setResponse); err != nil {
		t.Fatalf("Failed to unmarshal SET response: %v", err)
	}

	if setResponse.Status != "success" {
		t.Errorf("Expected SET success, got '%s'", setResponse.Status)
	}

	// Test GET command
	getCmd := Command{Op: "get", Key: "integration_key"}
	getCmdJSON, _ := json.Marshal(getCmd)

	_, err = conn.Write(append(getCmdJSON, '\n'))
	if err != nil {
		t.Fatalf("Failed to write GET command: %v", err)
	}

	if !scanner.Scan() {
		t.Fatalf("Failed to read GET response")
	}

	var getResponse Response
	if err := json.Unmarshal([]byte(scanner.Text()), &getResponse); err != nil {
		t.Fatalf("Failed to unmarshal GET response: %v", err)
	}

	if getResponse.Status != "success" {
		t.Errorf("Expected GET success, got '%s'", getResponse.Status)
	}

	if getResponse.Value != "integration_value" {
		t.Errorf("Expected value 'integration_value', got '%s'", getResponse.Value)
	}
}

func TestJStore_InvalidJSON(t *testing.T) {
	js := NewJStore()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	actualAddr := listener.Addr().String()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		js.handleConnection(conn)
	}()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", actualAddr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Send invalid JSON
	_, err = conn.Write([]byte("invalid json\n"))
	if err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatalf("Failed to read response")
	}

	var response Response
	if err := json.Unmarshal([]byte(scanner.Text()), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("Expected error status, got '%s'", response.Status)
	}

	if response.Value != "invalid command format" {
		t.Errorf("Expected 'invalid command format', got '%s'", response.Value)
	}
}

func TestNewJStore(t *testing.T) {
	js := NewJStore()

	if js == nil {
		t.Fatal("NewJStore returned nil")
	}

	if js.Data == nil {
		t.Error("Data map is nil")
	}

	if len(js.Data) != 0 {
		t.Errorf("Expected empty Data map, got length %d", len(js.Data))
	}
}
