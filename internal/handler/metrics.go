package handler

import (
	"bluewave-uptime-agent/internal/metric"

	"github.com/gin-gonic/gin"
)

func Metrics(c *gin.Context) {
	metrics, metricsErr := metric.GetAllSystemMetrics()

	if metricsErr != nil {
		c.Status(500)
	}

	c.JSON(200, metrics)
}
