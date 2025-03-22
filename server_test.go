package axion

import (
	axlog "axion/log"
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

type SomeText string

type ClientMeta struct {
	header http.Header
}

func TestMain(m *testing.M) {
	os.Setenv("ENV_MODE", "dev")

	go func() {
		server := NewServer(&http.Server{Addr: ":8080"})

		server.HandleUpgrade(func(w http.ResponseWriter, r *http.Request, connect func()) {
			connect()
		})

		server.HandleConnect(func(client *Client, r *http.Request) {
			ctx := context.WithValue(client.Context(), SomeText("meta"), ClientMeta{header: r.Header})
			client.SetContext(ctx)

			client.HandleText(func(message string) {
				axlog.Loglf("text message from client %s: %s", client.Id(), message)
			})

			client.HandleClose(func(p []byte) {
				axlog.Loglf("client %s sent close", client.Id())
			})

			client.HandleDisconnect(func() {
				axlog.Loglf("client %s unregistered", client.Id())
			})
		})

		server.ListenAndServe()
	}()

	time.Sleep(2 * time.Minute)

	os.Exit(0)
}
