package protocol

import (
	"net"

	"github.com/lesismal/pipe"
)

func WithReadingDstAddr(dialer func(string) func(net.Conn) (net.Conn, error)) func(net.Conn) (net.Conn, error) {
	return func(src net.Conn) (net.Conn, error) {
		b, err := pipe.ReadFragment(src)
		if err != nil {
			return nil, err
		}
		addr := string(b)
		fDialer := dialer(addr)
		return fDialer(src)
	}
}

func WithWritingDstAddr(proxyAddr, serverAddr string, dialer func(string) func(net.Conn) (net.Conn, error)) func(net.Conn) (net.Conn, error) {
	return func(src net.Conn) (net.Conn, error) {
		fDialer := dialer(proxyAddr)
		dst, err := fDialer(src)
		if err != nil {
			return nil, err
		}
		_, err = pipe.WriteFragment(dst, []byte(serverAddr))
		if err != nil {
			return nil, err
		}
		return dst, err
	}
}
