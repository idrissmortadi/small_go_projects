# JStore

A simple, lightweight TCP-based key-value store written in Go with JSON command protocol and TTL support.

## Features

- **TCP Server**: Listens for connections and handles multiple clients concurrently
- **JSON Protocol**: Commands and responses use JSON format
- **TTL Support**: Keys can expire automatically with configurable time-to-live
- **Thread-Safe**: Uses read-write mutexes for concurrent access
- **Background Cleanup**: Automatically removes expired keys
- **Memory-Based**: Fast in-memory storage

## Quick Start

### Installation

```bash
git clone https://github.com/idrissmortadi/small_go_projects.git
cd jstore
go mod init jstore
go run main.go
```

The server will start on port 8080 by default.

### Basic Usage

Connect to the server using any TCP client (telnet, netcat, or custom client):

```bash
telnet localhost 8080
```

Send JSON commands:

```json
{"op":"set","key":"hello","value":"world","ttl":60}
{"op":"get","key":"hello"}
{"op":"delete","key":"hello"}
```

## Commands

### SET

Store a key-value pair with optional TTL (time-to-live in seconds).

```json
{ "op": "set", "key": "mykey", "value": "myvalue", "ttl": 300 }
```

- `key`: String key identifier
- `value`: String value to store
- `ttl`: Optional expiration time in seconds (default: 60)

**Response:**

```json
{ "status": "success" }
```

### GET

Retrieve a value by key.

```json
{ "op": "get", "key": "mykey" }
```

**Response (success):**

```json
{ "status": "success", "value": "myvalue" }
```

**Response (key not found/expired):**

```json
{ "status": "error", "value": "key not found" }
```

### DELETE

Remove a key-value pair.

```json
{ "op": "delete", "key": "mykey" }
```

**Response:**

```json
{ "status": "success" }
```

## Configuration

Default constants can be modified in `jstore/jstore.go`:

- `CONN_TIMEOUT`: Connection timeout (5 minutes)
- `DEFAULT_CLEANUP_INTERVAL`: Background cleanup interval (10 seconds)
- `DEFAULT_TTL`: Default TTL for keys (60 seconds)
- `SCANNER_INIT_BUFFER_SIZE`: Initial buffer size (64KB)
- `SCANNER_MAX_BUFFER_SIZE`: Maximum buffer size (1MB)

## Testing

Run the test suite:

```bash
go test ./jstore
```

Run tests with verbose output:

```bash
go test -v ./jstore
```

## Architecture

- **JStore**: Main struct containing the data map and mutex
- **Command**: JSON structure for client commands
- **Response**: JSON structure for server responses
- **StoreItem**: Internal structure storing values with expiration timestamps

## Example Client (Go)

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "net"
)

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

func main() {
    conn, err := net.Dial("tcp", "localhost:8080")
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // Set a value
    cmd := Command{Op: "set", Key: "test", Value: "hello world", TTL: 60}
    data, _ := json.Marshal(cmd)
    fmt.Fprintf(conn, "%s\n", data)

    // Read response
    scanner := bufio.NewScanner(conn)
    if scanner.Scan() {
        var resp Response
        json.Unmarshal([]byte(scanner.Text()), &resp)
        fmt.Printf("Response: %+v\n", resp)
    }
}
```
