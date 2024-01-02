package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net"

	"github.com/gorilla/websocket"
	"github.com/lesismal/pipe"
	"github.com/lesismal/pipe/cmd/config"
	"github.com/lesismal/pipe/packer"
	"github.com/lesismal/pipe/protocol"
)

func main() {
	key := make([]byte, 32)
	iv := make([]byte, 16)
	rand.Read(key)
	rand.Read(iv)
	packer := &packer.AESCBC{
		Key: key,
		IV:  iv,
	}
	cliSrc, cliDst := config.ClientAddrs()
	websocket.DefaultDialer.HandshakeTimeout = config.Timeout()
	pClient := &pipe.Pipe{
		Listen:  protocol.ListenUDP(cliSrc),
		Dial:    protocol.DialWebsocket(cliDst),
		Packer:  packer,
		Timeout: config.Timeout(),
	}
	pClient.StartClient()
	defer pClient.Stop()

	svrSrc, svrDst := config.ServerAddrs()
	pServer := &pipe.Pipe{
		Listen:  protocol.ListenWebsocket(svrSrc),
		Dial:    protocol.DialUDP(svrDst),
		Packer:  packer,
		Timeout: config.Timeout(),
	}
	pServer.StartServer()
	defer pServer.Stop()

	go udpServer("localhost:18082")
	udpClient("localhost:18080")
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
		// time.Sleep(time.Second / 100)
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
