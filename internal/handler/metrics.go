package handler

import (
	"bluewave-uptime-agent/internal/metric"

	"github.com/gin-gonic/gin"
)

func Metrics(c *gin.Context) {
	metrics := metric.GetAllSystemMetrics()
	c.JSON(200, metrics)
	return
}

func MetricsCPU(c *gin.Context) {
	cpuMetrics, metricsErrs := metric.CollectCpuMetrics()

	c.JSON(200, metric.ApiResponse{
		Data:   cpuMetrics,
		Errors: metricsErrs,
	})
	return
}

func MetricsMemory(c *gin.Context) {
	memoryMetrics, metricsErrs := metric.CollectMemoryMetrics()

	c.JSON(200, metric.ApiResponse{
		Data:   memoryMetrics,
		Errors: metricsErrs,
	})
	return
}

func MetricsDisk(c *gin.Context) {
	diskMetrics, metricsErrs := metric.CollectDiskMetrics()

	c.JSON(200, metric.ApiResponse{
		Data:   diskMetrics,
		Errors: metricsErrs,
	})
	return
}

func MetricsHost(c *gin.Context) {
	hostMetrics, metricsErrs := metric.GetHostInformation()

	c.JSON(200, metric.ApiResponse{
		Data:   hostMetrics,
		Errors: metricsErrs,
	})
	return
}
