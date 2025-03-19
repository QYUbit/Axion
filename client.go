package axion

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan SendMessage
	room *Room
}

func newClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan SendMessage),
		room: nil,
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		msgType, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("readPump error:", err)
			break
		}

		ctx := &MessageContext{
			Message: message,
			client:  c,
		}

		switch msgType {
		case websocket.TextMessage, websocket.BinaryMessage:
			for _, handler := range c.hub.server.messageHandlers {
				handler(ctx)
			}
		case websocket.CloseMessage:
			for _, handler := range c.hub.server.closeHandlers {
				handler(ctx)
			}
			return
		case websocket.PingMessage:
			for _, handler := range c.hub.server.pingHandlers {
				handler(ctx)
			}
		case websocket.PongMessage:
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
