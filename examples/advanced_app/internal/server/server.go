// Package server assembles the guarded handler chain:
//
//	routeID middleware (attaches guard route IDs) -> guard -> mux
//
// Order matters: the route-ID middleware must run before the guard so the
// engine can resolve per-route config.
package server

import (
	"net/http"
	"strings"

	guardcore "github.com/rennf93/guard-core-go/guardcore"
	nethttp "github.com/rennf93/nethttp-guard"
	"github.com/rennf93/nethttp-guard/examples/advanced_app/internal/routes"
)

// New builds the guarded root handler.
func New(engine *guardcore.Engine, routeIDs map[string]string) (http.Handler, error) {
	guard, err := nethttp.New(engine)
	if err != nil {
		return nil, err
	}

	return routeIDMiddleware(routeIDs)(guard(routes.Mux(&routes.App{Engine: engine}))), nil
}

// routeIDMiddleware maps request paths to route IDs registered on the
// engine's RouteRegistry. Longest matching prefix wins.
func routeIDMiddleware(routeIDs map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bestLen, bestID := 0, ""
			for prefix, id := range routeIDs {
				if strings.HasPrefix(r.URL.Path, prefix) && len(prefix) > bestLen {
					bestLen, bestID = len(prefix), id
				}
			}
			if bestID != "" {
				r = r.WithContext(nethttp.WithRouteID(r.Context(), bestID))
			}
			next.ServeHTTP(w, r)
		})
	}
}
