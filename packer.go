package pipe

type Packer interface {
	Pack(originData []byte) ([]byte, error)
	Unpack(crypted []byte) ([]byte, error)
}
