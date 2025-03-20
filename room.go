package axion

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

type Room struct {
	id        string
	broadcast chan WsMessage
	clients   []*Client
}

func newRoom(id string) *Room {
	return &Room{
		id:        id,
		broadcast: make(chan WsMessage),
		clients:   make([]*Client, 0),
	}
}

func generateRoomId() string {
	now := time.Now().UnixMilli()
	timestampBytes := binary.LittleEndian.AppendUint64([]byte{}, uint64(now))

	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x%x", timestampBytes, randomBytes)
}
