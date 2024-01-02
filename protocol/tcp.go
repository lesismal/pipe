package protocol

import (
	"net"
)

func ListenTCP(addr string) func() (net.Listener, error) {
	return func() (net.Listener, error) {
		return net.Listen("tcp", addr)
	}
}

func DialTCP(addr string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		return net.Dial("tcp", addr)
	}
}
