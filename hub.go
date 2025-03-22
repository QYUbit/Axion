package axion

import (
	axlog "axion/log"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// TODO Recover sessions

type RegisterClient struct {
	client *Client
	r      *http.Request
}

type Hub struct {
	server     *Server
	clients    map[string]*Client
	rooms      map[string]*Room
	broadcast  chan WsMessage
	register   chan *RegisterClient
	unregister chan *Client
	mu         sync.RWMutex
}

func newHub(server *Server) *Hub {
	return &Hub{
		broadcast:  make(chan WsMessage),
		rooms:      make(map[string]*Room),
		register:   make(chan *RegisterClient),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
		server:     server,
	}
}

func (h *Hub) getClientById(id string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clients[id]
}

func (h *Hub) getRoomById(id string) *Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms[id]
}

func (h *Hub) broadcastMessage(message WsMessage) {
	h.broadcast <- message
}

func (h *Hub) run() {
	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()
			axlog.Loglf("register client %s", reg.client.id)

			h.clients[reg.client.id] = reg.client

			h.server.handlers.connectHandler(reg.client, reg.r)
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.id]; ok {
				axlog.Loglf("unregister client %s", client.id)

				delete(h.clients, client.id)
				close(client.send)

				client.handlers.disconnectHandler()
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for clientID, client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, clientID)
				}
			}
			h.mu.Unlock()
		default:
			h.mu.RLock()
			for _, room := range h.rooms {
				select {
				case message := <-room.broadcast:
					room.mu.RLock()
					for _, client := range room.clients {
						client.send <- message
					}
					room.mu.RUnlock()
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	WriteBufferSize: 1024,
	ReadBufferSize:  1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (hub *Hub) handleNewConnection(w http.ResponseWriter, r *http.Request) {
	connect := func() {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		axlog.Loglf("new client: %s", r.RemoteAddr)

		client := newClient(hub, conn)
		hub.register <- &RegisterClient{client: client, r: r}

		go client.readPump()
		go client.writePump()
	}

	hub.server.handlers.upgradeHandler(w, r, connect)
}
