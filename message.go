package axion

import (
	axlog "axion/log"
	"encoding/binary"

	"github.com/gorilla/websocket"
)

const (
	StatusMessage    = 0x57A7513E
	BroadCastMessage = 0xB80ADCA5
	RoomMessage      = 0x5E14D300
	JoinRoomMessage  = 0x10114300
	LeaveRoomMessage = 0x1EAFE300
	OpenRoomMessage  = 0x09E14300
	CloseRoomMessage = 0xC105E300
)

const (
	SigInit          = 0x11411413
	SigClientError   = 0xC11E9E33
	SigServerError   = 0x5E3F3E33
	SigRoomAbandoned = 0xABAD0300
	SigClientLeft    = 0x1EF70300
	SigClientJoined  = 0x101ED300
)

type WsMessage struct {
	msgType int
	content []byte
}

// Creates a new message
func NewMessage(msgType int, message []byte) WsMessage {
	return WsMessage{
		msgType: msgType,
		content: message,
	}
}

// Creates a new text message
func NewTextMesssage(message string) WsMessage {
	return WsMessage{
		msgType: websocket.TextMessage,
		content: []byte(message),
	}
}

// Creates a new binary message
func NewBinaryMessage(message []byte) WsMessage {
	return WsMessage{
		msgType: websocket.BinaryMessage,
		content: message,
	}
}

func NewInitMessage(clientId string) WsMessage {
	var p []byte
	binary.BigEndian.AppendUint32(p, SigInit)
	p = append(p, []byte(clientId)...)
	return WsMessage{
		msgType: 0,
		content: p,
	}
}

// Creates a new ClientError message
func NewClientErrorMessage(message string) WsMessage {
	var p []byte
	binary.BigEndian.AppendUint32(p, SigClientError)
	p = append(p, []byte(message)...)
	return WsMessage{
		msgType: 0,
		content: p,
	}
}

// Creates a new ServerError message
func NewServerErrorMessage(message string) WsMessage {
	var p []byte
	binary.BigEndian.AppendUint32(p, SigServerError)
	p = append(p, []byte(message)...)
	return WsMessage{
		msgType: 0,
		content: p,
	}
}

// Creates a new RoomAbandoned message
func NewRoomAbandonedMessage(roomId string) WsMessage {
	var p []byte
	binary.BigEndian.AppendUint32(p, SigRoomAbandoned)
	p = append(p, []byte(roomId)...)
	return WsMessage{
		msgType: 0,
		content: p,
	}
}

// Creates a new ClientLeftRoom message
func NewClientLeftMessage(roomId string, clientId string) WsMessage {
	var p []byte
	binary.BigEndian.AppendUint32(p, SigClientLeft)
	p = append(p, []byte(roomId)...)
	p = append(p, []byte(clientId)...)
	return WsMessage{
		msgType: 0,
		content: p,
	}
}

// Creates a new ClientJoinedRoom message
func NewClientJoinedMessage(roomId string, clientId string) WsMessage {
	var p []byte
	binary.BigEndian.AppendUint32(p, SigClientJoined)
	p = append(p, []byte(roomId)...)
	p = append(p, []byte(clientId)...)
	return WsMessage{
		msgType: 0,
		content: p,
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
		c.send <- NewClientErrorMessage("invalid message type")
	}
	return nil
}

func (c *Client) readBinaryMessage(p []byte) {
	special := p[:4]
	rest := p[4:]

	switch int(binary.BigEndian.Uint32(special)) {
	case BroadCastMessage:
		if len(c.handlers.broadcastHandlers) == 0 {
			c.hub.broadcastMessage(NewBinaryMessage(rest))
		}
		for _, handler := range c.handlers.broadcastHandlers {
			handler(p)
		}
	case RoomMessage:
		roomId := string(rest[:36])
		message := rest[36:]
		if len(c.handlers.roomMessageHandlers) == 0 {
			room, exists := c.GetRoom(roomId)
			if !exists {
				c.SendMessage(NewClientErrorMessage("room not found"))
				return
			}
			room.Broadcast(0, message)
		}
		for _, handler := range c.handlers.roomMessageHandlers {
			handler(roomId, message)
		}
	case JoinRoomMessage:
		roomId := string(rest[:36])
		if len(c.handlers.joinHandlers) == 0 {
			room, exists := c.hub.server.GetRoomById(roomId)
			if !exists {
				c.SendMessage(NewClientErrorMessage("room not found"))
				return
			}
			c.JoinRoom(room)
		}
		for _, handler := range c.handlers.joinHandlers {
			handler(roomId, rest[36:])
		}
	case LeaveRoomMessage:
		roomId := string(rest[:36])
		if len(c.handlers.leaveHandlers) == 0 {
			room, exists := c.GetRoom(roomId)
			if !exists {
				c.SendMessage(NewClientErrorMessage("room not found"))
				return
			}
			c.LeaveRoom(room)
		}
		for _, handler := range c.handlers.leaveHandlers {
			handler(roomId, rest[36:])
		}
	case OpenRoomMessage:
		joinAfterwards := rest[0] != 0
		if len(c.handlers.openRoomHandlers) == 0 {
			room := c.hub.server.CreateRoom()
			if joinAfterwards {
				c.JoinRoom(room)
			}
		}
		for _, handler := range c.handlers.openRoomHandlers {
			handler(joinAfterwards, rest[1:])
		}
	case CloseRoomMessage:
		roomId := string(rest[:36])
		if len(c.handlers.closeRoomHandlers) == 0 {
			room, exists := c.GetRoom(roomId)
			if !exists {
				c.SendMessage(NewClientErrorMessage("room not found"))
				return
			}
			room.Close()
		}
		for _, handler := range c.handlers.closeRoomHandlers {
			handler(roomId, rest[36:])
		}
	case StatusMessage:
		_ = rest[0] != 0
	default:
		for _, handler := range c.handlers.binaryHandlers {
			handler(p)
		}
	}
}

// Client				Server
//	 |	   Http Upgrade    |
//	 |-------------------->|
//	 |<--------------------|
//	 |					   |
//	 |		   Init		   |
//	 |-------------------->|
//	 |					   |
//	 |		  Status	   |
//	 |<--------------------|
//	 |					   |
