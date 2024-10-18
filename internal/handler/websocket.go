package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var data = []byte("BlueWave Uptime Hardware Monitor Agent")
var interval = 2 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func WebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	defer conn.Close()
	for {
		conn.WriteMessage(websocket.TextMessage, data)
		time.Sleep(interval)
	}
}
