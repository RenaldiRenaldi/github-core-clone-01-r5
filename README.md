# R5 Go Forge

An in-memory GitHub-shaped forge implemented in Go. The entrypoint starts the prebuilt native process; the service owns its HTTP listener and harness lifecycle. It uses only the Go standard library and keeps issue and repository data in a shared in-memory store.

## Build and test

```sh
CGO_ENABLED=0 ~/bin/go/bin/go build -trimpath -ldflags='-s -w' -o forge .
~/bin/go/bin/go test ./...
~/bin/go/bin/go test -race ./...
```

The repository includes a statically linked Linux amd64 executable for the trial, where source files are read-only and no compiler is present. The local default token is `forge-demo`. `MAKEFUN_SHOP_TOKEN` and `MAKEFUN_ENV_URL` provide the trial token and registration service. Data lasts only for the process lifetime.

## Implemented surfaces

The server exposes health, REST repository, issue, label and pull request routes, a JSON-RPC MCP endpoint, a small GraphQL response surface, and initial smart-HTTP advertisement handling. REST, MCP and GraphQL use the same repository store. The initial Git transport responds to advertisements and request endpoints but does not yet implement pack negotiation or object transfer.

## Measurements

The first committed suite run scored **0/1000** because the native binary was absent and the entry exited before registration. The entrypoint now builds it when a Go compiler is available locally. This development fallback is not suitable for the grading pod, which requires a committed static executable. R3 and R4 data are not measurements of this source and are not presented as R5 results.

## License

MIT. See [LICENSE](LICENSE).
