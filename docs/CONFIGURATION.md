# Capture Configuration Guide

The Capture application now supports comprehensive configuration management using Viper. This guide covers all configuration options and usage patterns.

## Quick Start

### 1. Generate a default configuration file
```bash
./capture --generate-config capture.yaml
```

### 2. Edit the configuration file
Set your `api_secret` in the generated file:
```yaml
server:
  api_secret: "your-secure-secret-here"
```

### 3. Validate your configuration
```bash
./capture --validate-config capture.yaml
```

### 4. Run with configuration file
```bash
./capture --config capture.yaml
```

## Configuration File Locations

Capture automatically searches for configuration files in the following locations (in order):
1. Current directory: `./capture.yaml`
2. Config directory: `./config/capture.yaml`
3. User home: `$HOME/.capture/capture.yaml`
4. System: `/etc/capture/capture.yaml`

You can also specify a custom path:
```bash
./capture --config /path/to/your/config.yaml
```

## Configuration Structure

### Basic Configuration
```yaml
version: 1

server:
  port: "59232"
  api_secret: "your-secure-secret-here"

log_level: "info"
```

### Full Configuration Example
```yaml
version: 1

# Server configuration
server:
  port: "59232"
  api_secret: "your-secure-secret-here"

# Checkmate targets for data forwarding
targets:
  - name: "Primary Checkmate"
    endpoint: "https://checkmate.example.com/api/v1/metrics"
    api_secret: "checkmate-secret"
    timeout: "30s"
    retry_delay: "5s"
    retry_count: 3

  - name: "Backup Checkmate"
    endpoint: "https://backup.example.com/api/v1/metrics"
    api_secret: "backup-secret"

# Global settings (applied to targets without specific settings)
global_interval: "1m"
global_timeout: "30s"
global_retry_delay: "5s"
global_retry_count: 3

# Logging
log_level: "info"  # error, warn, info, debug

# External plugins
plugins:
  - name: "disk-health"
    command: "/usr/local/bin/check-disk.sh"
  
  - name: "service-monitor"
    command: "/opt/scripts/check-services.py"
```

## Configuration Options

### Server Section
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `port` | string | "59232" | HTTP server port |
| `api_secret` | string | *required* | API authentication secret |

### Targets Section
Each target supports:
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `name` | string | *required* | Target identifier |
| `endpoint` | string | *required* | Checkmate endpoint URL |
| `api_secret` | string | *required* | Target API secret |
| `timeout` | duration | global_timeout | Request timeout |
| `retry_delay` | duration | global_retry_delay | Delay between retries |
| `retry_count` | int | global_retry_count | Number of retries |

### Global Settings
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `global_interval` | duration | "1m" | Data collection interval |
| `global_timeout` | duration | "30s" | Default request timeout |
| `global_retry_delay` | duration | "5s" | Default retry delay |
| `global_retry_count` | int | 3 | Default retry count |

### Plugins Section
Each plugin supports:
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `name` | string | *required* | Plugin identifier |
| `command` | string | *required* | Command to execute |

### Other Options
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `version` | int | 1 | Configuration version |
| `log_level` | string | "info" | Logging level (error/warn/info/debug) |

## Environment Variables

All configuration options can be overridden with environment variables using the `CAPTURE_` prefix:

```bash
# Server configuration
export CAPTURE_SERVER_PORT=8080
export CAPTURE_SERVER_API_SECRET=my-secret

# Global settings
export CAPTURE_GLOBAL_TIMEOUT=45s
export CAPTURE_LOG_LEVEL=debug

# Legacy support (will override the above)
export PORT=8080
export API_SECRET=my-secret
```

## Command Line Options

| Flag | Description |
|------|-------------|
| `--config path` | Specify configuration file path |
| `--generate-config path` | Generate default config at path |
| `--validate-config path` | Validate configuration file |
| `--show-config` | Display current configuration |
| `--version` | Show version information |

## Duration Format

Duration values support Go's duration format:
- `30s` - 30 seconds
- `5m` - 5 minutes
- `1h` - 1 hour
- `1h30m` - 1 hour 30 minutes

## Examples

### Environment-only Configuration
```bash
export API_SECRET=my-secret
export PORT=8080
export CAPTURE_LOG_LEVEL=debug
./capture
```

### Mixed Configuration
```bash
# Use config file with environment overrides
export CAPTURE_SERVER_PORT=9090
./capture --config capture.yaml
```

### Validation Workflow
```bash
# Generate config
./capture --generate-config /etc/capture/capture.yaml

# Edit the file (set secrets, add targets)
vim /etc/capture/capture.yaml

# Validate
./capture --validate-config /etc/capture/capture.yaml

# Test configuration
./capture --config /etc/capture/capture.yaml --show-config

# Run
./capture --config /etc/capture/capture.yaml
```

## Security Considerations

1. **API Secrets**: Never commit configuration files with real secrets to version control
2. **File Permissions**: Set appropriate permissions on configuration files containing secrets:
   ```bash
   chmod 600 capture.yaml
   ```
3. **Environment Variables**: Prefer environment variables for secrets in production
4. **Templates**: Use configuration templates with placeholder values for version control

## Migration from Environment Variables

If you're currently using only environment variables, you can:

1. Generate a config file: `./capture --generate-config capture.yaml`
2. Edit the file with your current values
3. Test: `./capture --config capture.yaml --show-config`
4. Deploy: `./capture --config capture.yaml`

Your environment variables will still work and override config file values when needed.
