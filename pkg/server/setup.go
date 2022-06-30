package server

import (
	"github.com/openelb/openelb/pkg/server/internal/lib"
	"github.com/openelb/openelb/pkg/server/options"
)

func SetupHTTPServer(opts *options.Options) error {
	server := lib.NewHTTPServer([]lib.Endpoints{}, *opts)
	return server.ListenAndServe()
}
