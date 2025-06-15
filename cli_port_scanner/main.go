package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"
)

const timeout = 2 * time.Second

func main() {
	fmt.Println("Starting CLI Port Scanner...")

	// Get flag values
	host := flag.String("host", "0.0.0.0", "Host to scan")
	startPort := flag.Int("start", 1, "Starting port for scanning")
	endPort := flag.Int("end", 1, "End port for scanning")
	flag.Parse()

	if *endPort < *startPort {
		fmt.Println("End port must be greater or equal than start port")
		return
	}

	fmt.Printf("Scanning %s from port %d to %d\n", *host, *startPort, *endPort)

	for port := *startPort; port <= *endPort; port++ {
		address := *host + ":" + strconv.Itoa(port)
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			continue
		}
		conn.Close()
		fmt.Printf("\tPort %d open\n", port)
	}
}
