package lib

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/openelb/openelb/pkg/server/options"
)

type server struct {
	handler http.Handler
	options options.Options
}

func NewHTTPServer(endpoints []Endpoints, options options.Options) *server {
	router := chi.NewRouter()

	for _, endpoint := range endpoints {
		endpoint.Register(router)
	}

	return &server{
		handler: cors.New(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
			AllowedHeaders: []string{"Accept", "Content-Type"},
		}).Handler(router),
		options: options,
	}
}

func (s *server) ListenAndServe() error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.options.Port),
		Handler: s.handler,
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	errCh := make(chan error)
	go func() {
		err := server.ListenAndServe()
		select {
		case errCh <- err:
		case <-ctx.Done():
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, shutdownCancel :=
			context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
		return shutdownCtx.Err()
	}
}
