package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	timeout    = 500 * time.Millisecond
	maxWorkers = 100
)

func scanPort(wg *sync.WaitGroup, host string, port int, results chan<- int) {
	defer wg.Done()
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err == nil {
		conn.Close()
		results <- port
	}
}

func main() {
	host := flag.String("host", "localhost", "Host to scan")
	startPort := flag.Int("start", 1, "Start port")
	endPort := flag.Int("end", 1024, "End port")
	flag.Parse()

	if *endPort < *startPort {
		fmt.Println("End port must be greater or equal to start port")
		return
	}

	fmt.Printf("Scanning %s ports %d-%d...\n", *host, *startPort, *endPort)

	var wg sync.WaitGroup
	results := make(chan int, maxWorkers)

	// Limit concurrency with a buffered channel
	sem := make(chan struct{}, maxWorkers)

	for port := *startPort; port <= *endPort; port++ {
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(p int) {
			defer func() { <-sem }() // release
			scanPort(&wg, *host, p, results)
		}(port)
	}

	// Close results when all work is done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	openPorts := []int{}
	for port := range results {
		openPorts = append(openPorts, port)
	}

	fmt.Printf("Open ports on %s:\n", *host)
	if len(openPorts) == 0 {
		fmt.Println("None found.")
	}
	for _, port := range openPorts {
		fmt.Printf("  - %d open\n", port)
	}
}
