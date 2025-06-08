package protocolconn

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
)

type ProtocolConn struct {
	EncryptedReader io.Reader
	EncryptedWriter io.Writer
	RawReadWriter   io.ReadWriter
}

func New(conn net.Conn, key []byte, iv []byte) (*ProtocolConn, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, iv)

	return &ProtocolConn{
		EncryptedReader: cipher.StreamReader{S: stream, R: conn},
		EncryptedWriter: cipher.StreamWriter{S: stream, W: conn},
		RawReadWriter:   conn,
	}, nil
}

func (ec *ProtocolConn) Read(b []byte) (n int, err error) {
	var lengthBytes [4]byte
	_, err = ec.RawReadWriter.Read(lengthBytes[:])
	if err != nil {
		return 0, err
	}
	length := int(binary.BigEndian.Uint32(lengthBytes[:]))
	if length > len(b) {
		return 0, io.ErrShortBuffer
	}
	return ec.EncryptedReader.Read(b[:length])
}

func (ec *ProtocolConn) Write(b []byte) (int, error) {
	var lengthBytes [4]byte
	binary.BigEndian.PutUint32(lengthBytes[:], uint32(len(b)))
	_, err := ec.RawReadWriter.Write(b[0:4])
	if err != nil {
		return 0, err
	}
	n, err := ec.EncryptedWriter.Write(b[4:])
	return n + 4, err // +4 for the length prefix
}

func GenerateRandomKeyIV() (key, iv []byte, err error) {
	key = make([]byte, 32) // AES-256
	iv = make([]byte, aes.BlockSize)
	_, err = rand.Read(key)
	if err != nil {
		return nil, nil, err
	}
	_, err = rand.Read(iv)
	if err != nil {
		return nil, nil, err
	}
	return key, iv, nil
}
