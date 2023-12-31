package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net"

	"github.com/lesismal/pipe"
)

func main() {
	key := make([]byte, 32)
	iv := make([]byte, 16)
	rand.Read(key)
	rand.Read(iv)
	packer := &pipe.AESPacker{
		Key: key,
		IV:  iv,
	}
	pClient := &pipe.Pipe{
		Listen: pipe.ListenUDP("localhost:8080"),
		Dial: func() (net.Conn, error) {
			return net.Dial("tcp", "localhost:8081")
		},
		Pack:   packer.CBCEncrypt,
		Unpack: packer.CBCDecrypt,
	}
	pClient.Start()

	pServer := &pipe.Pipe{
		Listen: func() (net.Listener, error) {
			return net.Listen("tcp", "localhost:8081")
		},
		Dial:   pipe.DialUDP("localhost:8082"),
		Pack:   packer.CBCDecrypt,
		Unpack: packer.CBCEncrypt,
	}
	pServer.Start()

	go udpServer("localhost:8082")
	udpClient("localhost:8080")
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
