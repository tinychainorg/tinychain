# Development.

Tinychain development uses continuous integration. There are multiple workflows on submission of a PR:

 - **build**: builds the Go package.
 - **test**: runs the test suite. 
 - **format** - checks the Go code for formatting.
 - **vet** - a specialised Go tool which functions like a linter, catching potential errors in code.

## Running tests.

### Individual.

```sh
go test -v -run TestStartPeerHeartbeat ./...
```

### The entire test suite.

```sh
go test -v -run ./... > test.log
```

### Benchmarking.

Go will run your function multiple times and find the median for a benchmark:

```go
go test -v -bench=1 -run TestBenchmarkTxOpsPerDay
```

## Analysing binary size.

```sh
go install github.com/Zxilly/go-size-analyzer/cmd/gsa@latest
gsa --web  build/tinychain-darwin-arm64
```
