package axion

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type ServerHandlers struct {
	messageHandlers []func(message AxionMessage)
	textHandlers    []func(a string)
	binaryHandlers  []func(p []byte)
	closeHandlers   []func(p []byte)
	pingHandlers    []func(p []byte)
	pongHandlers    []func(p []byte)
	upgradeHandler  func(w http.ResponseWriter, r *http.Request, connect func())
	connectHandler  func(client *Client, r *http.Request)
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

	hub := newHub(s)
	s.hub = hub
	go hub.run()

	http.HandleFunc("/ws", hub.handleNewConnection)

	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	}()
	return s
}

func (s *Server) GetClientByID(id string) (*Client, bool) {
	client, exists := s.hub.clients[id]
	return client, exists
}

func (s *Server) GetRoomById(id string) (*Room, bool) {
	room, exists := s.hub.rooms[id]
	return room, exists
}

func (s *Server) Broadcast(msgType int, message []byte) {
	s.hub.broadcast <- newMessage(msgType, message)
}

func (s *Server) CreateRoom() *Room {
	id := uuid.New().String()
	room := newRoom(id, s.hub)

	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	s.hub.rooms[id] = room

	return room
}

func (s *Server) HandleMessage(fun func(message AxionMessage)) {
	s.handlers.messageHandlers = append(s.handlers.messageHandlers, fun)
}

func (s *Server) HandleText(fun func(a string)) {
	s.handlers.textHandlers = append(s.handlers.textHandlers, fun)
}

func (s *Server) HandleBinary(fun func(p []byte)) {
	s.handlers.binaryHandlers = append(s.handlers.binaryHandlers, fun)
}

func (s *Server) HandleClose(fun func(p []byte)) {
	s.handlers.closeHandlers = append(s.handlers.closeHandlers, fun)
}

func (s *Server) HandlePing(fun func(p []byte)) {
	s.handlers.pingHandlers = append(s.handlers.pingHandlers, fun)
}

func (s *Server) HandlePong(fun func(p []byte)) {
	s.handlers.pongHandlers = append(s.handlers.pongHandlers, fun)
}

func (s *Server) HandleUpgrade(fun func(w http.ResponseWriter, r *http.Request, connect func())) {
	s.handlers.upgradeHandler = fun
}

func (s *Server) HandleConnect(fun func(client *Client, r *http.Request)) {
	s.handlers.connectHandler = fun
}
