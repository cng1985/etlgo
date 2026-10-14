package server

import (
	"net/http"

	"go.uber.org/fx"
)

var Module = fx.Module("server",
	fx.Provide(New),
	fx.Invoke(func(*http.Server) {}),
)
