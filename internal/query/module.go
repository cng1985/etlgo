package query

import "go.uber.org/fx"

var Module = fx.Module("query",
	fx.Provide(New),
)
