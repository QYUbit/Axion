package axion

import (
	"encoding/binary"

	"github.com/gorilla/websocket"
)

// TODO Better message sending

const (
	MessageMagicNumber uint32 = 0x13E55A7E
)

type WsMessage struct {
	msgType int
	content []byte
}

func newMessage(msgType int, message []byte) WsMessage {
	return WsMessage{
		msgType: msgType,
		content: message,
	}
}

func newTextMesssage(message []byte) WsMessage {
	return WsMessage{
		msgType: websocket.TextMessage,
		content: message,
	}
}

type AxionMessage struct {
	action uint32
	p      []byte
}

func isAxionMessage(b []byte) (AxionMessage, bool) {
	if binary.BigEndian.Uint32(b[:4]) != MessageMagicNumber {
		return AxionMessage{}, false
	}
	return AxionMessage{binary.BigEndian.Uint32(b[4:8]), b[8:]}, true
}
