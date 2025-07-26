![Tests](https://github.com/prevostcorentin/go-qga/actions/workflows/test.yml/badge.svg)
![Lint](https://github.com/prevostcorentin/go-qga/actions/workflows/lint.yml/badge.svg)
[![codecov](https://codecov.io/gh/prevostcorentin/go-qga/branch/main/graph/badge.svg?token=ZGKL57SARB)](https://codecov.io/gh/prevostcorentin/go-qga)
[![Go Report Card](https://goreportcard.com/badge/github.com/prevostcorentin/go-qga)](https://goreportcard.com/report/github.com/prevostcorentin/go-qga)
[![License](https://img.shields.io/github/license/prevostcorentin/go-qga)](./LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/prevostcorentin/go-qga.svg)](https://pkg.go.dev/github.com/prevostcorentin/go-qga)
[![Last Commit](https://img.shields.io/github/last-commit/prevostcorentin/go-qga)](https://github.com/prevostcorentin/go-qga)


# go-qga

**go-qga** is a high-performance Go library for interacting with the **QEMU Guest Agent** using the **QEMU Machine Protocol (QMP)**.

It provides both simple and advanced APIs with optimistic error handling, connection pooling, async execution, and comprehensive testing infrastructure for reliable VM automation and management.

---

## 🚀 Features

- 🚀 **High Performance**: Optimistic error handling, connection pooling, and async execution
- 📡 **Multiple APIs**: Simple client for basic use, executor for advanced scenarios
- 🔒 **Type Safety**: Strongly-typed command interface with structured error handling
- 🧪 **Testing Infrastructure**: Built-in test agents (Simple, Fast, Perfect) for development
- 🔁 **Connection Management**: Unix socket transport with automatic reconnection
- ⚡ **Worker Pools**: Concurrent command execution with configurable workers
- 🛠️ **Extensible**: Clean interfaces for custom commands and transports

---

## 📦 Installation

```bash
go get github.com/prevostcorentin/go-qga
```

## 🧰 Usage

### Simple Client API

```go
package main

import (
    "fmt"
    "log"
    "github.com/prevostcorentin/go-qga/internal/qmp"
)

func main() {
    // Simple one-liner connection
    client := qmp.NewClient("/path/to/qga.sock")
    if client.IsErr() {
        log.Fatal(client.Error())
    }
    defer client.Value().Close()

    // Get hostname with optimistic error handling
    hostname := client.Value().GetHostname()
    if hostname.IsErr() {
        log.Fatal(hostname.Error())
    }
    
    fmt.Println("VM hostname:", hostname.Value())
}
```

### Advanced Executor API

```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/prevostcorentin/go-qga/internal/qmp"
    "github.com/prevostcorentin/go-qga/internal/qmp/transport"
)

type HostnameCommand struct{}

func (c *HostnameCommand) Execute() string { return "guest-get-host-name" }
func (c *HostnameCommand) Arguments() any { return nil }
func (c *HostnameCommand) Response() any { return &HostnameResponse{} }

type HostnameResponse struct {
    Name string `json:"name"`
}

func main() {
    // Advanced usage with executor
    transport := transport.NewUnix("/path/to/qga.sock")
    connection, err := qmp.NewConnection(transport)
    if err != nil {
        log.Fatal(err)
    }
    defer connection.Close()

    executor, err := qmp.NewExecutor(connection)
    if err != nil {
        log.Fatal(err)
    }
    defer executor.Close()

    // Synchronous execution
    result, err := executor.Run(context.Background(), &HostnameCommand{})
    if err != nil {
        log.Fatal(err)
    }
    
    response := result.(*HostnameResponse)
    fmt.Println("VM hostname:", response.Name)

    // Asynchronous execution
    resultCh := executor.RunAsync(context.Background(), &HostnameCommand{})
    select {
    case execResult := <-resultCh:
        if execResult.Err != nil {
            log.Fatal(execResult.Err)
        }
        response := execResult.Data.(*HostnameResponse)
        fmt.Println("Async hostname:", response.Name)
    }
}
```

## 🧪 Testing

The library includes comprehensive testing infrastructure with fake QMP agents:

```go
package main

import (
    "github.com/prevostcorentin/go-qga/internal/testing"
    "github.com/prevostcorentin/go-qga/internal/qmp"
)

func main() {
    // Start a test agent
    agent := testing.Perfect("/tmp/test-qga.sock")
    result := agent.Start()
    if result.IsErr() {
        panic(result.Error())
    }
    defer agent.Stop()

    // Connect and test
    client := qmp.NewClient(agent.Path())
    hostname := client.Value().GetHostname()
    // hostname.Value() == "perfect-vm"
}
```

**Available Test Agents:**
- `testing.Simple()` - Basic functionality
- `testing.Fast()` - High-performance testing  
- `testing.Perfect()` - Full-featured with benchmarking

All agents support fluent configuration:
```go
agent := testing.Perfect("/tmp/qga.sock").
    WithConnections(10000).
    WithTimeout(30*time.Second).
    WithBuffer(8192)
```

## 🔮 Roadmap

- [x] QMP socket communication
- [x] Generic command execution  
- [x] Async execution with worker pools
- [x] Comprehensive error handling system
- [x] Connection pooling and management
- [x] Testing infrastructure with fake agents
- [ ] JSON struct code generation from QMP schema
- [ ] Command retry/replay support
- [ ] HTTP/TCP transport support
- [ ] Metrics and observability

## 🤝 Contributing

This is currently a solo project but contributions, ideas and feedback are always welcome. Open an issue to start a conversation or suggest a feature.

## License

This project is licensed under the Apache License, Version 2.0.  
See the [LICENSE](./LICENSE) file for details.

⚠️ Previous versions were published under the MIT License. The license was changed starting from version v0.2.0.
