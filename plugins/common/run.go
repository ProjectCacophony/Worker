package common

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Run struct {
	Plugin string
	Launch time.Time

	ctx    context.Context
	logger *zap.Logger
}

func NewRun(pluginName string) *Run {
	return &Run{
		Plugin: pluginName,
		Launch: time.Now(),
	}
}
