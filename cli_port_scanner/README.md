# Go Port Scanner

A simple, concurrent TCP port scanner written in Go.
Scans a range of ports on a specified host and reports which ports are open.

## Features

- Fast, concurrent scanning using `goroutines` and worker pools
- Configurable host and port range via command-line flags
- Graceful output of open ports

## Usage

```sh
go run main.go --host <hostname> --start <start_port> --end <end_port>
```

### Flags

- `--host` (default: `localhost`): Host to scan
- `--start` (default: `1`): Start of port range
- `--end` (default: `1024`): End of port range

### Example

```sh
go run main.go --host example.com --start 20 --end 80
```

## Output

```text
Scanning example.com ports 20-80...
Open ports on example.com:
  - 22 open
  - 80 open
```

If no open ports are found:

```text
Open ports on example.com:
None found.
```
