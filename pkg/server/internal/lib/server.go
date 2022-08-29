package lib

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/openelb/openelb/pkg/server/options"
)

type server struct {
	handler http.Handler
	options options.Options
}

func NewHTTPServer(routers []Router, options options.Options) *server {
	httpRouter := chi.NewRouter()

	for _, endpoint := range routers {
		endpoint.Register(httpRouter)
	}

	return &server{
		handler: cors.New(cors.Options{
			AllowedOrigins: []string{"http://localhost:3000"},
			AllowCredentials: true,
			AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			AllowedHeaders: []string{"Accept", "Content-Type"},
		}).Handler(httpRouter),
		options: options,
	}
}

func (s *server) ListenAndServe(stopCh <-chan struct{}) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.options.Port),
		Handler: s.handler,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-stopCh:
		ctx, cancel :=
			context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return ctx.Err()
	}
}
