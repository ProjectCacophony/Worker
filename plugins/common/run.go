package common

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Run struct {
	Launch time.Time

	ctx    context.Context
	logger *zap.Logger
}

func NewRun() *Run {
	return &Run{
		Launch: time.Now(),
	}
}
