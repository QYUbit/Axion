package axion

import (
	"fmt"
	"testing"
)

func TestServer(t *testing.T) {
	server := NewServer(3000)

	server.HandleMessage(func(ctx *MessageContext) {
		fmt.Printf("recieved a message: %s", ctx.Message)
	})
}
