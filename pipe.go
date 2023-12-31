package pipe

import (
	"log"
	"net"
	"sync"
	"time"
)

type Pipe struct {
	mux sync.Mutex

	running bool
	ln      net.Listener
	conns   map[net.Conn]net.Conn

	Listen         func() (net.Listener, error)
	Dial           func() (net.Conn, error)
	Pack           func([]byte) ([]byte, error)
	Unpack         func([]byte) ([]byte, error)
	Timeout        time.Duration
	ReadBufferSize int
}

func (p *Pipe) Start() error {
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.running {
		return nil
	}

	p.running = true

	p.initConfig()

	var err error
	p.ln, err = p.Listen()
	if err == nil {
		go p.accept()
	}
	return err
}

func (p *Pipe) Stop() {
	p.mux.Lock()
	defer p.mux.Unlock()
	if !p.running {
		return
	}
	p.running = false

	if p.ln != nil {
		p.ln.Close()
	}

	if p.conns == nil {
		return
	}

	for src, dst := range p.conns {
		src.Close()
		dst.Close()
	}
	p.conns = nil
}

func (p *Pipe) initConfig() {
	if p.Timeout <= 0 {
		p.Timeout = 60 * time.Second
	}
	if p.ReadBufferSize <= 0 {
		p.ReadBufferSize = 4096
	}
	log.Printf("Pipe Start with [timeout: %v seconds, read buffer: %v]", p.Timeout.Seconds(), p.ReadBufferSize)

}

func (p *Pipe) accept() error {
	var err error
	var src net.Conn
	for p.running {
		src, err = p.ln.Accept()
		if err == nil {
			log.Printf("Accept: [local %v, remote %v]", src.LocalAddr(), src.RemoteAddr())
			go p.serve(src)
		}
	}
	return err
}

func (p *Pipe) serve(src net.Conn) {
	dst, err := p.Dial()
	if err != nil {
		log.Printf("[local %v, remote %v] Dial failed: %v", src.LocalAddr(), src.RemoteAddr(), err)
		src.Close()
		return
	}
	log.Printf("[local %v, remote %v] Dial success", src.LocalAddr(), src.RemoteAddr())

	p.mux.Lock()
	if p.conns == nil {
		p.conns = map[net.Conn]net.Conn{}
	}
	p.conns[src] = dst
	p.mux.Unlock()

	closePipe := func() {
		src.Close()
		dst.Close()

		p.mux.Lock()
		delete(p.conns, src)
		p.mux.Unlock()
	}

	srcLocalAddr := src.LocalAddr().String()
	srcRemoteAddr := src.RemoteAddr().String()
	dstLocalAddr := dst.LocalAddr().String()
	dstRemoteAddr := dst.RemoteAddr().String()
	defer closePipe()
	go func() {
		defer closePipe()
		log.Printf("[dst remote %v -> dst local %v -> src local %v -> src remote %v] copying...", dstRemoteAddr, dstLocalAddr, srcLocalAddr, srcRemoteAddr)
		nCopy, err := p.copy(src, dst, p.Unpack)
		log.Printf("[dst remote %v -> dst local %v -> src local %v -> src remote %v, %v coppied] done: %v", dstRemoteAddr, dstLocalAddr, srcLocalAddr, srcRemoteAddr, nCopy, err)
	}()
	log.Printf("[src remote %v -> src local %v -> dst local %v -> dst remote %v] copying...", srcRemoteAddr, srcLocalAddr, dstLocalAddr, dstRemoteAddr)
	nCopy, err := p.copy(dst, src, p.Pack)
	log.Printf("[src remote %v -> src local %v -> dst local %v -> dst remote %v, %v coppied] done: %v", srcRemoteAddr, srcLocalAddr, dstLocalAddr, dstRemoteAddr, nCopy, err)
}

func (p *Pipe) copy(dst, src net.Conn, pack func([]byte) ([]byte, error)) (int64, error) {
	var (
		err    error
		nread  int
		ncopy  int64
		buffer = make([]byte, p.ReadBufferSize)
		packet []byte
	)
	for {
		if p.Timeout > 0 {
			src.SetReadDeadline(time.Now().Add(p.Timeout))
		}
		nread, err = src.Read(buffer)
		if err != nil {
			goto Exit
		}
		if p.Pack != nil {
			packet, err = pack(buffer[:nread])
			if err != nil {
				goto Exit
			}
		} else {
			packet = buffer[:nread]
		}
		_, err = dst.Write(packet)
		if err != nil {
			goto Exit
		}
		ncopy += int64(nread)
	}

Exit:
	return ncopy, err
}
