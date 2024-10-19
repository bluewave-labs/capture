package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var data = []byte("BlueWave Uptime Hardware Monitor Agent")
var interval = 2 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Make allowedOrigins Configurable. ENV var?, config?
		allowedOrigins := []string{"*"}

		if allowedOrigins[0] == "*" {
			return true // Accept connections from everywhere
		}

		return false // Decline connections
	},
}

func WebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[FAIL] | Failed to set websocket upgrade: %v", err)
		return
	}

	defer conn.Close()
	for {
		conn.WriteMessage(websocket.TextMessage, data)
		time.Sleep(interval)
	}
}
