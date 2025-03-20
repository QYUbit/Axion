package axion

import (
	axlog "axion/log"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Setenv("ENV_MODE", "dev")

	go func() {
		server := NewServer(8080)

		server.HandleMessage(func(ctx *MessageContext) {
			axlog.Loglf("recieved a message: %s", ctx.Message)
			ctx.SendMessage(1, []byte("test response"))
		})

		server.HandlePing(func(ctx *MessageContext) {
			axlog.Loglf("received ping: %s", ctx.Message)
			ctx.SendPong([]byte{})
		})

		server.HandleClose(func(ctx *MessageContext) {
			axlog.Loglf("received client close")
		})
	}()

	time.Sleep(2 * time.Minute)

	os.Exit(0)
}
