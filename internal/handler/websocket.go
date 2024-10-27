package handler

import (
	"bluewave-uptime-agent/internal/metric"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

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
	done := make(chan struct{}) // Channel to signal when the client disconnects

	// Goroutine to handle incoming messages and detect disconnection
	go func() {
		for {
			_, _, err := conn.ReadMessage() // ReadMessage blocks until there's a message or an error
			if err != nil {
				log.Println("Client disconnected:", err)
				close(done) // Signal that the client has disconnected
				return
			}
		}
	}()

	// Streaming messages to the client
	for {
		metrics, metricsErr := metric.GetAllSystemMetrics()
		if metricsErr != nil {
			log.Printf("[FAIL] | Failed to get system metrics: %v", metricsErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get system metrics"})
			return
		}
		data, dataErr := json.Marshal(metrics)
		if dataErr != nil {
			log.Printf("[FAIL] | Failed to marshal system metrics: %v", dataErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal system metrics"})
			return
		}
		select {
		case <-done: // If client disconnects, exit the loop
			log.Println("Stopped streaming due to client disconnect")
			return
		default:
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("Write error:", err)
				return
			}
			time.Sleep(interval) // Simulate real-time data generation
		}
	}
}
