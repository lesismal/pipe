package protocol

import (
	"net"
	"time"
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

func DialTCPWithTimeout(dstAddr string, timeout time.Duration) func(net.Conn) (net.Conn, error) {
	return func(src net.Conn) (net.Conn, error) {
		return net.DialTimeout("tcp", dstAddr, timeout)
	}
}
