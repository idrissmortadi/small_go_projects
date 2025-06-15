package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	cacheTTL  = 5 * time.Second // Cache TTL for 5 seconds
	rateLimit = 1 * time.Second // Rate limit of 1 request per second
)

type IPData struct {
	lastConnection time.Time
}

var (
	IPLimit       = make(map[string]IPData)
	ResponseCache = make(map[string]TodoCache)
	mu            sync.Mutex // Mutex to protect shared data structures
)

type Todo struct {
	UserID    int    `json:"userId"`
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// Cache with TTL
type TodoCache struct {
	todo      Todo
	Timestamp int64
	ExpiresAt time.Time
}

// handler is an HTTP handler that enforces a simple rate limit per IP address.
// If an IP makes requests more frequently than once per second, it receives a 429 error.
// Otherwise, it responds with "Hello".
func handler(w http.ResponseWriter, r *http.Request) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("Failed to parse remote address: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	now := time.Now().Unix()
	mu.Lock()
	data, exists := IPLimit[ip]
	if exists && time.Since(data.lastConnection) < rateLimit {
		mu.Unlock()
		log.Printf("Rate limit exceeded for IP: %s at %d", ip, now)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	log.Printf("Accepted request from IP: %s at %d", ip, now)
	IPLimit[ip] = IPData{lastConnection: time.Now()}
	mu.Unlock()
	// Fetch data
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 2 && parts[0] == "todos" {
		id := parts[1]
		// Check cache first
		mu.Lock()
		todoCache, found := ResponseCache[id]
		if found {
			// Check if the cache is still valid (TTL of 5 seconds)
			if time.Now().After(todoCache.ExpiresAt) {
				log.Printf("Cache expired for ID: %s", id)
				delete(ResponseCache, id)
			} else {
				todo := todoCache.todo
				mu.Unlock()
				log.Printf("Cache hit for ID: %s", id)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(todo)
				return
			}
		}
		mu.Unlock()
		resp, err := http.Get("https://jsonplaceholder.typicode.com/todos/" + id)
		if err != nil || resp.StatusCode != http.StatusOK {
			if err == nil {
				log.Printf("API returned status: %d", resp.StatusCode)
			} else {
				log.Printf("Can't access API: %v", err)
			}
			http.Error(w, "Failed to fetch todo", http.StatusBadGateway)
			if resp != nil {
				resp.Body.Close()
			}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading content: %v", err)
		}
		todo := Todo{}
		if err := json.Unmarshal(body, &todo); err != nil {
			log.Printf("Error unmarshaling json: %v", err)
			http.Error(w, "Failed to fetch todo", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(todo)
		todoCache = TodoCache{
			todo,
			time.Now().Unix(),
			time.Now().Add(cacheTTL),
		}
		mu.Lock()
		ResponseCache[id] = todoCache
		mu.Unlock()
		return
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Hello"))
		if err != nil {
			log.Printf("Failed to write response: %v", err)
		}
	}
}

func main() {
	log.Println("Starting server on :8080")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
