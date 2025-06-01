package handler

import (
	"github.com/bluewave-labs/capture/internal/metric"
	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	metadata *CaptureMeta
}

func NewMetricsHandler(metadata *CaptureMeta) *MetricsHandler {
	return &MetricsHandler{
		metadata: metadata,
	}
}

func (h *MetricsHandler) handleResponse(c *gin.Context, metrics metric.Metric, errs []metric.CustomErr) {
	statusCode := 200
	if len(errs) > 0 {
		statusCode = 207
	}
	c.JSON(statusCode, APIResponse{
		Data:    metrics,
		Errors:  errs,
		Capture: *h.metadata,
	})
}

func (h *MetricsHandler) Metrics(c *gin.Context) {
	metrics, metricsErrs := metric.GetAllSystemMetrics()
	h.handleResponse(c, metrics, metricsErrs)
}

func (h *MetricsHandler) MetricsCPU(c *gin.Context) {
	cpuMetrics, metricsErrs := metric.CollectCPUMetrics()
	h.handleResponse(c, cpuMetrics, metricsErrs)
}

func (h *MetricsHandler) MetricsMemory(c *gin.Context) {
	memoryMetrics, metricsErrs := metric.CollectMemoryMetrics()
	h.handleResponse(c, memoryMetrics, metricsErrs)
}

func (h *MetricsHandler) MetricsDisk(c *gin.Context) {
	diskMetrics, metricsErrs := metric.CollectDiskMetrics()
	h.handleResponse(c, diskMetrics, metricsErrs)
}

func (h *MetricsHandler) MetricsHost(c *gin.Context) {
	hostMetrics, metricsErrs := metric.GetHostInformation()
	h.handleResponse(c, hostMetrics, metricsErrs)
}

func (h *MetricsHandler) SmartMetrics(c *gin.Context) {
	smartMetrics, smartErrs := metric.GetSmartMetrics()
	h.handleResponse(c, smartMetrics, smartErrs)
}

func MetricsNet(c *gin.Context) {
	netMetrics, netErrs := metric.GetNetInformation()
	handleMetricResponse(c, netMetrics, netErrs)
}
