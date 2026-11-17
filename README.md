# R5 Go Forge

An in-memory GitHub-shaped forge implemented in Go. The entrypoint starts the prebuilt native process; the service owns its HTTP listener and harness lifecycle. It uses only the Go standard library and keeps issue and repository data in a shared in-memory store.

## Build and test

```sh
CGO_ENABLED=0 ~/bin/go/bin/go build -trimpath -ldflags='-s -w' -o forge .
~/bin/go/bin/go test ./...
~/bin/go/bin/go test -race ./...
```

The local default token is `forge-demo`. In a trial, `MAKEFUN_SHOP_TOKEN` supplies the token and `MAKEFUN_ENV_URL` supplies the registration service. The store is ephemeral.

## Implemented surfaces

The server exposes health, REST repository, issue, label and pull request routes, a JSON-RPC MCP endpoint, a small GraphQL response surface, and initial smart-HTTP advertisement handling. REST, MCP and GraphQL use the same repository store. The initial Git transport responds to advertisements and request endpoints but does not yet implement pack negotiation or object transfer.

## Measurements

No valid suite benchmark has been recorded for this revision yet. R3 and R4 same-host data from the previous implementation are not measurements of this source and are intentionally not presented as R5 results. Run the kit's `grade-local` after building and record its emitted performance export here.

## License

MIT. See [LICENSE](LICENSE).
