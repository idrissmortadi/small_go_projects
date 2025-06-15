package main

import (
	"flag"
	"log"
	"net"
	"strconv"
	"time"
)

const timeout = 2 * time.Second

func main() {
	log.Println("Starting CLI Port Scanner...")

	// Get flag values
	host := flag.String("host", "0.0.0.0", "Host to scan")
	startPort := flag.Int("start", 1, "Starting port for scanning")
	endPort := flag.Int("end", 1, "End port for scanning")

	if *endPort-*startPort < 0 {
		log.Fatal("End port must be greater or equal than start port")
	}

	flag.Parse()

	log.Printf("Scanning %s from port %d to %d\n", *host, *startPort, *endPort)

	for i := range *endPort - *startPort {
		port := *startPort + i
		address := *host + ":" + strconv.Itoa(port)
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err == nil {
			log.Printf("Port %d open\n", port)
		}
		conn.Close()
	}
}
