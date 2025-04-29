package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

type messageID uint8

const (
	MsgChoke         messageID = 0
	MsgUnchoke       messageID = 1
	MsgInterested    messageID = 2
	MsgNotInterested messageID = 3
	MsgHave          messageID = 4
	MsgBitfield      messageID = 5
	MsgRequest       messageID = 6
	MsgPiece         messageID = 7
	MsgCancel        messageID = 8
)

// Message stores ID and payload of a message
type Message struct {
	ID      messageID
	Payload []byte
}

func FormatRequest(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: MsgRequest, Payload: payload}
}

func FormatHave(index int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))
	return &Message{ID: MsgHave, Payload: payload}
}

// Read parses a message from a stream. Returns `nil` on keep-alive message
func Read(r io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lengthBuf)

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	messageBuf := make([]byte, length)
	_, err = io.ReadFull(r, messageBuf)
	if err != nil {
		return nil, err
	}

	m := Message{
		ID:      messageID(messageBuf[0]),
		Payload: messageBuf[1:],
	}

	return &m, nil
}

// Serialize serializes a message into a buffer of the form
// <length prefix><message ID><payload>
// Interprets `nil` as a keep-alive message
func (m *Message) Serialize() []byte {
	if m == nil {
		return make([]byte, 4)
	}
	length := uint32(len(m.Payload) + 1) // +1 for id
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)
	return buf
}

// returns the index recieved from the message
func (m *Message) ParseHave() (int, error) {
	if m.ID != MsgHave {
		return 0, fmt.Errorf("expected message ID have, instead got: %d", m.ID)
	}
	if len(m.Payload) != 4 {
		return 0, fmt.Errorf("have message not in length 4")
	}
	index := int(binary.BigEndian.Uint32(m.Payload))
	return index, nil
}

func (m *Message) ParsePiece(index int, buf []byte) (int, error) {
	if m.ID != MsgPiece {
		return 0, fmt.Errorf("expected message ID piece, instead got: %d", m.ID)
	}
	if len(m.Payload) < 8 {
		return 0, fmt.Errorf("piece message length is less than 8: %d", len(m.Payload))
	}
	parsedIndex := int(binary.BigEndian.Uint32(m.Payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("parsed piece index %d doesnt match expected index %d", parsedIndex, index)
	}
	start := int(binary.BigEndian.Uint32(m.Payload[4:8]))
	if start >= len(buf) {
		return 0, fmt.Errorf("receieved start offset too high - %d >= %d", start, len(buf))
	}
	data := m.Payload[8:]
	if start+len(data) > len(buf) {
		return 0, fmt.Errorf("data too long (%d) for offset %d with length %d", len(data), start, len(buf))
	}
	copy(buf[start:], data)
	return len(data), nil
}
