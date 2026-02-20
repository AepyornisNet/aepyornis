# Backoff

Exponential backoff implementation extracted from [`gRPC-Go`](https://github.com/grpc/grpc-go)

`gRPC-Go` has really nice exponential backoff implementation, but it is impossible to reuse as it resides
in `internal` package, so it can be imported only within `gRPC-Go`.

## Usage

```go
package main

import "github.com/vgarvardt/backoff"

func main() {
    retry := 5
    waitFor := backoff.DefaultExponential.Backoff(retry)
    ...
}
```
