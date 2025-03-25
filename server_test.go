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

			client.HandleBinary(func(p []byte) {
				axlog.Loglf("message from client %s: %s", client.Id(), p)
			})

			client.HandleBroadcast(func(p []byte) {
				axlog.Logln("recieved broadcast (not allowed)")
			})

			client.HandleOpenRoom(func(joinAfterwards bool, rest []byte) {
				room := server.CreateRoom()
				if joinAfterwards {
					client.JoinRoom(room)
				}
			})

			client.HandleLeave(func(roomId string, rest []byte) {
				room, exists := client.GetRoom(roomId)
				if !exists {
					client.SendMessage(NewClientErrorMessage("room not found"))
					return
				}
				client.LeaveRoom(room)
			})

			client.HandleCloseRoom(func(roomId string, rest []byte) {
				room, exists := client.GetRoom(roomId)
				if !exists {
					client.SendMessage(NewClientErrorMessage("room not found"))
					return
				}
				room.Close()
			})

			client.HandleRoomMessage(func(roomId string, message []byte) {
				room, exists := client.GetRoom(roomId)
				if !exists {
					client.SendMessage(NewClientErrorMessage("room not found"))
					return
				}
				room.Broadcast(0, message)
			})

			client.HandleClose(func(p []byte) {
				axlog.Loglf("close message received from client %s", client.Id())
			})

			client.HandlePing(func(p []byte) {
				axlog.Loglf("ping message received from client %s", client.Id())
			})

			client.HandlePong(func(p []byte) {
				axlog.Loglf("pong message received from client %s", client.Id())
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
