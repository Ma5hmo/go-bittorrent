package connection

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"net"
)

type EncryptedConnection struct {
	cipher.StreamReader
	cipher.StreamWriter
}

// WrapConnWithAES wraps a net.Conn with AES-CTR encryption using the given key and IV.
func WrapConnWithAES(conn net.Conn, key, iv []byte) (*EncryptedConnection, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, iv)
	return &EncryptedConnection{
		cipher.StreamReader{S: stream, R: conn},
		cipher.StreamWriter{S: stream, W: conn},
	}, nil
}

// GenerateRandomKeyIV generates a random AES key and IV.
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
