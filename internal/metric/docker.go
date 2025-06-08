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
	Status        string `json:"status"` // "created", "running", "paused", "restarting", "removing", "exited", "dead"
	Running       bool   `json:"running"`
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
			ContainerName: getContainerName(container.Names),
			Healthy:       healthy,
			Status:        containerInspectResponse.State.Status, // Can be one of "created", "running", "paused", "restarting", "removing", "exited", or "dead"
			Running:       containerInspectResponse.State.Running,
			BaseImage:     container.Image,
			ExposedPorts:  portList,
			StartedAt:     containerInspectResponse.State.StartedAt,
			FinishedAt:    containerInspectResponse.State.FinishedAt,
		})
	}

	return metrics, nil
}

func healthCheck(inspectResponse container.InspectResponse) bool {
	// Check for explicit failure conditions first
	if inspectResponse.State.OOMKilled || inspectResponse.State.Dead || inspectResponse.State.ExitCode != 0 {
		return false
	}

	// Only consider healthy if running and status is "running"
	return inspectResponse.State != nil && inspectResponse.State.Running && inspectResponse.State.Status == "running"
}

// getContainerName extracts the container name from the list of names.
func getContainerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	// Remove the leading '/' from the container name
	return names[0][1:]
}
