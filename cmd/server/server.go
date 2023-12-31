package main

import (
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/lesismal/arpc/extension/protocol/websocket"
	"github.com/lesismal/pipe"
	"github.com/lesismal/pipe/cmd/config"
)

func main() {
	key, iv := config.KeyIV()
	packer := &pipe.AESPacker{
		Key: key,
		IV:  iv,
	}

	svrSrc, svrDst := config.ServerAddrs()
	pServer := &pipe.Pipe{
		// Listen: func() (net.Listener, error) {
		// 	return net.Listen("tcp", svrSrc)
		// },
		Listen: func() (net.Listener, error) {
			ln, err := websocket.Listen(svrSrc, nil)
			mux := &http.ServeMux{}
			mux.HandleFunc("/ws", ln.(*websocket.Listener).Handler)
			server := http.Server{
				Addr:    svrSrc,
				Handler: mux,
			}
			go server.ListenAndServe()
			return ln, err
		},
		Dial:    pipe.DialUDP(svrDst),
		Pack:    packer.CBCDecrypt,
		Unpack:  packer.CBCEncrypt,
		Timeout: config.Timeout(),
	}
	pServer.Start()
	defer pServer.Stop()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
}
