package client

import (
	"github.com/gorilla/websocket"
	"log"
	"net/url"
)

// NewClient generates new client.
func NewClient(internalRpc string) (*websocket.Conn, error) {
	// The url.
	u, err := url.Parse(internalRpc)
	if err != nil {
		log.Fatalln("unable to parse rpc:", err)
	}

	// Connect to the RPC.
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("unable to dial:", err)
	}

	return c, err
}
