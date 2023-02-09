package main

import (
	"fmt"
	"github.com/tarik0/GethAuth/server"
	"log"
	"sync"
)

// main entry point.
func main() {
	log.Println(fmt.Sprintf("Starting the websocket server at 8082..."))

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.StartServer()
	}()

	log.Println("Server started! Listening to clients...")
	wg.Wait()
}
