package metric

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ContainerMetrics struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Healthy       bool   `json:"healthy"`
	BaseImage     string `json:"base_image"`
	ExposedPorts  []Port `json:"exposed_ports"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at"`
}

func (c ContainerMetrics) isMetric() {}

type Port struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
}

func GetDockerMetrics(all bool) (MetricsSlice, []CustomErr) {
	var metrics = make(MetricsSlice, 0)
	var containerErrors []CustomErr

	ctx := context.Background()

	// Initialize the Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		containerErrors = append(containerErrors, CustomErr{
			Metric: []string{"docker.client"},
			Error:  err.Error(),
		})
		return nil, containerErrors
	}
	defer cli.Close()

	// List all containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All: all,
	})
	if err != nil {
		return nil, append(containerErrors, CustomErr{
			Metric: []string{"docker.container.list"},
			Error:  err.Error(),
		})
	}

	for _, container := range containers {
		// Inspect each container
		containerInspectResponse, err := cli.ContainerInspect(ctx, container.ID)
		if err != nil {
			continue
		}

		healthy := healthCheck(containerInspectResponse)
		portList := make([]Port, 0)
		for port := range containerInspectResponse.Config.ExposedPorts {
			portList = append(portList, Port{
				Port:     port.Port(),
				Protocol: port.Proto(),
			})
		}

		metrics = append(metrics, ContainerMetrics{
			ContainerID:   container.ID,
			ContainerName: container.Names[0],
			Healthy:       healthy,
			BaseImage:     container.Image,
			ExposedPorts:  portList,
			StartedAt:     containerInspectResponse.State.StartedAt,
			FinishedAt:    containerInspectResponse.State.FinishedAt,
		})
	}

	return metrics, nil
}

func healthCheck(inspectResponse container.InspectResponse) bool {
	var healthStatus bool

	if inspectResponse.State.OOMKilled || inspectResponse.State.Dead || inspectResponse.State.ExitCode != 0 || inspectResponse.State.Status != "running" {
		healthStatus = false
	}

	if inspectResponse.State != nil && inspectResponse.State.Running {
		healthStatus = true
	}

	return healthStatus
}
