package service

import "go.uber.org/fx"

var SourceModule = fx.Module("source",
	fx.Provide(NewSourceService),
)

var PipelineModule = fx.Module("pipeline",
	fx.Provide(NewPipelineService),
	fx.Invoke(StartScheduler),
)

var DatasetModule = fx.Module("dataset",
	fx.Provide(NewDatasetService),
)

var Module = fx.Module("service",
	SourceModule,
	PipelineModule,
	DatasetModule,
)
