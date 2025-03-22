package axion

import (
	"net/http"

	"github.com/google/uuid"
)

// TODO Multiplexing

type ServerHandlers struct {
	upgradeHandler func(w http.ResponseWriter, r *http.Request, connect func())
	connectHandler func(client *Client, r *http.Request)
}

type Server struct {
	hub        *Hub
	handlers   *ServerHandlers
	httpServer *http.Server
}

// Creates a new Axion instance on the provided http server.
func NewServer(httpServer *http.Server) *Server {
	s := &Server{
		handlers:   new(ServerHandlers),
		httpServer: httpServer,
	}
	s.handlers.upgradeHandler = func(w http.ResponseWriter, r *http.Request, connect func()) { connect() }
	s.handlers.connectHandler = func(client *Client, r *http.Request) {}

	hub := newHub(s)
	s.hub = hub
	go hub.run()

	http.HandleFunc("/ws", hub.handleNewConnection)
	return s
}

// Listen with the underlying http server. Blocks the current go routine.
func (s *Server) ListenAndServe() {
	s.httpServer.ListenAndServe()
}

// Returns all clients.
func (s *Server) Clients() []*Client {
	return s.hub.getClients()
}

// Returns all rooms.
func (s *Server) Rooms() []*Room {
	return s.hub.getRooms()
}

// Reports whether the client with specified id exists and returns it if existing.
func (s *Server) GetClientById(id string) (*Client, bool) {
	client := s.hub.getClientById(id)
	return client, client != nil
}

// Reports whether the room with specified id exists and returns it if existing.
func (s *Server) GetRoomById(id string) (*Room, bool) {
	room := s.hub.getRoomById(id)
	return room, room != nil
}

// Broadcasts a message with the given websocket message type and content to all connected clients.
func (s *Server) Broadcast(msgType int, content []byte) {
	s.hub.broadcastMessage(NewMessage(msgType, content))
}

// Broadcasts a message to all connected clients.
func (s *Server) BroadcastMessage(message WsMessage) {
	s.hub.broadcastMessage(message)
}

// Creates and returns a new empty room.
func (s *Server) CreateRoom() *Room {
	id := uuid.New().String()
	room := newRoom(id, s.hub)

	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()

	s.hub.rooms[id] = room
	return room
}

// Handles incomming upgrade requests (on the path /ws). Call connect to accept the upgrade.
func (s *Server) HandleUpgrade(fun func(w http.ResponseWriter, r *http.Request, connect func())) {
	s.handlers.upgradeHandler = fun
}

// Handles new connected clients
func (s *Server) HandleConnect(fun func(client *Client, r *http.Request)) {
	s.handlers.connectHandler = fun
}
