package nethttp

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/rennf93/guard-core-go/guardcore"
)

const failClosedMessage = "Security check failed"

type middleware struct {
	engine   *guardcore.Engine
	maxBytes int64
	logger   *log.Logger
}

type Option func(*middleware)

func WithMaxBodyBytes(maxBodyBytes int64) Option {
	return func(m *middleware) {
		if maxBodyBytes > 0 {
			m.maxBytes = maxBodyBytes
		}
	}
}

func WithLogger(logger *log.Logger) Option {
	return func(m *middleware) {
		if logger != nil {
			m.logger = logger
		}
	}
}

func New(engine *guardcore.Engine, opts ...Option) (func(http.Handler) http.Handler, error) {
	if engine == nil {
		return nil, errors.New("engine must not be nil")
	}
	m := &middleware{engine: engine, maxBytes: DefaultMaxBodyBytes, logger: log.Default()}
	for _, opt := range opts {
		opt(m)
	}
	return m.wrap, nil
}

func (m *middleware) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := newRequestShim(r, m.maxBytes)
		verdict, err := m.check(req)
		if err != nil {
			m.logger.Printf("guardcore nethttp: engine malfunction, failing closed: %v", err)
			applyResponse(w, m.engine.CreateErrorResponse(500, failClosedMessage))
			return
		}
		if verdict != nil {
			applyResponse(w, verdict)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *middleware) check(req guardcore.Request) (verdict *guardcore.Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("engine panic: %v", r)
		}
	}()
	return m.engine.Check(req), nil
}

func applyResponse(w http.ResponseWriter, response *guardcore.Response) {
	for name, value := range response.Headers {
		w.Header().Set(name, value)
	}
	w.WriteHeader(response.StatusCode)
	if len(response.Body) > 0 {
		_, _ = w.Write(response.Body)
	}
}
