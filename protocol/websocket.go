package protocol

import (
	"fmt"
	"net"
	"net/http"
	"time"

	gorilla "github.com/gorilla/websocket"
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

func DialWebsocket(dstAddr string) func(net.Conn) (net.Conn, error) {
	dialer := &gorilla.Dialer{HandshakeTimeout: time.Second * 10}
	return func(src net.Conn) (net.Conn, error) {
		return websocket.Dial(fmt.Sprintf("ws://%v/ws", dstAddr), dialer)
	}
}

func DialWebsocketWithTimeout(dstAddr string, timeout time.Duration) func(net.Conn) (net.Conn, error) {
	dialer := &gorilla.Dialer{HandshakeTimeout: timeout}
	return func(src net.Conn) (net.Conn, error) {
		return websocket.Dial(fmt.Sprintf("ws://%v/ws", dstAddr), dialer)
	}
}
