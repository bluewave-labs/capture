# BlueWave Uptime Agent

## API Responses

| Endpoint          | Method | Description                                          |
|-------------------|--------|------------------------------------------------------|
| `/metrics`        | GET    | Returns the all system metrics(cpu,memory,disk,host) |
| `/metrics/cpu`    | GET    | Returns the system cpu metrics                       |
| `/metrics/memory` | GET    | Returns the system memory metrics                    |
| `/metrics/disk`   | GET    | Returns the system disk metrics                      |
| `/metrics/host`   | GET    | Returns the system host informations                 |

### CPU Response

```jsonc
{
    "physical_core": integer, // Physical cores
    "logical_core":  integer, // Logical cores aka Threads
    "frequency":     integer, // Frequency in mHz
    "temperature":   null,    // WIP
    "free_percent":  null,    // WIP
    "usage_percent": null     // WIP
}
```

### Memory Response

```jsonc
{
    "total_bytes":     integer, // Total space in bytes
    "available_bytes": integer, // Available space in bytes
    "used_bytes":      integer, // Used space in bytes      //* Total - Free - Buffers - Cached
    "usage_percent":   float    // Usage Percent            //* (Used / Total) * 100.0
}
```

### Disk Response

```jsonc
{
    "read_speed_bytes":  integer, // WIP
    "write_speed_bytes": integer, // WIP
    "total_bytes":       integer, // Total space of "/" in bytes
    "free_bytes":        integer, // WIP
    "usage_percent":     float    // WIP
}
```

### Host Response

```jsonc
{
    "os":             string, // linux, darwin, windows
    "platform":       string, // arch, debian, suse...
    "kernel_version": string, // 6.10.10, 6.0.0, 6.10.0-zen...
}
```

## Availability

| CPU                 | GNU/Linux | Windows | MacOS     |
| --------------------|-----------|---------|-----------|
| Physical Core Count | ✅        | -       | -         |
| Logical Core Count  | ✅        | -       | -         |
| Frequency           | ✅        | -       | -         |
| Temperature         | -         | -       | -         |
| Free Percent        | -         | -       | -         |
| Usage Percent       | -         | -       | -         |

| Memory          | GNU/Linux | Windows | MacOS     |
| ----------------|-----------|---------|-----------|
| Total Bytes     | ✅        | -       | -         |
| Available Bytes | ✅        | -       | -         |
| Used Bytes      | ✅        | -       | -         |
| Usage Percent   | ✅        | -       | -         |

| Disk          | GNU/Linux | Windows | MacOS     |
| --------------|-----------|---------|-----------|
| Total Bytes   | -         | -       | -         |
| Free Bytes    | -         | -       | -         |
| Usage Percent | -         | -       | -         |

| Host           | GNU/Linux | Windows | MacOS     |
| ---------------|-----------|---------|-----------|
| OS             | ✅        | -       | -         |
| Platform       | ✅        | -       | -         |
| Kernel Version | ✅        | -       | -         |
