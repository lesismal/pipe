package main

import (
	"os"
	"os/signal"

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

	svrSrc, svrDst := config.ServerAddrs()
	pServer := &pipe.Pipe{
		Listen:  protocol.ListenWebsocket(svrSrc),
		Dial:    protocol.DialTCP(svrDst),
		Packer:  packer,
		Timeout: config.Timeout(),
	}
	pServer.StartServer()
	defer pServer.Stop()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
}
