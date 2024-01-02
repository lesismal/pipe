package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lesismal/pipe"
	"github.com/lesismal/pipe/cmd/config"
	"github.com/lesismal/pipe/packer"
	"github.com/lesismal/pipe/protocol"
)

func main() {
	websocket.DefaultDialer.HandshakeTimeout = config.Timeout()

	key := make([]byte, 32)
	iv := make([]byte, 16)
	rand.Read(key)
	rand.Read(iv)
	packer := &packer.AESCBC{
		Key: key,
		IV:  iv,
	}

	cliSrc, cliDst := config.ClientAddrs()
	svrSrc, svrDst := config.ServerAddrs()

	pClient := &pipe.Pipe{
		Listen: protocol.ListenUDP(cliSrc),
		// Dial:    protocol.DialWebsocket(cliDst),
		Dial:    protocol.WithWritingDstAddr(cliDst, svrDst, protocol.DialWebsocket),
		Packer:  packer,
		Timeout: config.Timeout(),
	}
	pClient.StartClient()
	defer pClient.Stop()

	pServer := &pipe.Pipe{
		Listen: protocol.ListenWebsocket(svrSrc),
		// Dial:    protocol.DialUDP(svrDst),
		Dial:    protocol.WithReadingDstAddr(protocol.DialUDP),
		Packer:  packer,
		Timeout: config.Timeout(),
	}
	pServer.StartServer()
	defer pServer.Stop()

	go udpServer(svrDst)
	udpClient(cliSrc)
}

func udpServer(addr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("[UDP Server] ResolveUDPAddr failed: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("[UDP Server] ListenUDP failed: %v", err)
	}

	buf := make([]byte, 4096)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatalf("[UDP Server] ReadFromUDP failed: %v", err)
		}
		log.Printf("[UDP Server] ReadFromUDP[%v]: %v", addr.String(), string(buf[:n]))

		n, err = conn.WriteToUDP(buf[:n], addr)
		if err != nil {
			log.Fatalf("[UDP Server] WriteToUDP failed: %v", err)
		}
		log.Printf("[UDP Server] WriteToUDP[%v]: %v", addr.String(), string(buf[:n]))
	}
}

func udpClient(addr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("[UDP Client] ResolveUDPAddr failed: %v", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatalf("[UDP Client] DialUDP failed: %v", err)
	}

	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		data := []byte(fmt.Sprintf("hello from client %v", i))
		n, err := conn.Write(data)
		if err != nil {
			log.Fatalf("[UDP Client] Write failed: %v", err)
		}
		log.Printf("[UDP Client] Write: %v", string(data[:n]))

		n, err = conn.Read(data)
		if err != nil {
			log.Fatalf("[UDP Client] Write failed: %v", err)
		}
		log.Printf("[UDP Client] Read: %v", string(data[:n]))
	}
}
