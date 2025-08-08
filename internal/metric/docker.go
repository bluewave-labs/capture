package metric

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ContainerMetrics struct {
	ContainerID   string                 `json:"container_id"`
	ContainerName string                 `json:"container_name"`
	Status        string                 `json:"status"` // "created", "running", "paused", "restarting", "removing", "exited", "dead"
	Health        *ContainerHealthStatus `json:"health"`
	Running       bool                   `json:"running"`
	BaseImage     string                 `json:"base_image"`
	ExposedPorts  []Port                 `json:"exposed_ports"`
	StartedAt     int64                  `json:"started_at"`  // Unix timestamp
	FinishedAt    int64                  `json:"finished_at"` // Unix timestamp
}

func (c ContainerMetrics) isMetric() {}

type Port struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
}

type ContainerHealthStatus struct {
	Healthy bool                  `json:"healthy"`
	Source  ContainerHealthSource `json:"source"`  // "container_health_check", "state_based_health_check"
	Message string                `json:"message"` // Additional message if needed
}

type ContainerHealthSource string

const (
	SourceContainerHealthCheck  ContainerHealthSource = "container_health_check"
	SourceStateBasedHealthCheck ContainerHealthSource = "state_based_health_check"
)

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
			containerErrors = append(containerErrors, CustomErr{
				Metric: []string{"docker.container.inspect"},
				Error:  err.Error(),
			})
			continue
		}

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
			Status:        containerInspectResponse.State.Status, // Can be one of "created", "running", "paused", "restarting", "removing", "exited", or "dead"
			Running:       containerInspectResponse.State.Running,
			BaseImage:     container.Image,
			ExposedPorts:  portList,
			StartedAt:     GetUnixTimestamp(containerInspectResponse.State.StartedAt),
			FinishedAt:    GetUnixTimestamp(containerInspectResponse.State.FinishedAt),
			Health:        healthCheck(containerInspectResponse),
		})
	}

	return metrics, nil
}

func stateBasedHealthCheck(inspectResponse container.InspectResponse) bool {
	if inspectResponse.State == nil {
		// If the state is nil, we cannot determine health
		return false
	}
	// Check for explicit failure conditions first
	if inspectResponse.State.OOMKilled || inspectResponse.State.Dead || inspectResponse.State.ExitCode != 0 {
		return false
	}

	// Only consider healthy if running and status is "running"
	return inspectResponse.State != nil && inspectResponse.State.Running && inspectResponse.State.Status == "running"
}

// healthCheck returns the health status of a container based on its inspection response.
// If there is health check information available, it returns the health status.
// If not, it runs a state based health check based on the container's state.
// If the container is running and healthy, it returns a healthy status.
// If the container is not running or has failed, it returns an unhealthy status.
// If the container is starting, it returns 'healthy' status.
func healthCheck(inspectResponse container.InspectResponse) *ContainerHealthStatus {
	if inspectResponse.State.Health != nil {
		// If the container has a health check, return its status
		return &ContainerHealthStatus{
			// If the health check is healthy or starting, consider it healthy
			Healthy: inspectResponse.State.Health.Status == "healthy" || inspectResponse.State.Health.Status == "starting",
			Source:  SourceContainerHealthCheck,
			Message: "Based on container health check",
		}
	}

	// If no health check is defined, run state based health check
	return &ContainerHealthStatus{
		Healthy: stateBasedHealthCheck(inspectResponse),
		Source:  SourceStateBasedHealthCheck,
		Message: "Based on container state",
	}
}

// getContainerName extracts the container name from the list of names.
func getContainerName(names []string) string {
	if len(names) == 0 || len(names[0]) == 0 {
		// If there are no names or the first name is empty, return an empty string
		return ""
	}

	if names[0][0] == '/' {
		return names[0][1:] // Remove the leading '/' from the container name
	}

	return names[0]
}

// GetUnixTimestamp converts a timestamp string in RFC3339 format to a Unix timestamp.
// If the timestamp is invalid or represents the zero value, it returns 0.
// The function handles both seconds and nanoseconds precision.
func GetUnixTimestamp(timestamp string) int64 {
	// Convert the timestamp string to a Unix timestamp
	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil || timestamp == "0001-01-01T00:00:00Z" {
		return 0 // Return 0 if parsing fails or if the timestamp is the zero value
	}
	return t.Unix()
}
