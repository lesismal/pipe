package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"

	gorilla "github.com/gorilla/websocket"
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
	cliSrc, cliDst := config.ClientAddrs()
	gorilla.DefaultDialer.HandshakeTimeout = config.Timeout()
	pClient := &pipe.Pipe{
		Listen: pipe.ListenUDP(cliSrc),
		// Dial: func() (net.Conn, error) {
		// 	return net.Dial("tcp", cliDst)
		// },
		Dial: func() (net.Conn, error) {
			return websocket.Dial(fmt.Sprintf("ws://%v/ws", cliDst))
		},
		Pack:    packer.CBCEncrypt,
		Unpack:  packer.CBCDecrypt,
		Timeout: config.Timeout(),
	}
	pClient.Start()
	defer pClient.Stop()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
}
