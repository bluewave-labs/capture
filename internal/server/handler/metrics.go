package handler

import (
	"github.com/bluewave-labs/capture/internal/metric"
	"github.com/gin-gonic/gin"
)

var Metadata *metric.CaptureMeta

func handleMetricResponse(c *gin.Context, metrics metric.Metric, errs []metric.CustomErr) {
	statusCode := 200
	if len(errs) > 0 {
		statusCode = 207
	}
	c.JSON(statusCode, metric.APIResponse{
		Data:    metrics,
		Errors:  errs,
		Capture: *Metadata, // Include metadata in the response
	})
}

func Metrics(c *gin.Context) {
	metrics, metricsErrs := metric.GetAllSystemMetrics()
	handleMetricResponse(c, metrics, metricsErrs)
}

func MetricsCPU(c *gin.Context) {
	cpuMetrics, metricsErrs := metric.CollectCPUMetrics()
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

func SmartMetrics(c *gin.Context) {
	smartMetrics, smartErrs := metric.GetSmartMetrics()
	handleMetricResponse(c, smartMetrics, smartErrs)
}
