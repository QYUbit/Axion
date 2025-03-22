package axion

import (
	"slices"
	"sync"
)

type Room struct {
	id        string
	hub       *Hub
	broadcast chan WsMessage
	clients   []*Client
	mu        sync.RWMutex
}

func newRoom(id string, hub *Hub) *Room {
	return &Room{
		id:        id,
		hub:       hub,
		broadcast: make(chan WsMessage),
		clients:   make([]*Client, 0),
	}
}

func (r *Room) addClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients = append(r.clients, client)
}

func (r *Room) removeClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	index := slices.Index(r.clients, client)
	r.clients = slices.Delete(r.clients, index, index+1)
}

func (r *Room) GetId() string {
	return r.id
}

func (r *Room) GetMembers() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients
}

func (r *Room) Broadcast(msgType int, content []byte) {
	r.broadcast <- NewMessage(msgType, content)
}

func (r *Room) BroadcastMessage(message WsMessage) {
	r.broadcast <- message
}

func (r *Room) Close() {
	close(r.broadcast)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, client := range r.clients {
		client.send <- NewTextMesssage(("room abandoned"))
	}

	r.hub.mu.Lock()
	defer r.hub.mu.Unlock()
	delete(r.hub.rooms, r.id)
}
