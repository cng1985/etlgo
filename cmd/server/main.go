package main

import (
	"net/http"

	"etlgo/internal/config"
	"etlgo/internal/handler"
	"etlgo/internal/seed"
	"etlgo/internal/server"
	"etlgo/internal/service"
	"etlgo/internal/store"

	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			config.New,
			store.New,
			service.NewSourceService,
			service.NewPipelineService,
			service.NewDatasetService,
			handler.New,
			server.New,
		),
		fx.Invoke(seed.Apply),
		fx.Invoke(service.StartScheduler),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
