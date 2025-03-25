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
	r.BroadcastMessage(NewClientJoinedMessage(r.id, client.id))
}

func (r *Room) removeClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.BroadcastMessage(NewClientLeftMessage(r.id, client.id))
	index := slices.Index(r.clients, client)
	r.clients = slices.Delete(r.clients, index, index+1)
}

// Returns the room id.
func (r *Room) Id() string {
	return r.id
}

// Returns all clients in the room
func (r *Room) Members() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients
}

// Broadcasts a message with the given websocket message type and content to all clients in the room.
func (r *Room) Broadcast(msgType int, content []byte) {
	r.broadcast <- NewMessage(msgType, content)
}

// Broadcasts a message to all clients in the room.
func (r *Room) BroadcastMessage(message WsMessage) {
	r.broadcast <- message
}

// Closes the room. Sends a RoomAbandoned message to all members and removes them from the room.
func (r *Room) Close() {
	r.BroadcastMessage(NewRoomAbandonedMessage(r.Id()))

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		c.mu.Lock()
		defer c.mu.Unlock()

		index := slices.Index(c.rooms, r)
		c.rooms = slices.Delete(c.rooms, index, index+1)
	}

	close(r.broadcast)

	r.hub.mu.Lock()
	defer r.hub.mu.Unlock()
	delete(r.hub.rooms, r.id)
}
