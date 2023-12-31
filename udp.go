package pipe

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type UDPConn struct {
	isClient bool
	raw      *net.UDPConn
	closed   int32
	chClsoed chan struct{}
	laddr    net.Addr
	raddr    *net.UDPAddr
	fClose   func() error
	chData   chan []byte
	wrote    int32
	chWrote  chan struct{}
	cache    []byte
	parent   *net.UDPConn
	rTimer   *time.Timer
	wTimer   *time.Timer
}

func (conn *UDPConn) Read(b []byte) (n int, err error) {
	if conn.isClient {
		return conn.raw.Read(b)
	}

	if atomic.LoadInt32(&conn.closed) == 1 {
		return 0, io.EOF
	}

	if len(conn.cache) > 0 {
		n = copy(b, conn.cache)
		if n == len(conn.cache) {
			conn.cache = conn.cache[:0]
		} else {
			copy(conn.cache, conn.cache[n:])
			conn.cache = conn.cache[:len(conn.cache)-n]
		}
	}
	if n == len(b) {
		return n, nil
	}
	var ok bool
	var pkt []byte
	if n > 0 {
		select {
		case pkt, ok = <-conn.chData:
			goto CP
		default:
		}
		return n, nil
	}
	pkt, ok = <-conn.chData
	if !ok {
		if n > 0 {
			return n, nil
		}
		return n, io.EOF
	}
CP:
	n2 := copy(b[n:], pkt)
	n += n2
	if n2 < len(pkt) {
		conn.cache = append(conn.cache, pkt[n2:]...)
	}
	return n, nil
}

func (conn *UDPConn) Write(b []byte) (n int, err error) {
	if conn.isClient {
		return conn.raw.Write(b)
	}

	if atomic.CompareAndSwapInt32(&conn.wrote, 0, 1) {
		close(conn.chWrote)
	}
	return conn.parent.WriteToUDP(b, conn.raddr)
}

func (conn *UDPConn) Close() error {
	var err error
	if atomic.CompareAndSwapInt32(&conn.closed, 0, 1) {
		if conn.isClient {
			err = conn.raw.Close()
		}
		if conn.fClose != nil {
			conn.fClose()
		}
		close(conn.chData)
		close(conn.chClsoed)
	}
	return err
}

func (conn *UDPConn) LocalAddr() net.Addr {
	return conn.laddr
}

func (conn *UDPConn) RemoteAddr() net.Addr {
	return conn.raddr
}

func (conn *UDPConn) SetDeadline(t time.Time) error {
	if conn.isClient {
		return conn.raw.SetDeadline(t)
	}
	return nil
}

func (conn *UDPConn) SetReadDeadline(t time.Time) error {
	if conn.isClient {
		return conn.raw.SetReadDeadline(t)
	}
	if conn.rTimer == nil {
		if !t.IsZero() {
			conn.rTimer = time.AfterFunc(time.Until(t), func() {
				conn.Close()
			})
		}
	} else {
		if !t.IsZero() {
			conn.rTimer.Reset(time.Until(t))
		} else {
			conn.rTimer.Stop()
		}
	}
	return nil
}

func (conn *UDPConn) SetWriteDeadline(t time.Time) error {
	if conn.isClient {
		return conn.raw.SetWriteDeadline(t)
	}
	if conn.wTimer == nil {
		if !t.IsZero() {
			conn.wTimer = time.AfterFunc(time.Until(t), func() {
				conn.Close()
			})
		}
	} else {
		if !t.IsZero() {
			conn.wTimer.Reset(time.Until(t))
		} else {
			conn.wTimer.Stop()
		}
	}
	return nil
}

type UDPListener struct {
	mux sync.Mutex

	uc       *net.UDPConn
	ch       chan *UDPConn
	ctx      context.Context
	cancel   func()
	closed   int32
	listened int32
	conns    map[string]*UDPConn
}

func (ln *UDPListener) deleteConn(uc *UDPConn) {
	ln.mux.Lock()
	defer ln.mux.Unlock()
	delete(ln.conns, uc.raddr.String())
}

func (ln *UDPListener) accept() {
	for {
		buf := make([]byte, 2048)
		n, raddr, err := ln.uc.ReadFromUDP(buf)
		if err != nil {
			return
		}
		pkt := buf[:n]
		saddr := raddr.String()
		func() {
			ln.mux.Lock()
			defer ln.mux.Unlock()
			uc, ok := ln.conns[saddr]
			if ok {
				select {
				case uc.chData <- pkt:
				default:
				}
				return
			}
			uc = &UDPConn{
				laddr:    ln.Addr(),
				raddr:    raddr,
				cache:    buf[:n],
				chData:   make(chan []byte, 1024),
				chWrote:  make(chan struct{}),
				chClsoed: make(chan struct{}),
				parent:   ln.uc,
			}
			uc.fClose = func() error {
				ln.deleteConn(uc)
				return nil
			}
			ln.conns[saddr] = uc
			ln.ch <- uc
		}()
	}
}

func (ln *UDPListener) Accept() (net.Conn, error) {
	if atomic.CompareAndSwapInt32(&ln.listened, 0, 1) {
		go ln.accept()
	}
	select {
	case c := <-ln.ch:
		return c, nil
	case <-ln.ctx.Done():
		return nil, io.EOF
	}
}

func (ln *UDPListener) Close() error {
	if atomic.CompareAndSwapInt32(&ln.closed, 0, 1) {
		if ln.uc != nil {
			err := ln.uc.Close()
			ln.uc = nil
			ln.cancel()
			return err
		}
	}
	return nil
}

func (ln *UDPListener) Addr() net.Addr {
	if ln.uc == nil {
		return nil
	}
	return ln.uc.LocalAddr()
}

func ListenUDP(addr string) func() (net.Listener, error) {
	return func() (net.Listener, error) {
		var err error
		var ln = &UDPListener{
			ch:    make(chan *UDPConn, 1024),
			conns: map[string]*UDPConn{},
		}
		ln.ctx, ln.cancel = context.WithCancel(context.Background())
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}
		ln.uc, err = net.ListenUDP("udp", udpAddr)
		if err != nil {
			return nil, err
		}
		return ln, nil
	}
}

func DialUDP(addr string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}

		conn, err := net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			return nil, err
		}

		uc := &UDPConn{
			isClient: true,
			raw:      conn,
			laddr:    conn.LocalAddr(),
			raddr:    udpAddr,
			chData:   make(chan []byte, 1024),
			chWrote:  make(chan struct{}),
			chClsoed: make(chan struct{}),
		}

		return uc, nil
	}
}
