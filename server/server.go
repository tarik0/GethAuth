package server

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/tarik0/GethAuth/client"
	"github.com/tarik0/GethAuth/utils"
	"golang.org/x/exp/slices"
	"log"
	"net/http"
	"os"
	"time"
)

// The server upgrader.
var upgrader = websocket.Upgrader{
	HandshakeTimeout:  45 * time.Second,
	EnableCompression: true,
}

// serverHandler is the websocket handler.
func serverHandler(w http.ResponseWriter, r *http.Request) {
	// Get parameters.
	params := r.URL.Query()
	if !params.Has("auth") {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("403 Forbidden - Unauthorized."))
		return
	}

	// Get keys list.
	keys, err := utils.ImportKeys("./keys.txt")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("500 Internal - Something is fucked up."))
		log.Fatalln("Unable to load keys:", err)
		return
	}

	// Check key.
	key := params.Get("auth")
	if !slices.Contains(keys, key) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("403 Forbidden - Unauthorized."))
		return
	}

	// Upgrade the connection.
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Unable to upgrade:", err)
		return
	}
	defer c.Close()

	// Create new client.
	internalConn, err := client.NewClient(os.Args[1])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("500 Internal - Something is fucked up."))
		log.Fatalln("Unable to connect to internal rpc:", err)
		return
	}
	defer internalConn.Close()

	// The internal listener.
	go func() {
		defer internalConn.Close()
		defer c.Close()

		for {
			// Read the internal rpc.
			mt, message, err := internalConn.ReadMessage()
			if err != nil {
				log.Println("internal read err:", err)
				_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, ""))
				break
			}

			// Write to the client.
			err = c.WriteMessage(mt, message)
			if err != nil {
				log.Println("internal write err:", err)
				_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, ""))
				break
			}
		}
	}()

	log.Println(fmt.Sprintf("A client connected with the key: %s", key))

	// Read messages until disconnected.
	for {
		// Read message.
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("server read err:", err)
			_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, ""))
			break
		}

		// Send it to the internal rpc.
		err = internalConn.WriteMessage(mt, message)
		if err != nil {
			log.Println("internal write err:", err)
			_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, ""))
			break
		}
	}
}

// StartServer starts the websocket server.
func StartServer() {
	http.HandleFunc("/geth", serverHandler)
	log.Fatalln(http.ListenAndServe(":8082", nil))
}
