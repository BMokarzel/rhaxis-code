package pkg_http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server é o contrato que cada módulo HTTP deve implementar.
// A camada cmd não precisa conhecer detalhes internos do módulo —
// só obter um Server e entregá-lo ao Run.
type Server interface {
	// Name identifica o módulo (usado em logs).
	Name() string
	// Addr é o endereço de bind, ex.: ":8081".
	Addr() string
	// Routes devolve o handler raiz (normalmente um *chi.Mux).
	Routes() http.Handler
}

// Timeouts agrupa os tempos aplicados ao http.Server.
type Timeouts struct {
	Read     time.Duration
	Write    time.Duration
	Idle     time.Duration
	Shutdown time.Duration
}

// Run inicia o Server e bloqueia até que ctx seja cancelado ou
// o servidor falhe. Faz graceful shutdown respeitando t.Shutdown.
func Run(ctx context.Context, s Server, t Timeouts) error {
	srv := &http.Server{
		Addr:         s.Addr(),
		Handler:      s.Routes(),
		ReadTimeout:  t.Read,
		WriteTimeout: t.Write,
		IdleTimeout:  t.Idle,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("%s: listen: %w", s.Name(), err)
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), t.Shutdown)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("%s: shutdown: %w", s.Name(), err)
		}
		return nil
	}
}

// NewRouter devolve um chi.Mux com middlewares padrão.
// Cada módulo monta suas rotas em cima deste mux.
func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return r
}
