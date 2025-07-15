package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/bluewave-labs/capture/internal/metric"
	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	metadata      *CaptureMeta
	InfluxStorage *metric.InfluxDBStorage
}

func NewMetricsHandler(metadata *CaptureMeta, influxStorage *metric.InfluxDBStorage) *MetricsHandler {
	if metadata == nil {
		metadata = &CaptureMeta{Version: "unknown", Mode: "unknown"}
	}
	return &MetricsHandler{
		metadata:      metadata,
		InfluxStorage: influxStorage,
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
	for _, m := range diskMetrics {
		d, ok := m.(*metric.DiskData)
		if !ok {
			continue
		}
		tags := map[string]string{"device": d.Device}
		fields := map[string]interface{}{
			"total_bytes":   derefUint64(d.TotalBytes),
			"used_bytes":    derefUint64(d.UsedBytes),
			"free_bytes":    derefUint64(d.FreeBytes),
			"usage_percent": derefFloat64(d.UsagePercent),
			// add more fields as needed
		}
		log.Printf("Writing disk metric to InfluxDB: device=%s, total_bytes=%d, used_bytes=%d, free_bytes=%d, usage_percent=%.4f",
			d.Device, fields["total_bytes"], fields["used_bytes"], fields["free_bytes"], fields["usage_percent"])
		if h.InfluxStorage != nil {
			err := h.InfluxStorage.WriteMetric("disk", tags, fields, time.Now())
			if err != nil {
				log.Println("Failed to write disk metric:", err)
			}
		}
	}
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

func (h *MetricsHandler) MetricsNet(c *gin.Context) {
	netMetrics, netErrs := metric.GetNetInformation()
	h.handleResponse(c, netMetrics, netErrs)
}

func (h *MetricsHandler) MetricsDocker(c *gin.Context) {
	all := c.Query("all") == "true"

	// Get Docker metrics, passing the "all" flag from the context
	// This will include all containers if "all" is true, otherwise only running containers
	dockerMetrics, dockerErrs := metric.GetDockerMetrics(all)
	h.handleResponse(c, dockerMetrics, dockerErrs)
}

func (h *MetricsHandler) DiskHistory(c *gin.Context) {
	log.Println("DiskHistory endpoint hit")
	device := c.Query("device")

	start := c.DefaultQuery("start", "-1h")
	stop := c.DefaultQuery("stop", "now()")

	records, err := h.InfluxStorage.QueryDiskHistory(device, start, stop)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": records})
}

// Helper functions (add these if not already present)
func derefUint64(p *uint64) uint64 {
	if p != nil {
		return *p
	}
	return 0
}
func derefFloat64(p *float64) float64 {
	if p != nil {
		return *p
	}
	return 0
}
