package scheduler

import (
	"context"
	"time"

	"gitlab.com/Cacophony/Worker/plugins"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

type Scheduler struct {
	logger *zap.Logger
}

func NewScheduler(
	logger *zap.Logger,
) *Scheduler {
	return &Scheduler{
		logger: logger,
	}
}

func (s *Scheduler) Start() {
	var err error

	for {
		for _, plugin := range plugins.PluginList {

			run := common.NewRun(plugin.Name())

			logger := s.logger.With(
				zap.String("plugin", plugin.Name()),
				zap.String("launch", run.Launch.String()),
			)

			run.WithContext(context.Background())
			run.WithLogger(logger)

			err = plugin.Run(run)
			if err != nil {
				logger.Error("run execution failed",
					zap.Error(err),
				)
			}

		}

		time.Sleep(60 * time.Second)
	}
}
