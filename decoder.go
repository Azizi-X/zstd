package decoders

type Decoder interface {
	Decompress([]byte) ([]byte, error)
	Close()
}
