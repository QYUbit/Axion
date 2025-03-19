package axion

import (
	"fmt"
	"log"
	"net/http"
)

type Server struct {
	hub             *Hub
	messageHandlers []func(ctx *MessageContext)
	closeHandlers   []func(ctx *MessageContext)
	pingHandlers    []func(ctx *MessageContext)
}

func NewServer(port int) *Server {
	s := &Server{
		messageHandlers: make([]func(ctx *MessageContext), 0),
		closeHandlers:   make([]func(ctx *MessageContext), 0),
		pingHandlers:    make([]func(ctx *MessageContext), 0),
	}

	hub := newHub()
	s.hub = hub
	go hub.run()

	http.HandleFunc("/ws", hub.handleNewConnection)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	return s
}

func (s *Server) HandleMessage(fun func(ctx *MessageContext)) {
	s.messageHandlers = append(s.messageHandlers, fun)
}

func (s *Server) HandleClose(fun func(ctx *MessageContext)) {
	s.closeHandlers = append(s.closeHandlers, fun)
}

func (s *Server) HandlePing(fun func(ctx *MessageContext)) {
	s.pingHandlers = append(s.pingHandlers, fun)
}

func (s *Server) Broadcast(message []byte) {
	s.hub.broadcast <- message
}

func (s *Server) BroadcastRoom(roomId string, message []byte) error {
	room, exists := s.hub.rooms[roomId]
	if !exists {
		return fmt.Errorf("room does not exist")
	}
	room.broadcast <- message
	return nil
}

func (s *Server) CreateRoom() string {
	id := generateRoomId()
	room := newRoom(id)
	s.hub.rooms[id] = room
	return id
}

func (s *Server) RemoveRoom(roomId string) error {
	room, exists := s.hub.rooms[roomId]
	if !exists {
		return fmt.Errorf("room does not exist")
	}

	close(room.broadcast)
	for _, client := range room.clients {
		client.send <- newTextMesssage([]byte("room abandoned"))
	}

	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	delete(s.hub.rooms, roomId)
	return nil
}
