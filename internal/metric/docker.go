package metric

import (
	"context"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ContainerMetrics struct {
	ContainerID   string
	ContainerName string
	Healthy       bool
	BaseImage     string
	ExposedPorts  []Port
	StartedAt     string
	FinishedAt    string
}

type Port struct {
	Port     string
	Protocol string
}

func GetDockerMetrics(all bool) []ContainerMetrics {
	var metrics = make([]ContainerMetrics, 0)
	ctx := context.Background()

	// Initialize the Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer cli.Close()

	// List all containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All: all,
	})
	if err != nil {
		log.Fatalf("Failed to list containers: %v", err)
	}

	for _, container := range containers {
		// Inspect each container
		containerInspectResponse, err := cli.ContainerInspect(ctx, container.ID)
		if err != nil {
			log.Printf("Failed to inspect container %s: %v", container.ID, err)
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

	return metrics
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
