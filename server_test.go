package axion

import (
	axlog "axion/log"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Setenv("ENV_MODE", "dev")

	go func() {
		server := NewServer(8080)

		server.HandleUpgrade(func(w http.ResponseWriter, r *http.Request, connect func()) {
			connect()
		})

		server.HandleConnect(func(client *Client) {
			client.HandleText(func(message string) {
				axlog.Logln(message)
			})

			client.HandleClose(func(p []byte) {
				axlog.Logln("client left")
			})
		})
	}()

	time.Sleep(2 * time.Minute)

	os.Exit(0)
}
