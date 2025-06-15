package main

import (
	"net"
	"strconv"
	"sync"
	"testing"
)

func startTestServer(t *testing.T, port int) net.Listener {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	return ln
}

func TestScanPort_OpenAndClosed(t *testing.T) {
	openPort := 12345
	closedPort := 12346

	ln := startTestServer(t, openPort)
	defer ln.Close()

	var wg sync.WaitGroup
	results := make(chan int, 2)

	wg.Add(2)
	go scanPort(&wg, "localhost", openPort, results)
	go scanPort(&wg, "localhost", closedPort, results)
	wg.Wait()
	close(results)

	found := map[int]bool{}
	for port := range results {
		found[port] = true
	}

	if !found[openPort] {
		t.Errorf("Expected open port %d to be found", openPort)
	}
	if found[closedPort] {
		t.Errorf("Did not expect closed port %d to be found", closedPort)
	}
}
