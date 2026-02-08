package web

import (
	"net/http"

	"github.com/gorilla/websocket"
	"sunnyproxy/internal/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	broadcaster *logger.Broadcaster
}

func NewWSHandler() *WSHandler {
	return &WSHandler{
		broadcaster: logger.GetBroadcaster(),
	}
}

func (h *WSHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.broadcaster.AddClient(conn)

	go func() {
		defer h.broadcaster.RemoveClient(conn)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
