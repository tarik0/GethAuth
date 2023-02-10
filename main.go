package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/tarik0/GethAuth/server"
	"log"
	"sync"
)

// main entry point.
func main() {
	// Load env.
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}

	log.Println(fmt.Sprintf("Starting the websocket server at 8082..."))

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.StartServer()
	}()

	log.Println("Server started! Listening to clients...")
	log.Println("WS RPC  : ws://0.0.0.0:8082/geth?auth=API_KEY")
	log.Println("HTTP RPC: http://0.0.0.0:8082/geth?auth=API_KEY")
	wg.Wait()
}
