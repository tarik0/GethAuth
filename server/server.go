package server

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/tarik0/GethAuth/client"
	"github.com/tarik0/GethAuth/utils"
	"golang.org/x/exp/slices"
	"io"
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

// The http client.
var httpClient = &http.Client{}

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// httpRedirect gets triggered when normal RPC request comes.
func httpRedirect(w http.ResponseWriter, r *http.Request) {
	// Remove some headers.
	redirectedReq, err := http.NewRequest(r.Method, os.Getenv("HTTP_RPC"), r.Body)
	if err != nil {
		http.Error(w, "500 Internal - Something is fucked up.", http.StatusInternalServerError)
		log.Fatalln("Unable to redirect http create:", err)
	}

	// Redirect the http request.
	res, err := httpClient.Do(redirectedReq)
	if err != nil {
		http.Error(w, "500 Internal - Something is fucked up.", http.StatusInternalServerError)
		log.Fatalln("Unable to redirect http req:", err)
	}
	defer res.Body.Close()

	// Remove response headers.
	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	// Copy response.
	w.WriteHeader(res.StatusCode)
	_, err = io.Copy(w, res.Body)
	if err != nil {
		http.Error(w, "500 Internal - Something is fucked up.", http.StatusInternalServerError)
		log.Fatalln("Unable to redirect http req:", err)
	}
}

// serverHandler is the websocket handler.
func serverHandler(w http.ResponseWriter, r *http.Request) {
	// Get parameters.
	params := r.URL.Query()
	if !params.Has("auth") {
		http.Error(w, "403 Forbidden - Unauthorized.", http.StatusForbidden)
		return
	}

	// Get keys list.
	keys, err := utils.ImportKeys("./keys.txt")
	if err != nil {
		http.Error(w, "500 Internal - Something is fucked up.", http.StatusInternalServerError)
		log.Fatalln("Unable to load keys:", err)
		return
	}

	// Check key.
	key := params.Get("auth")
	if !slices.Contains(keys, key) {
		http.Error(w, "400 Bad Request - Invalid scheme.", http.StatusBadRequest)
		return
	}

	// Check websocket.
	upgrade := false
	for _, header := range r.Header["Upgrade"] {
		if header == "websocket" {
			upgrade = true
			break
		}
	}
	if !upgrade {
		httpRedirect(w, r)
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
	internalConn, err := client.NewClient(os.Getenv("WS_RPC"))
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
