package main

import (
	js "jstore/jstore"
	"log"
)

func main() {
	jstore := js.NewJStore()
	if err := jstore.ListenAndServe(":8080"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
