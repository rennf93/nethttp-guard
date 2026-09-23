//go:build integration

package nethttp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rennf93/guard-core-go/v4/guardcore"
)

func newIntegrationMiddleware(t *testing.T) (func(http.Handler) http.Handler, *guardcore.Engine) {
	t.Helper()
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		t.Skip("REDIS_HOST not set")
	}
	cfg := guardcore.DefaultSecurityConfig()
	cfg.EnableRedis = true
	cfg.RedisURL = "redis://" + host + ":6379"
	cfg.RedisPrefix = "guard_core_nethttp_test:"
	cfg.RedisFailOpen = false
	engine, err := guardcore.NewEngine(cfg)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if err := engine.Initialize(); err != nil {
		t.Fatalf("engine initialize: %v", err)
	}
	wrap, err := New(engine)
	if err != nil {
		t.Fatalf("middleware: %v", err)
	}
	t.Cleanup(func() {
		_, _ = engine.Redis.DeletePattern("banned_ips:*")
		_, _ = engine.Redis.DeletePattern("rate_limit:rate:*")
		_ = engine.Close()
	})
	return wrap, engine
}

func TestIntegrationMiddlewareBlocksBannedIP(t *testing.T) {
	wrap, engine := newIntegrationMiddleware(t)
	bannedIP := "203.0.113.60"
	applied, err := engine.Ban.Ban(bannedIP, 120, "integration-test")
	if err != nil || !applied {
		t.Fatalf("ban not applied: %v %v", applied, err)
	}
	blocked := httptest.NewRequest("GET", "/api", nil)
	blocked.RemoteAddr = bannedIP + ":4711"
	rec := httptest.NewRecorder()
	wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("blocked request must not reach the handler")
	})).ServeHTTP(rec, blocked)
	if rec.Code != 403 || rec.Body.String() != guardcore.IPBanBlockedMessage {
		t.Fatalf("banned IP must be blocked with 403 %q, got %d %q", guardcore.IPBanBlockedMessage, rec.Code, rec.Body.String())
	}
}

func TestIntegrationMiddlewareStartupWiringAcrossEngines(t *testing.T) {
	_, first := newIntegrationMiddleware(t)
	bannedIP := "203.0.113.61"
	applied, err := first.Ban.Ban(bannedIP, 120, "integration-test")
	if err != nil || !applied {
		t.Fatalf("ban not applied: %v %v", applied, err)
	}
	secondWrap, _ := newIntegrationMiddleware(t)
	blocked := httptest.NewRequest("GET", "/api", nil)
	blocked.RemoteAddr = bannedIP + ":4711"
	rec := httptest.NewRecorder()
	secondWrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("blocked request must not reach the handler")
	})).ServeHTTP(rec, blocked)
	if rec.Code != 403 || rec.Body.String() != guardcore.IPBanBlockedMessage {
		t.Fatalf("second engine must see the redis-backed ban after startup wiring, got %d %q", rec.Code, rec.Body.String())
	}
	allowed := httptest.NewRequest("GET", "/api", nil)
	allowed.RemoteAddr = "203.0.113.62:4711"
	rec = httptest.NewRecorder()
	secondWrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, allowed)
	if rec.Code != 200 {
		t.Fatalf("unbanned IP must pass through the second engine, got %d", rec.Code)
	}
}
