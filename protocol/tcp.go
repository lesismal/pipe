package protocol

import (
	"net"
)

func ListenTCP(addr string) func() (net.Listener, error) {
	return func() (net.Listener, error) {
		return net.Listen("tcp", addr)
	}
}

func DialTCP(dstAddr string) func(net.Conn) (net.Conn, error) {
	return func(src net.Conn) (net.Conn, error) {
		return net.Dial("tcp", dstAddr)
	}
}
