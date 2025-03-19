package axion

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	server     *Server
	clients    map[*Client]bool
	rooms      map[string]*Room
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		rooms:      map[string]*Room{},
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- newTextMesssage(message):
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		default:
			h.mu.Lock()
			for _, room := range h.rooms {
				select {
				case message := <-room.broadcast:
					for _, client := range room.clients {
						client.send <- newTextMesssage(message)
					}
				default:
				}
			}
			h.mu.Unlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	WriteBufferSize: 1024,
	ReadBufferSize:  1024,
}

func (hub *Hub) handleNewConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "failed to connect", http.StatusInternalServerError)
		return
	}

	client := newClient(hub, conn)
	hub.register <- client

	go client.readPump()
	go client.writePump()
}
