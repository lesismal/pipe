package protocol

import (
	"fmt"
	"net"
	"net/http"

	"github.com/lesismal/arpc/extension/protocol/websocket"
)

func ListenWebsocket(addr string) func() (net.Listener, error) {
	return func() (net.Listener, error) {
		ln, err := websocket.Listen(addr, nil)
		mux := &http.ServeMux{}
		mux.HandleFunc("/ws", ln.(*websocket.Listener).Handler)
		server := http.Server{
			Addr:    addr,
			Handler: mux,
		}
		go server.ListenAndServe()
		return ln, err
	}
}

func DialWebsocket(addr string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		return websocket.Dial(fmt.Sprintf("ws://%v/ws", addr))
	}
}
