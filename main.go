package main

import (
	"hlc21/client"
	"log"
	"os"
	"runtime"
)

func main() {
	address := os.Getenv("ADDRESS")
	port := os.Getenv("Port")

	if port == "" {
		port = os.Getenv("PORT")
	}

	log.Printf("CPUs: %d\n", runtime.NumCPU())
	log.Printf("ADDRESS: %s\n", address)
	log.Printf("Port: %s\n", port)

	port = "8000"

	httpClient := client.NewClient(address, port)
	game := client.NewGamePool(httpClient)

	if err := game.Run(); err != nil {
		log.Printf("ERROR: %s\n", err)
	}
}
