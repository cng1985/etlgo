package app

import (
	"etlgo/internal/config"
	"etlgo/internal/handler"
	"etlgo/internal/query"
	"etlgo/internal/seed"
	"etlgo/internal/server"
	"etlgo/internal/service"
	"etlgo/internal/store"

	"go.uber.org/fx"
)

var Module = fx.Module("etl",
	config.Module,
	store.Module,
	service.Module,
	query.Module,
	seed.Module,
	handler.Module,
	server.Module,
)
