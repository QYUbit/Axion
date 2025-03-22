package axion

import (
	axlog "axion/log"
	"encoding/binary"
	"fmt"

	"github.com/gorilla/websocket"
)

// TODO Better message sending

const (
	BroadCastMessage = 0xB80ADCA5
	RoomMessage      = 0x5E14D300
	JoinRoomMessage  = 0x10114300
	LeaveRoomMessage = 0x1EAFE300
	OpenRoomMessage  = 0x09E14300
	CloseRoomMessage = 0xC105E300
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

func (c *Client) readMessage() error {
	msgType, message, err := c.conn.ReadMessage()
	if err != nil {
		return err
	}
	axlog.Loglf("received message: type: %d, content: %s", msgType, string(message))

	switch msgType {
	case websocket.BinaryMessage:
		c.readBinaryMessage(message)
	case websocket.TextMessage:
		for _, handler := range c.handlers.textHandlers {
			handler(string(message))
		}
	case websocket.CloseMessage:
		for _, handler := range c.handlers.closeHandlers {
			handler(message)
		}
	case websocket.PingMessage:
		for _, handler := range c.handlers.pingHandlers {
			handler(message)
		}
	case websocket.PongMessage:
		for _, handler := range c.handlers.pongHandlers {
			handler(message)
		}
	default:
		c.send <- newTextMesssage(fmt.Appendf(nil, "Error: invalid message type: %d", msgType))
	}
	return nil
}

func (c *Client) readBinaryMessage(p []byte) {
	special := p[:4]
	rest := p[4:]

	switch int(binary.BigEndian.Uint32(special)) {
	case BroadCastMessage:
		if len(c.handlers.broadcastHandlers) == 0 {
			c.hub.broadcastMessage(0, rest)
		}
		for _, handler := range c.handlers.broadcastHandlers {
			handler(p)
		}
	case RoomMessage:
		if len(c.handlers.roomMessageHandlers) == 0 {
			c.room.Broadcast(0, rest)
		}
		for _, handler := range c.handlers.roomMessageHandlers {
			handler(p)
		}
	case JoinRoomMessage:
		if len(c.handlers.joinHandlers) == 0 {
			room, exists := c.hub.server.GetRoomById(string(rest))
			if !exists {
				axlog.Loglf("room does not exist")
			}
			c.JoinRoom(room)
		}
		for _, handler := range c.handlers.joinHandlers {
			handler(p)
		}
	case LeaveRoomMessage:
		if len(c.handlers.leaveHandlers) == 0 {
			c.LeaveRoom()
		}
		for _, handler := range c.handlers.leaveHandlers {
			handler(p)
		}
	case OpenRoomMessage:
		if len(c.handlers.openRoomHandlers) == 0 {
			room := c.hub.server.CreateRoom()
			if rest[0] != 0 {
				c.JoinRoom(room)
			}
		}
		for _, handler := range c.handlers.openRoomHandlers {
			handler(p)
		}
	case CloseRoomMessage:
		if len(c.handlers.closeRoomHandlers) == 0 {
			c.room.Close()
		}
		for _, handler := range c.handlers.closeRoomHandlers {
			handler(p)
		}
	default:
		for _, handler := range c.handlers.binaryHandlers {
			handler(p)
		}
	}
}
