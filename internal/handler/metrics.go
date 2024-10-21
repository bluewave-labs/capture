package handler

import (
	"bluewave-uptime-agent/internal/metric"

	"github.com/gin-gonic/gin"
)

func Metrics(c *gin.Context) {
	metrics, metricsErr := metric.GetAllSystemMetrics()
	if metricsErr != nil {
		c.JSON(500, "Unable to get metrics")
		return
	}

	c.JSON(200, metrics)
	return
}

func MetricsCPU(c *gin.Context) {
	metrics, metricsErr := metric.CollectCpuMetrics()
	if metricsErr != nil {
		c.JSON(500, "Unable to get metrics")
		return
	}

	c.JSON(200, metrics)
	return
}

func MetricsMemory(c *gin.Context) {
	metrics, metricsErr := metric.CollectMemoryMetrics()
	if metricsErr != nil {
		c.JSON(500, "Unable to get metrics")
		return
	}

	c.JSON(200, metrics)
	return
}

func MetricsDisk(c *gin.Context) {
	metrics, metricsErr := metric.CollectDiskMetrics()
	if metricsErr != nil {
		c.JSON(500, "Unable to get metrics")
		return
	}

	c.JSON(200, metrics)
	return
}

func MetricsHost(c *gin.Context) {
	metrics, metricsErr := metric.GetHostInformation()
	if metricsErr != nil {
		c.JSON(500, "Unable to get metrics")
		return
	}

	c.JSON(200, metrics)
	return
}
