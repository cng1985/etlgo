package seed

import "go.uber.org/fx"

var Module = fx.Module("seed",
	fx.Invoke(Apply),
)
