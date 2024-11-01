package handler

import (
	"bluewave-uptime-agent/internal/metric"

	"github.com/gin-gonic/gin"
)

type HttpResponse struct {
	Data   interface{} `json:"metrics"`
	Errors []string
}

func Metrics(c *gin.Context) {
	metrics, metricsErrs := metric.GetAllSystemMetrics()
	response := HttpResponse{
		Data:   metrics,
		Errors: metricsErrs,
	}
	c.JSON(200, response)
	return
}

func MetricsCPU(c *gin.Context) {
	metrics, metricsErrs := metric.CollectCpuMetrics()
	response := HttpResponse{
		Data:   metrics,
		Errors: metricsErrs,
	}
	c.JSON(200, response)
	return
}

func MetricsMemory(c *gin.Context) {
	metrics, metricsErrs := metric.CollectMemoryMetrics()
	response := HttpResponse{
		Data:   metrics,
		Errors: metricsErrs,
	}
	c.JSON(200, response)
	return
}

func MetricsDisk(c *gin.Context) {
	metrics, metricsErrs := metric.CollectDiskMetrics()
	response := HttpResponse{
		Data:   metrics,
		Errors: metricsErrs,
	}
	c.JSON(200, response)
	return
}

func MetricsHost(c *gin.Context) {
	metrics, metricsErrs := metric.GetHostInformation()
	response := HttpResponse{
		Data:   metrics,
		Errors: metricsErrs,
	}
	c.JSON(200, response)
	return
}
