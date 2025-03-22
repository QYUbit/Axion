package axion

import (
	axlog "axion/log"
	"context"
	"log"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ClientHandlers struct {
	textHandlers        []func(a string)
	binaryHandlers      []func(p []byte)
	closeHandlers       []func(p []byte)
	pingHandlers        []func(p []byte)
	pongHandlers        []func(p []byte)
	broadcastHandlers   []func(message []byte)
	roomMessageHandlers []func(roomId string, message []byte)
	joinHandlers        []func(roomId string, rest []byte)
	leaveHandlers       []func(roomId string, rest []byte)
	openRoomHandlers    []func(joinAfterwards bool, rest []byte)
	closeRoomHandlers   []func(roomId string, rest []byte)
	disconnectHandler   func()
}

type Client struct {
	id       string
	hub      *Hub
	conn     *websocket.Conn
	send     chan WsMessage
	rooms    []*Room
	handlers *ClientHandlers
	ctx      context.Context
	mu       sync.RWMutex
}

func newClient(hub *Hub, conn *websocket.Conn, id string) *Client {
	c := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan WsMessage),
		rooms:    make([]*Room, 0),
		id:       id,
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

// Returns the id of the client.
func (c *Client) Id() string {
	return c.id
}

// Returns all rooms containing the client.
func (c *Client) Rooms() []*Room {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rooms
}

// Get a room containing the client by id.
func (c *Client) GetRoom(id string) (*Room, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, room := range c.rooms {
		if room.id == id {
			return room, true
		}
	}
	return nil, false
}

// Sends a message with the given websocket message type and content to the client.
func (c *Client) Send(msgType int, content []byte) {
	c.send <- NewMessage(msgType, content)
}

// Sends a message to the client.
func (c *Client) SendMessage(message WsMessage) {
	c.send <- message
}

// Sends a message with the specified close reason and close code. Leaves all rooms and closes the connection.
func (c *Client) Close(code int, reason string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.hub.unregister <- c
	for _, room := range c.rooms {
		c.LeaveRoom(room)
	}
	c.Send(code, []byte(reason))
	c.conn.Close()
}

// Joins the specified room.
func (c *Client) JoinRoom(room *Room) {
	c.mu.Lock()
	defer c.mu.Unlock()

	room.addClient(c)
	c.rooms = append(c.rooms, room)
}

// Leaves the specified room.
func (c *Client) LeaveRoom(room *Room) {
	c.mu.Lock()
	defer c.mu.Unlock()

	room.removeClient(c)
	index := slices.Index(c.rooms, room)
	c.rooms = slices.Delete(c.rooms, index, index+1)
}

// Returns the remote adress of the client.
func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Returns the context attached to the client.
func (c *Client) Context() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

// Sets the context attached to the client.
func (c *Client) SetContext(ctx context.Context) {
	if ctx == nil {
		panic("nil context")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctx = ctx
}

// Triggered when client sends a text message
func (c *Client) HandleText(fun func(a string)) {
	c.handlers.textHandlers = append(c.handlers.textHandlers, fun)
}

// Triggered when client sends a text message (which does not match any special message type)
func (c *Client) HandleBinary(fun func(p []byte)) {
	c.handlers.binaryHandlers = append(c.handlers.binaryHandlers, fun)
}

// Triggerd when the client sends a close message. The close gets executed automatically.
func (c *Client) HandleClose(fun func(p []byte)) {
	c.handlers.closeHandlers = append(c.handlers.closeHandlers, fun)
}

// Triggerd when the client sends a ping message. Pong message will be send automatically.
func (c *Client) HandlePing(fun func(p []byte)) {
	c.handlers.pingHandlers = append(c.handlers.pingHandlers, fun)
}

// Triggerd when the client sends a pong message.
func (c *Client) HandlePong(fun func(p []byte)) {
	c.handlers.pongHandlers = append(c.handlers.pongHandlers, fun)
}

// Triggerd when the client sends a broadcast message. If there are no handlers registered the message gets broadcasted.
func (c *Client) HandleBroadcast(fun func(p []byte)) {
	c.handlers.broadcastHandlers = append(c.handlers.pongHandlers, fun)
}

// Triggerd when the client sends a message to a room. If there are no handlers registered the message gets broadcasted to the room.
func (c *Client) HandleRoomMessage(fun func(roomId string, message []byte)) {
	c.handlers.roomMessageHandlers = append(c.handlers.roomMessageHandlers, fun)
}

// Triggerd when the client sends a JoinRoom message. If there are no handlers registered the client joins automatically.
func (c *Client) HandleJoin(fun func(roomId string, rest []byte)) {
	c.handlers.joinHandlers = append(c.handlers.joinHandlers, fun)
}

// Triggerd when the client sends a LeaveRoom message. If there are no handlers registered the client leaves automatically.
func (c *Client) HandleLeave(fun func(roomId string, rest []byte)) {
	c.handlers.leaveHandlers = append(c.handlers.leaveHandlers, fun)
}

// Triggerd when the client sends a OpenRoom message. If there are no handlers registered the room gets created automatically.
func (c *Client) HandleOpenRoom(fun func(joinAfterwards bool, rest []byte)) {
	c.handlers.openRoomHandlers = append(c.handlers.openRoomHandlers, fun)
}

// Triggerd when the client sends a CloseRoom message. If there are no handlers registered the room gets closed automatically.
func (c *Client) HandleCloseRoom(fun func(roomId string, rest []byte)) {
	c.handlers.closeRoomHandlers = append(c.handlers.closeRoomHandlers, fun)
}

// Triggerd when the client disconnects
func (c *Client) HandleDisconnect(fun func()) {
	c.handlers.disconnectHandler = fun
}
