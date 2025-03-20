package axion

import (
	"fmt"

	"github.com/gorilla/websocket"
)

const ErrorMagicNumber int32 = 0xE3303

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

type MessageContext struct {
	Message []byte
	client  *Client
}

func (ctx *MessageContext) SendMessage(msgType int, message []byte) {
	ctx.client.send <- newMessage(msgType, message)
}

func (ctx *MessageContext) Broadcast(msgType int, message []byte) {
	ctx.client.hub.broadcast <- newMessage(msgType, message)
}

func (ctx *MessageContext) BroadcastRoom(msgType int, message []byte) {
	ctx.client.room.broadcast <- newMessage(msgType, message)
}

func (ctx *MessageContext) SendPong(message []byte) {
	ctx.client.send <- newMessage(websocket.PongMessage, message)
}

func (ctx *MessageContext) GetRoomId() string {
	return ctx.client.room.id
}

func (ctx *MessageContext) JoinRoom(roomId string) error {
	room, exists := ctx.client.hub.rooms[roomId]
	if !exists {
		return fmt.Errorf("room does not exist")
	}
	room.clients = append(room.clients, ctx.client)
	ctx.client.room = room
	return nil
}

func (ctx *MessageContext) LeaveRoom() {
	ctx.client.leaveRoom()
}
