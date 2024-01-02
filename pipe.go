package pipe

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Pipe struct {
	mux sync.Mutex

	running  bool
	ln       net.Listener
	conns    map[net.Conn]net.Conn
	isServer bool

	Listen         func() (net.Listener, error)
	Dial           func() (net.Conn, error)
	Packer         Packer
	Timeout        time.Duration
	ReadBufferSize int
}

func (p *Pipe) StartServer() error {
	return p.start(true)
}

func (p *Pipe) StartClient() error {
	return p.start(false)
}

func (p *Pipe) start(isServer bool) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.running {
		return nil
	}

	p.running = true
	p.isServer = isServer

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
	if p.ReadBufferSize > 32768 {
		p.ReadBufferSize = 32768
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

	if p.isServer {
		go func() {
			defer closePipe()
			log.Printf("[svr] [dst remote %v -> dst local %v -> src local %v -> src remote %v] copying...", dstRemoteAddr, dstLocalAddr, srcLocalAddr, srcRemoteAddr)
			nCopy, err := p.copyRawToFragment(src, dst)
			log.Printf("[svr] [dst remote %v -> dst local %v -> src local %v -> src remote %v, %v coppied] done: %v", dstRemoteAddr, dstLocalAddr, srcLocalAddr, srcRemoteAddr, nCopy, err)
		}()
		log.Printf("[svr] [src remote %v -> src local %v -> dst local %v -> dst remote %v] copying...", srcRemoteAddr, srcLocalAddr, dstLocalAddr, dstRemoteAddr)
		nCopy, err := p.copyFragmentToRaw(dst, src)
		log.Printf("[svr] [src remote %v -> src local %v -> dst local %v -> dst remote %v, %v coppied] done: %v", srcRemoteAddr, srcLocalAddr, dstLocalAddr, dstRemoteAddr, nCopy, err)
	} else {
		go func() {
			defer closePipe()
			log.Printf("[cli] [dst remote %v -> dst local %v -> src local %v -> src remote %v] copying...", dstRemoteAddr, dstLocalAddr, srcLocalAddr, srcRemoteAddr)
			nCopy, err := p.copyFragmentToRaw(src, dst)
			log.Printf("[cli] [dst remote %v -> dst local %v -> src local %v -> src remote %v, %v coppied] done: %v", dstRemoteAddr, dstLocalAddr, srcLocalAddr, srcRemoteAddr, nCopy, err)
		}()
		log.Printf("[cli] [src remote %v -> src local %v -> dst local %v -> dst remote %v] copying...", srcRemoteAddr, srcLocalAddr, dstLocalAddr, dstRemoteAddr)
		nCopy, err := p.copyRawToFragment(dst, src)
		log.Printf("[cli] [src remote %v -> src local %v -> dst local %v -> dst remote %v, %v coppied] done: %v", srcRemoteAddr, srcLocalAddr, dstLocalAddr, dstRemoteAddr, nCopy, err)
	}
}

func (p *Pipe) copyRawToFragment(dst, src net.Conn) (int64, error) {
	var (
		err       error
		nread     int
		ncopy     int64
		buffer    = make([]byte, p.ReadBufferSize)
		packet    []byte
		srcReader = src // bufio.NewReader(src)
		dstWriter = dst // bufio.NewWriter(dst)
		pack      func([]byte) ([]byte, error)
	)
	if p.Packer != nil {
		pack = p.Packer.Pack
	}
	for {
		if p.Timeout > 0 {
			src.SetReadDeadline(time.Now().Add(p.Timeout))
		}
		nread, err = srcReader.Read(buffer)
		if err != nil {
			goto Exit
		}
		if pack != nil {
			packet, err = pack(buffer[:nread])
			if err != nil {
				goto Exit
			}
		} else {
			packet = buffer[:nread]
		}
		_, err = p.writeFragment(dstWriter, packet)
		if err != nil {
			goto Exit
		}
		ncopy += int64(nread)
	}

Exit:
	return ncopy, err
}

func (p *Pipe) copyFragmentToRaw(dst, src net.Conn) (int64, error) {
	var (
		err       error
		nread     int
		ncopy     int64
		srcReader = src // bufio.NewReader(src)
		pack      func([]byte) ([]byte, error)
	)
	if p.Packer != nil {
		pack = p.Packer.Unpack
	}
	for {
		if p.Timeout > 0 {
			src.SetReadDeadline(time.Now().Add(p.Timeout))
		}
		b, err := p.readFragment(srcReader)
		if err != nil {
			goto Exit
		}
		nread = len(b)
		if pack != nil {
			b, err = pack(b)
			if err != nil {
				goto Exit
			}
		}
		_, err = dst.Write(b)
		if err != nil {
			goto Exit
		}
		ncopy += int64(nread)
	}

Exit:
	return ncopy, err
}

func (p *Pipe) readFragment(src io.Reader) ([]byte, error) {
	head := make([]byte, 2)
	_, err := io.ReadFull(src, head)
	if err != nil {
		return nil, err
	}

	l := binary.LittleEndian.Uint16(head)
	b := make([]byte, l)
	_, err = io.ReadFull(src, b)
	if err != nil {
		return nil, err
	}
	return b, err
}

func (p *Pipe) writeFragment(dst io.Writer, b []byte) (int, error) {
	nTotal := 0
	head := make([]byte, 2)
	binary.LittleEndian.PutUint16(head, uint16(len(b)))

	n1, err := dst.Write(head[:])
	if n1 > 0 {
		nTotal = n1
	}
	if err != nil {
		return nTotal, err
	}

	n2, err := dst.Write(b)
	if n2 > 0 {
		nTotal += n2
	}
	return nTotal, err
}
