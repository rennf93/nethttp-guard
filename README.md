# nethttp-guard

net/http middleware adapter for [guard-core-go](https://github.com/rennf93/guard-core-go). Translates `*http.Request` into the guardcore request surface, runs the engine, and translates verdicts to exact HTTP responses. Works with the stdlib mux, chi, httprouter, gorilla, and anything speaking `func(http.Handler) http.Handler`.

Docs: <https://rennf93.github.io/nethttp-guard/>

## Install

```
go get github.com/rennf93/nethttp-guard@v1.0.0 github.com/rennf93/guard-core-go/v4@v4.0.4
```

## Usage

```go
package main

import (
	"log"
	"net/http"

	guardcore "github.com/rennf93/guard-core-go/v4/guardcore"
	nethttp "github.com/rennf93/nethttp-guard"
)

func main() {
	cfg := guardcore.DefaultSecurityConfig()
	engine, err := guardcore.NewEngine(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := engine.Initialize(); err != nil {
		log.Fatal(err)
	}

	guard, err := nethttp.New(engine)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	log.Fatal(http.ListenAndServe(":8080", guard(mux)))
}
```

Options: `nethttp.WithMaxBodyBytes(n)` bounds the body bytes the engine scans (default 262144), `nethttp.WithLogger(l)` swaps the fail-closed logger. Route-level configuration uses `engine.Routes.Register` plus `nethttp.WithRouteID(ctx, id)` on the request context.

Engine malfunctions fail closed with a 500. Detection covers at most the first `MaxBodyBytes` of the body; payloads beyond the bound are not scanned, and the full body still reaches your handler untouched.

## Development

The middleware consumes the core as a normal module dependency (`github.com/rennf93/guard-core-go/v4 v4.0.4`); no `replace` directive is used or needed. For cross-repo work on the core itself, add a temporary local `replace` line in your own checkout and drop it before committing.

Integration tests run against real Redis:

```
REDIS_HOST=127.0.0.1 go test -tags integration ./...
```

## License

MIT
