# Usage

## Constructor

```go
func New(engine *guardcore.Engine, opts ...Option) (func(http.Handler) http.Handler, error)
```

`New` returns the standard `net/http` middleware decorator. It rejects a nil
engine with an error; invalid option values are silently ignored so a bad
flag can never weaken security.

## Options

| Option | Default | Purpose |
|---|---|---|
| `WithMaxBodyBytes(int64)` | `DefaultMaxBodyBytes` (262144) | Body prefix handed to the engine for inspection |
| `WithLogger(*log.Logger)` | none | Receive fail-closed diagnostics (prefix `guardcore nethttp: ...`) |

```go
guard, err := nethttp.New(engine,
    nethttp.WithMaxBodyBytes(64*1024),
    nethttp.WithLogger(log.New(os.Stderr, "", log.LstdFlags)),
)
```

## Route IDs

Per-route configuration lives in the engine's `RouteRegistry`. Attach a route
ID to the request context before the guard runs by registering the ID
middleware first in the chain:

```go
engine.Routes.Register("admin", func(rc *guardcore.RouteConfig) {
    rc.RequiredHeaders = guardcore.RequiredHeaders{
        {Name: "X-Admin-Token", Value: "secret"},
    }
})

routeID := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.HasPrefix(r.URL.Path, "/admin/") {
            r = r.WithContext(nethttp.WithRouteID(r.Context(), "admin"))
        }
        next.ServeHTTP(w, r)
    })
}

handler := routeID(guard(mux)) // route middleware runs before the guard
```

## Verdicts

When the engine returns a block verdict, the middleware writes the verdict
status code, headers, and body, and never calls the wrapped handler:

| Situation | Status | Body |
|---|---|---|
| Banned IP | 403 | `IP address banned` |
| Suspicious content | 400 | `Suspicious activity detected` |
| Rate limit exceeded | 429 | `Too many requests` |

Bodies can be overridden globally through `SecurityConfig.CustomErrorResponses`.

## Fail-closed behavior

If the engine check panics, the middleware recovers, logs through the
`WithLogger` sink, and responds `500 Security check failed` rather than
letting the request through.
