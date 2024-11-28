package handler

import (
	"github.com/bluewave-labs/capture/internal/metric"
	"github.com/gin-gonic/gin"
)

func handleMetricResponse(c *gin.Context, metrics metric.Metric, errs []metric.CustomErr) {
	statusCode := 200
	if len(errs) > 0 {
		statusCode = 207
	}
	c.JSON(statusCode, metric.ApiResponse{
		Data:   metrics,
		Errors: errs,
	})
	return
}

func Metrics(c *gin.Context) {
	metrics, metricsErrs := metric.GetAllSystemMetrics()
	handleMetricResponse(c, metrics, metricsErrs)
}

func MetricsCPU(c *gin.Context) {
	cpuMetrics, metricsErrs := metric.CollectCpuMetrics()
	handleMetricResponse(c, cpuMetrics, metricsErrs)
}

func MetricsMemory(c *gin.Context) {
	memoryMetrics, metricsErrs := metric.CollectMemoryMetrics()
	handleMetricResponse(c, memoryMetrics, metricsErrs)
}

func MetricsDisk(c *gin.Context) {
	diskMetrics, metricsErrs := metric.CollectDiskMetrics()
	handleMetricResponse(c, diskMetrics, metricsErrs)
}

func MetricsHost(c *gin.Context) {
	hostMetrics, metricsErrs := metric.GetHostInformation()
	handleMetricResponse(c, hostMetrics, metricsErrs)
}
