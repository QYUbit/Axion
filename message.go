package axion

import (
	"fmt"
	"slices"

	"github.com/gorilla/websocket"
)

const ErrorMagicNumber int32 = 0xE3303

type SendMessage struct {
	msgType int
	content []byte
}

func newMessage(msgType int, message []byte) SendMessage {
	return SendMessage{
		msgType: msgType,
		content: message,
	}
}

func newTextMesssage(message []byte) SendMessage {
	return SendMessage{
		msgType: websocket.TextMessage,
		content: message,
	}
}

type MessageContext struct {
	Message []byte
	client  *Client
}

func (ctx *MessageContext) Broadcast(message []byte) {
	ctx.client.hub.broadcast <- message
}

func (ctx *MessageContext) BroadcastRoom(message []byte) {
	ctx.client.room.broadcast <- message
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
	room := ctx.client.room
	if room == nil {
		return
	}
	index := slices.Index(room.clients, ctx.client)
	room.clients = slices.Delete(room.clients, index, index+1)
	ctx.client.room = nil
}
