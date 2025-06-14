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
	EncryptedReader io.Reader // decrypting reader
	EncryptedWriter io.Writer // encrypting writer
	RawReadWriter   io.ReadWriteCloser
}

func New(conn net.Conn, encKey, decKey, encIV, decIV []byte) (*ProtocolConn, error) {
	// Outgoing (encrypt)
	encBlock, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	encStream := cipher.NewCTR(encBlock, encIV)

	// Incoming (decrypt)
	decBlock, err := aes.NewCipher(decKey)
	if err != nil {
		return nil, err
	}
	decStream := cipher.NewCTR(decBlock, decIV)

	return &ProtocolConn{
		EncryptedReader: &cipher.StreamReader{S: decStream, R: conn},
		EncryptedWriter: &cipher.StreamWriter{S: encStream, W: conn},
		RawReadWriter:   conn,
	}, nil
}

func (pc *ProtocolConn) Read(p []byte) (int, error) {
	var n uint32
	if err := binary.Read(pc.RawReadWriter, binary.BigEndian, &n); err != nil {
		return 0, err
	}
	if int(n) > len(p) {
		return 0, io.ErrShortBuffer
	}
	if _, err := io.ReadFull(pc.EncryptedReader, p[:n]); err != nil {
		return 0, err
	}
	return int(n), nil
}

func (pc *ProtocolConn) Write(p []byte) (int, error) {
	if err := binary.Write(pc.RawReadWriter, binary.BigEndian, uint32(len(p))-4); err != nil {
		return 0, err
	}
	written, err := pc.EncryptedWriter.Write(p[4:])
	if err != nil {
		return written, err
	}
	return 4 + written, nil
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
