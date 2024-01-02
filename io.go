package pipe

import (
	"encoding/binary"
	"io"
)

func ReadFragment(src io.Reader) ([]byte, error) {
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

func WriteFragment(dst io.Writer, b []byte) (int, error) {
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
