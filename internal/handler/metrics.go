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
	metrics, metricsErrs := metric.CollectCpuMetrics()
	apiResponse := metric.ApiResponse{
		Cpu:    metrics,
		Errors: metricsErrs,
	}
	c.JSON(200, apiResponse)
	return
}

func MetricsMemory(c *gin.Context) {
	metrics, metricsErrs := metric.CollectMemoryMetrics()
	apiResponse := metric.ApiResponse{
		Memory: metrics,
		Errors: metricsErrs,
	}
	c.JSON(200, apiResponse)
	return
}

func MetricsDisk(c *gin.Context) {
	metrics, metricsErrs := metric.CollectDiskMetrics()
	apiResponse := metric.ApiResponse{
		Disk:   metrics,
		Errors: metricsErrs,
	}
	c.JSON(200, apiResponse)
	return
	return
}

func MetricsHost(c *gin.Context) {
	metrics, metricsErrs := metric.GetHostInformation()
	apiResponse := metric.ApiResponse{
		Host:   metrics,
		Errors: metricsErrs,
	}
	c.JSON(200, apiResponse)
	return
	return
}
