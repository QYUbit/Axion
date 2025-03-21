package axion

import (
	axlog "axion/log"
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ClientHandlers struct {
	messageHandlers   []func(message AxionMessage)
	textHandlers      []func(a string)
	binaryHandlers    []func(p []byte)
	closeHandlers     []func(p []byte)
	pingHandlers      []func(p []byte)
	pongHandlers      []func(p []byte)
	disconnectHandler func()
}

type Client struct {
	id       string
	hub      *Hub
	conn     *websocket.Conn
	send     chan WsMessage
	room     *Room
	handlers *ClientHandlers
	ctx      context.Context
}

func newClient(hub *Hub, conn *websocket.Conn) *Client {
	c := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan WsMessage),
		room:     nil,
		id:       uuid.New().String(),
		handlers: new(ClientHandlers),
	}
	c.handlers.disconnectHandler = func() {}
	return c
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		msgType, message, err := c.conn.ReadMessage()
		if err != nil {
			axlog.Logln("readPump error: ", err)
			break
		}
		axlog.Loglf("received message: type: %d, content: %s", msgType, string(message))

		c.conn.PongHandler()

		switch msgType {
		case websocket.BinaryMessage:
			axMessage, ok := isAxionMessage(message)
			if ok {
				for _, handler := range c.hub.server.handlers.messageHandlers {
					handler(axMessage)
				}
				for _, handler := range c.handlers.messageHandlers {
					handler(axMessage)
				}
			} else {
				for _, handler := range c.hub.server.handlers.binaryHandlers {
					handler(message)
				}
				for _, handler := range c.handlers.binaryHandlers {
					handler(message)
				}
			}
		case websocket.TextMessage:
			for _, handler := range c.hub.server.handlers.textHandlers {
				handler(string(message))
			}
			for _, handler := range c.handlers.textHandlers {
				handler(string(message))
			}
		case websocket.CloseMessage:
			for _, handler := range c.hub.server.handlers.closeHandlers {
				handler(message)
			}
			for _, handler := range c.handlers.closeHandlers {
				handler(message)
			}
			return
		case websocket.PingMessage:
			for _, handler := range c.hub.server.handlers.pingHandlers {
				handler(message)
			}
			for _, handler := range c.handlers.pingHandlers {
				handler(message)
			}
		case websocket.PongMessage:
			for _, handler := range c.hub.server.handlers.pongHandlers {
				handler(message)
			}
			for _, handler := range c.handlers.pongHandlers {
				handler(message)
			}
		default:
			c.send <- newTextMesssage(fmt.Appendf(nil, "Error: invalid message type: %d", msgType))
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(60 * time.Second)
	defer func() {
		ticker.Stop()
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(message.msgType, message.content); err != nil {
				log.Println("writePump error:", err)
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) GetId() string {
	return c.id
}

func (c *Client) GetRoom() *Room {
	return c.room
}

func (c *Client) SendMessage(msgType int, message []byte) {
	c.send <- newMessage(msgType, message)
}

func (c *Client) Close() {
	c.hub.unregister <- c
	c.LeaveRoom()
	c.conn.Close()
}

func (c *Client) JoinRoom(roomId string) error {
	room, exists := c.hub.rooms[roomId]
	if !exists {
		return fmt.Errorf("room does not exist")
	}
	room.addClient(c)
	c.room = room
	return nil
}

func (c *Client) LeaveRoom() {
	if c.room == nil {
		return
	}
	c.room.removeClient(c)
	c.room = nil
}

func (c *Client) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Client) WithContext(ctx context.Context) *Client {
	if ctx == nil {
		panic("nil context")
	}
	c2 := new(Client)
	*c2 = *c
	c2.ctx = ctx
	return c2
}

func (c *Client) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

func (c *Client) HandleMessage(fun func(message AxionMessage)) {
	c.handlers.messageHandlers = append(c.handlers.messageHandlers, fun)
}

func (c *Client) HandleText(fun func(a string)) {
	c.handlers.textHandlers = append(c.handlers.textHandlers, fun)
}

func (c *Client) HandleBinary(fun func(p []byte)) {
	c.handlers.binaryHandlers = append(c.handlers.binaryHandlers, fun)
}

func (c *Client) HandleClose(fun func(p []byte)) {
	c.handlers.closeHandlers = append(c.handlers.closeHandlers, fun)
}

func (c *Client) HandlePing(fun func(p []byte)) {
	c.handlers.pingHandlers = append(c.handlers.pingHandlers, fun)
}

func (c *Client) HandlePong(fun func(p []byte)) {
	c.handlers.pongHandlers = append(c.handlers.pongHandlers, fun)
}

func (c *Client) HandleDisconnect(fun func()) {
	c.handlers.disconnectHandler = fun
}
