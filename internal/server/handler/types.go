package handler

import "github.com/bluewave-labs/capture/internal/metric"

type APIResponse struct {
	Data    metric.Metric      `json:"data"`
	Capture CaptureMeta        `json:"capture"`
	Errors  []metric.CustomErr `json:"errors"`
}

type CaptureMeta struct {
	Version string `json:"version"`
	Mode    string `json:"mode"`
}
