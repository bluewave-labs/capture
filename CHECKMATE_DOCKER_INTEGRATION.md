# Capture Docker Monitoring API - Integration Guide for Checkmate

This document contains all necessary information to implement Docker container monitoring in Checkmate using Capture's API.

## Table of Contents
- [Overview](#overview)
- [Authentication](#authentication)
- [Docker Endpoint](#docker-endpoint)
- [Data Structures](#data-structures)
- [Example Responses](#example-responses)
- [Integration Architecture](#integration-architecture)
- [Error Handling](#error-handling)
- [Display Recommendations](#display-recommendations)

---

## Overview

**Capture** is a hardware monitoring agent that exposes metrics via REST API. Checkmate polls these endpoints periodically to display real-time monitoring data.

**Base URL Format:** `http://<server-ip>:<port>`
**Default Port:** `59232`

---

## Authentication

All `/api/v1/**` endpoints require Bearer token authentication.

### Request Header
```
Authorization: Bearer <API_SECRET>
```

The `API_SECRET` is configured in Capture and must match what's stored in Checkmate's server configuration.

### Public Endpoints
- `GET /health` - No authentication required (returns `"OK"`)

---

## Docker Endpoint

### Endpoint Details

**URL:** `GET /api/v1/metrics/docker`

**Query Parameters:**
- `all` (optional) - Include stopped containers
  - `?all=true` - Returns all containers (running + stopped)
  - No parameter or `?all=false` - Returns only running containers (default)

**Authentication:** Required (Bearer token)

**Response Status Codes:**
- `200 OK` - All metrics collected successfully
- `207 Multi-Status` - Partial success (some containers failed, see `errors` array)
- `401 Unauthorized` - Missing or invalid API secret
- `500 Internal Server Error` - Docker daemon unreachable or system error

---

## Data Structures

### Response Envelope

All Capture endpoints use this wrapper:

```json
{
  "data": <metric-specific-data>,
  "capture": {
    "version": "string",
    "mode": "string"
  },
  "errors": [<optional-error-objects>]
}
```

### Docker Response Data Structure

The `data` field contains an **array** of container objects:

```typescript
interface DockerResponse {
  data: ContainerMetrics[];
  capture: CaptureMetadata;
  errors: MetricError[] | null;
}

interface ContainerMetrics {
  container_id: string;       // Full container ID (e.g., "abc123def456...")
  container_name: string;     // Container name without leading slash
  status: ContainerStatus;    // Docker state
  health: ContainerHealthStatus;
  running: boolean;           // Simple running flag
  base_image: string;         // Image name (e.g., "nginx:latest")
  exposed_ports: Port[];      // Array of exposed ports
  started_at: number;         // Unix timestamp (seconds), 0 if never started
  finished_at: number;        // Unix timestamp (seconds), 0 if still running
}

type ContainerStatus =
  | "created"
  | "running"
  | "paused"
  | "restarting"
  | "removing"
  | "exited"
  | "dead";

interface ContainerHealthStatus {
  healthy: boolean;           // true = healthy/starting, false = unhealthy
  source: HealthCheckSource;
  message: string;            // Human-readable explanation
}

type HealthCheckSource =
  | "container_health_check"        // From Docker HEALTHCHECK instruction
  | "state_based_health_check";     // Fallback when no healthcheck defined

interface Port {
  port: string;               // Port number as string (e.g., "80")
  protocol: string;           // "tcp" or "udp"
}

interface CaptureMetadata {
  version: string;            // Capture version (e.g., "1.2.0")
  mode: string;               // "release" or "debug"
}

interface MetricError {
  metric: string[];           // Error source path (e.g., ["docker.client"])
  err: string;                // Error message
}
```

---

## Example Responses

### Success - Multiple Containers Running

```json
{
  "data": [
    {
      "container_id": "abc123def456789012345678901234567890abcdef123456789012345678901234",
      "container_name": "my-web-app",
      "status": "running",
      "health": {
        "healthy": true,
        "source": "container_health_check",
        "message": "Based on container health check"
      },
      "running": true,
      "base_image": "nginx:1.25-alpine",
      "exposed_ports": [
        {
          "port": "80",
          "protocol": "tcp"
        },
        {
          "port": "443",
          "protocol": "tcp"
        }
      ],
      "started_at": 1735689600,
      "finished_at": 0
    },
    {
      "container_id": "def456abc789012345678901234567890abcdef123456789012345678901234",
      "container_name": "redis-cache",
      "status": "running",
      "health": {
        "healthy": true,
        "source": "state_based_health_check",
        "message": "Based on container state"
      },
      "running": true,
      "base_image": "redis:7.2",
      "exposed_ports": [
        {
          "port": "6379",
          "protocol": "tcp"
        }
      ],
      "started_at": 1735689700,
      "finished_at": 0
    }
  ],
  "capture": {
    "version": "1.2.0",
    "mode": "release"
  },
  "errors": null
}
```

### With Stopped Containers (`?all=true`)

```json
{
  "data": [
    {
      "container_id": "xyz789abc012345678901234567890abcdef123456789012345678901234",
      "container_name": "old-database",
      "status": "exited",
      "health": {
        "healthy": false,
        "source": "state_based_health_check",
        "message": "Based on container state"
      },
      "running": false,
      "base_image": "postgres:14",
      "exposed_ports": [
        {
          "port": "5432",
          "protocol": "tcp"
        }
      ],
      "started_at": 1735600000,
      "finished_at": 1735686000
    }
  ],
  "capture": {
    "version": "1.2.0",
    "mode": "release"
  },
  "errors": null
}
```

### Partial Failure (207 Multi-Status)

```json
{
  "data": [
    {
      "container_id": "abc123...",
      "container_name": "web-server",
      "status": "running",
      "health": {
        "healthy": true,
        "source": "container_health_check",
        "message": "Based on container health check"
      },
      "running": true,
      "base_image": "nginx:latest",
      "exposed_ports": [],
      "started_at": 1735689600,
      "finished_at": 0
    }
  ],
  "capture": {
    "version": "1.2.0",
    "mode": "release"
  },
  "errors": [
    {
      "metric": ["docker.container.inspect"],
      "err": "Error response from daemon: No such container: def456"
    }
  ]
}
```

### Docker Daemon Unreachable (500 Error)

```json
{
  "data": null,
  "capture": {
    "version": "1.2.0",
    "mode": "release"
  },
  "errors": [
    {
      "metric": ["docker.client"],
      "err": "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?"
    }
  ]
}
```

### Empty Response (No Containers)

```json
{
  "data": [],
  "capture": {
    "version": "1.2.0",
    "mode": "release"
  },
  "errors": null
}
```

---

## Integration Architecture

### Polling Pattern

Checkmate should poll Capture's Docker endpoint periodically:

```
Checkmate Backend → (every N seconds) → GET /api/v1/metrics/docker → Capture API
                   ← (JSON response)
```

**Recommended Polling Interval:** 10-30 seconds

### Configuration in Checkmate

Users configure Capture servers in Checkmate with:
1. **Server URL:** `http://192.168.1.100:59232` (or domain name)
2. **API Secret:** The matching secret configured in Capture
3. **Optional: Enable Docker Monitoring** (checkbox to poll `/api/v1/metrics/docker`)

### Data Flow

1. **User adds Capture server to Checkmate**
   - Enters IP/hostname, port, and API secret
   - Checkmate validates connection via `/health` endpoint

2. **Checkmate periodically polls endpoints**
   - Main metrics: `GET /api/v1/metrics` (CPU, memory, disk, network)
   - Docker metrics: `GET /api/v1/metrics/docker` (containers)

3. **Checkmate stores and displays data**
   - Store time-series data for historical graphs
   - Display current state in dashboard
   - Alert on container failures

---

## Error Handling

### Common Error Scenarios

| HTTP Status | Scenario | Action |
|-------------|----------|--------|
| `200` | Success | Process data normally |
| `207` | Partial failure | Display available containers, log errors |
| `401` | Invalid API secret | Show authentication error to user |
| `404` | Endpoint not found | Check Capture version (Docker added in v1.2.0) |
| `500` | Docker daemon down | Show "Docker unavailable" status |
| Network timeout | Capture server offline | Mark server as offline in UI |

### Health Check Source Logic

- **`container_health_check`**: Container has `HEALTHCHECK` instruction in Dockerfile
  - `healthy: true` → Status is "healthy" or "starting"
  - `healthy: false` → Status is "unhealthy"

- **`state_based_health_check`**: Fallback when no HEALTHCHECK defined
  - `healthy: true` → Container is running AND status is "running"
  - `healthy: false` → Container has any of: OOMKilled, Dead, non-zero ExitCode, or not running

### Error Array Interpretation

The `errors` array contains objects with:
- `metric`: Array path indicating what failed (e.g., `["docker.client"]`, `["docker.container.inspect"]`)
- `err`: Error message string

**Common error metrics:**
- `["docker.client"]` → Cannot connect to Docker daemon
- `["docker.container.list"]` → Failed to list containers
- `["docker.container.inspect"]` → Failed to inspect specific container (partial failure)

---

## Display Recommendations

### Dashboard Layout Ideas

#### Container List View
```
┌─────────────────────────────────────────────────────────────┐
│ Docker Containers                                      [All]│
├─────────────────────────────────────────────────────────────┤
│ ● my-web-app          nginx:1.25-alpine      Running    2d  │
│   Ports: 80/tcp, 443/tcp                                    │
│                                                             │
│ ● redis-cache         redis:7.2              Running    2d  │
│   Ports: 6379/tcp                                           │
│                                                             │
│ ○ old-database        postgres:14            Exited     1d  │
│   Ports: 5432/tcp                                           │
└─────────────────────────────────────────────────────────────┘
```

#### Status Indicators
- 🟢 Green dot: `status: "running"` and `healthy: true`
- 🟡 Yellow dot: `status: "running"` and `healthy: false`
- 🔴 Red dot: `status: "exited"`, `"dead"`, or `"removing"`
- ⚪ Gray dot: `status: "created"` or `"paused"`
- 🔄 Spinner: `status: "restarting"`

#### Key Metrics to Display

**Per Container:**
- Container name (large, bold)
- Status badge (colored)
- Health status (icon + tooltip showing `message`)
- Base image (smaller text)
- Exposed ports (comma-separated)
- Uptime calculation: `current_time - started_at` (if running)
- Stopped duration: `finished_at - started_at` (if stopped)

**Summary Statistics:**
- Total containers
- Running containers count
- Stopped containers count
- Unhealthy containers count (alert badge)

#### Time Calculations

```javascript
// Uptime (for running containers)
if (container.running && container.started_at > 0) {
  const uptimeSeconds = Math.floor(Date.now() / 1000) - container.started_at;
  const uptime = formatDuration(uptimeSeconds); // "2d 5h 30m"
}

// Stopped duration (for exited containers)
if (!container.running && container.finished_at > 0) {
  const stoppedAgo = Math.floor(Date.now() / 1000) - container.finished_at;
  const stoppedTime = formatDuration(stoppedAgo); // "1d 3h ago"
}
```

#### Filtering Options
- Show only running containers (default)
- Show all containers (toggle to include stopped)
- Filter by status (running, exited, unhealthy)
- Search by name or image

#### Sorting Options
- By name (alphabetical)
- By status (running first)
- By health (unhealthy first)
- By uptime (longest first)

---

## Additional Notes

### Important Limitations

1. **Docker metrics are NOT in `/api/v1/metrics`**
   - Must call `/api/v1/metrics/docker` separately
   - The aggregated metrics endpoint does NOT include Docker data

2. **Capture must run on the Docker host**
   - If Capture runs inside a Docker container, it won't see host Docker containers unless `/var/run/docker.sock` is mounted
   - Recommended: Run Capture directly on the host machine for full visibility

3. **No resource usage metrics**
   - Capture currently does NOT provide CPU/memory usage per container
   - Only status, health, and metadata are available
   - For resource metrics, consider adding Docker stats API in future

### Feature Version

Docker monitoring was added in **Capture v1.2.0** (June 2025).

Servers running older versions will return `404 Not Found` for `/api/v1/metrics/docker`.

### Checkmate Should:

1. **Detect Capture version** from `capture.version` in responses
2. **Hide Docker features** if version < 1.2.0
3. **Handle 404 gracefully** and suggest upgrading Capture
4. **Store API secret securely** (encrypted in database)
5. **Implement retry logic** for network failures
6. **Cache responses** (optional) to reduce API load
7. **Support multiple Capture servers** (multi-server dashboard)

---

## Example Implementation (Pseudo-code)

```javascript
// Checkmate Backend - Docker Poller

class CaptureClient {
  constructor(baseURL, apiSecret) {
    this.baseURL = baseURL;
    this.apiSecret = apiSecret;
  }

  async fetchDockerMetrics(includeAll = false) {
    const url = `${this.baseURL}/api/v1/metrics/docker${includeAll ? '?all=true' : ''}`;

    try {
      const response = await fetch(url, {
        headers: {
          'Authorization': `Bearer ${this.apiSecret}`
        },
        timeout: 10000 // 10 second timeout
      });

      if (response.status === 401) {
        throw new Error('Invalid API secret');
      }

      if (response.status === 404) {
        throw new Error('Docker monitoring not available (Capture version too old)');
      }

      const data = await response.json();

      if (response.status === 207) {
        // Partial failure - log errors but use available data
        console.warn('Docker metrics partial failure:', data.errors);
      }

      return {
        containers: data.data || [],
        errors: data.errors || [],
        version: data.capture.version
      };

    } catch (error) {
      if (error.name === 'TimeoutError') {
        throw new Error('Capture server unreachable');
      }
      throw error;
    }
  }
}

// Usage
const capture = new CaptureClient('http://192.168.1.100:59232', 'secret-key');

setInterval(async () => {
  try {
    const { containers, errors } = await capture.fetchDockerMetrics(false);

    // Update database
    await db.updateDockerContainers(serverId, containers);

    // Trigger alerts for unhealthy containers
    const unhealthy = containers.filter(c => !c.health.healthy && c.running);
    if (unhealthy.length > 0) {
      await alertService.sendAlert('Unhealthy containers detected', unhealthy);
    }

  } catch (error) {
    console.error('Failed to fetch Docker metrics:', error);
    await db.markServerOffline(serverId);
  }
}, 15000); // Poll every 15 seconds
```

---

## Support

For questions about Capture API:
- GitHub: https://github.com/bluewave-labs/capture
- Documentation: Check README.md and openapi.yml

For Checkmate integration questions:
- Contact Checkmate development team
