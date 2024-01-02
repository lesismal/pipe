package main

import (
	"os"
	"os/signal"

	gorilla "github.com/gorilla/websocket"
	"github.com/lesismal/pipe"
	"github.com/lesismal/pipe/cmd/config"
	"github.com/lesismal/pipe/packer"
	"github.com/lesismal/pipe/protocol"
)

func main() {
	key, iv := config.KeyIV()
	packer := &packer.AESCBC{
		Key: key,
		IV:  iv,
	}
	cliSrc, cliDst := config.ClientAddrs()
	gorilla.DefaultDialer.HandshakeTimeout = config.Timeout()
	pClient := &pipe.Pipe{
		Listen:  protocol.ListenTCP(cliSrc),
		Dial:    protocol.DialWebsocket(cliDst),
		Packer:  packer,
		Timeout: config.Timeout(),
	}
	pClient.StartClient()
	defer pClient.Stop()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
}
