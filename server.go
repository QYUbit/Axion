package axion

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// TODO Add documentation

type ServerHandlers struct {
	upgradeHandler   func(w http.ResponseWriter, r *http.Request, connect func())
	connectHandler   func(client *Client, r *http.Request)
	listeningHandler func()
}

type Server struct {
	hub      *Hub
	handlers *ServerHandlers
}

func NewServer(port int) *Server {
	s := &Server{
		handlers: new(ServerHandlers),
	}
	s.handlers.upgradeHandler = func(w http.ResponseWriter, r *http.Request, connect func()) { connect() }
	s.handlers.connectHandler = func(client *Client, r *http.Request) {}
	s.handlers.listeningHandler = func() {}

	hub := newHub(s)
	s.hub = hub
	go hub.run()

	http.HandleFunc("/ws", hub.handleNewConnection)

	go func() {
		s.handlers.listeningHandler()
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	}()
	return s
}

func (s *Server) GetClientById(id string) (*Client, bool) {
	client := s.hub.getClientById(id)
	return client, client != nil
}

func (s *Server) GetRoomById(id string) (*Room, bool) {
	room := s.hub.getRoomById(id)
	return room, room != nil
}

func (s *Server) Broadcast(msgType int, content []byte) {
	s.hub.broadcastMessage(NewMessage(msgType, content))
}

func (s *Server) BroadcastMessage(message WsMessage) {
	s.hub.broadcastMessage(message)
}

func (s *Server) CreateRoom() *Room {
	id := uuid.New().String()
	room := newRoom(id, s.hub)

	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()

	s.hub.rooms[id] = room
	return room
}

func (s *Server) HandleUpgrade(fun func(w http.ResponseWriter, r *http.Request, connect func())) {
	s.handlers.upgradeHandler = fun
}

func (s *Server) HandleConnect(fun func(client *Client, r *http.Request)) {
	s.handlers.connectHandler = fun
}

func (s *Server) HandleListening(fun func()) {
	s.handlers.listeningHandler = fun
}
