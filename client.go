package axion

import (
	axlog "axion/log"
	"context"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// TODO multiple rooms

type ClientHandlers struct {
	textHandlers        []func(a string)
	binaryHandlers      []func(p []byte)
	closeHandlers       []func(p []byte)
	pingHandlers        []func(p []byte)
	pongHandlers        []func(p []byte)
	broadcastHandlers   []func(p []byte)
	roomMessageHandlers []func(p []byte)
	joinHandlers        []func(p []byte)
	leaveHandlers       []func(p []byte)
	openRoomHandlers    []func(p []byte)
	closeRoomHandlers   []func(p []byte)
	disconnectHandler   func()
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
		if err := c.readMessage(); err != nil {
			axlog.Logln("readPump error: ", err)
			break
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

func (c *Client) JoinRoom(room *Room) {
	room.addClient(c)
	c.room = room
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

func (c *Client) HandleBroadcast(fun func(p []byte)) {
	c.handlers.broadcastHandlers = append(c.handlers.pongHandlers, fun)
}

func (c *Client) HandleRoomMessage(fun func(p []byte)) {
	c.handlers.roomMessageHandlers = append(c.handlers.pongHandlers, fun)
}

func (c *Client) HandleJoin(fun func(p []byte)) {
	c.handlers.joinHandlers = append(c.handlers.pongHandlers, fun)
}

func (c *Client) HandleLeave(fun func(p []byte)) {
	c.handlers.leaveHandlers = append(c.handlers.pongHandlers, fun)
}

func (c *Client) HandleOpenRoom(fun func(p []byte)) {
	c.handlers.openRoomHandlers = append(c.handlers.pongHandlers, fun)
}

func (c *Client) HandleCloseRoom(fun func(p []byte)) {
	c.handlers.closeRoomHandlers = append(c.handlers.pongHandlers, fun)
}

func (c *Client) HandleDisconnect(fun func()) {
	c.handlers.disconnectHandler = fun
}
