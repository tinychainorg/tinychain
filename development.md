# Development.

## Running tests.

### Individual.

```sh
go test -v -run TestStartPeerHeartbeat ./...
```

### The entire test suite.

```sh
go test -v -run ./... > test.log
```

## Analysing binary size.

```sh
go install github.com/Zxilly/go-size-analyzer/cmd/gsa@latest
gsa --web  build/tinychain-darwin-arm64
```
